// Package smtpd is the SMTP daemon
package smtpd

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/shortuuid"
	"github.com/axllent/mailpit/internal/stats"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/pkg/errors"
)

var (
	// DisableReverseDNS allows rDNS to be disabled
	DisableReverseDNS bool

	warningResponse = regexp.MustCompile(`^4\d\d `)
	errorResponse   = regexp.MustCompile(`^5\d\d `)
)

// MailHandler handles the incoming message to store in the database
func mailHandler(origin net.Addr, from string, to []string, data []byte, smtpUser *string) (string, error) {
	return SaveToDatabase(origin, from, to, data, smtpUser)
}

// SaveToDatabase will attempt to save a message to the database
func SaveToDatabase(origin net.Addr, from string, to []string, data []byte, smtpUser *string) (string, error) {
	if config.SMTPStrictRFCHeaders {
		// reject headers with line breaks which are not <CR><LF>: <CR><CR><LF>, <LF> or <CR> on its own
		if _, invalid := tools.NormaliseHeaderLineBreaks(data); invalid {
			logger.Log().Warn("[smtpd] rejected message, the headers contain line breaks which are not <CR><LF>")
			stats.LogSMTPRejected()

			return "", errors.New("550 5.6.0 Message headers must use <CR><LF> line breaks")
		}
	} else {
		// Fix the line breaks of the headers: <CR><CR><LF> (@see https://github.com/axllent/mailpit/issues/87
		// & https://github.com/axllent/mailpit/issues/153), and <LF> or <CR> on its own, so the headers are the
		// same to everything which reads the message. The body is stored as it was received (it used to
		// have its <CR><CR><LF> changed as well), and the line breaks are made <CR><LF> when it is sent on.
		data, _ = tools.NormaliseHeaderLineBreaks(data)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		logger.Log().Warnf("[smtpd] error parsing message: %s", err.Error())
		stats.LogSMTPRejected()

		return "", invalidHeadersError(err)
	}

	// check / set the Return-Path based on SMTP from
	returnPath := strings.Trim(msg.Header.Get("Return-Path"), "<>")
	if returnPath != from {
		data, err = tools.SetMessageHeader(data, "Return-Path", "<"+from+">")
		if err != nil {
			logger.Log().Warnf("[smtpd] error setting the Return-Path: %s", err.Error())
			stats.LogSMTPRejected()

			return "", invalidHeadersError(err)
		}
	}

	messageID := strings.Trim(msg.Header.Get("Message-ID"), "<>")

	// add a message ID if not set
	if messageID == "" {
		// generate unique ID
		messageID = shortuuid.New() + "@mailpit"
		// add unique ID
		data = append([]byte("Message-ID: <"+messageID+">\r\n"), data...)
	} else if config.IgnoreDuplicateIDs {
		if storage.MessageIDExists(messageID) {
			logger.Log().Debugf("[smtpd] duplicate message found, ignoring %s", messageID)
			stats.LogSMTPIgnored()
			return "", nil
		}
	}

	// if enabled, this may conditionally relay the email through to the preconfigured smtp server
	if relayErr := autoRelayMessage(from, to, &data); relayErr != nil {
		logger.Log().Error(relayErr.Error())

		if config.SMTPRelayConfig.ForwardSMTPErrors {
			for {
				unwrappedErr := errors.Unwrap(relayErr)
				if unwrappedErr == nil {
					break
				}
				relayErr = unwrappedErr
			}
			return "", relayErr
		}
	}

	// if enabled, this will forward a copy to preconfigured addresses
	if forwardErr := autoForwardMessage(from, &data); forwardErr != nil {
		logger.Log().Error(forwardErr.Error())

		if config.SMTPForwardConfig.ForwardSMTPErrors {
			for {
				unwrappedErr := errors.Unwrap(forwardErr)
				if unwrappedErr == nil {
					break
				}
				forwardErr = unwrappedErr
			}
			return "", forwardErr
		}
	}

	// build array of all addresses in the header to compare to the []to array
	emails, hasBccHeader := scanAddressesInHeader(msg.Header)

	missingAddresses := []string{}
	for _, a := range to {
		// loop through passed email addresses to check if they are in the headers
		if _, err := mail.ParseAddress(a); err == nil {
			_, ok := emails[strings.ToLower(a)]
			if !ok {
				missingAddresses = append(missingAddresses, a)
			}
		} else {
			logger.Log().Warnf("[smtpd] ignoring invalid email address: %s", a)
		}
	}

	// add missing email addresses to Bcc (eg: Laravel doesn't include these in the headers)
	if len(missingAddresses) > 0 {
		data, err = addBccAddresses(data, msg.Header, hasBccHeader, missingAddresses)
		if err != nil {
			logger.Log().Warnf("[smtpd] error adding the Bcc recipients: %s", err.Error())
			stats.LogSMTPRejected()

			return "", invalidHeadersError(err)
		}

		logger.Log().Debugf("[smtpd] added missing addresses to Bcc header: %s", strings.Join(missingAddresses, ", "))
	}

	id, err := storeWithinBudget(&data, smtpUser)
	if err != nil {
		logger.Log().Errorf("[db] error storing message: %s", err.Error())
		return "", err
	}

	stats.LogSMTPAccepted(len(data))

	data = nil // avoid memory leaks

	subject := msg.Header.Get("Subject")
	logger.Log().Debugf("[smtpd] received (%s) from:%s subject:%q", cleanIP(origin), from, subject)

	return id, err
}

// ingestBudgetBytes is how many bytes of messages are parsed, indexed, compressed and stored at the same
// time. Doing that takes about 7 times the size of a plain text message in memory and about 21 times that of
// an HTML message made of many small tags, and nothing else limited how many were being done at once.
// Measured with 4 HTML messages of 20 MiB arriving together, the peak memory of the process was 1774 MiB
// without a limit (4.5 seconds) and 895 MiB with this one (5.8 seconds); 8 at once were enough to run a
// machine with 15 GB out of memory. A message which is larger than the budget is stored on its own, and a
// small one only waits when the budget is full.
const ingestBudgetBytes = 20 << 20

var (
	// ingestBudget is shared by every message which is received. It is a variable for the tests.
	ingestBudget = tools.NewBudget(ingestBudgetBytes)

	// ingestWait is how long a message waits for its turn before the sender is told to try again later. It
	// is a variable for the tests.
	ingestWait = 2 * time.Minute
)

// storeWithinBudget stores the message when there is room for it in the budget, waiting for its turn if
// there is not. The sender is told to try again later (a temporary failure) if it waits too long.
func storeWithinBudget(data *[]byte, smtpUser *string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ingestWait)
	defer cancel()

	release, err := ingestBudget.Acquire(ctx, int64(len(*data)))
	if err != nil {
		logger.Log().Warnf("[smtpd] message of %d bytes not stored, as too many large messages are being processed", len(*data))

		return "", errors.New("451 4.3.2 Too many large messages are being processed, please try again later")
	}
	defer release()

	return storage.Store(data, smtpUser)
}

// maxReasonLength is the longest reason (from the error of the parser) which is sent to the client.
const maxReasonLength = 120

// invalidHeadersError is the error for a message whose headers can not be read. It is a permanent
// failure (550) as sending the same message again can not work. Without a code it was sent to the
// client as a temporary failure (451), which a client retries.
//
// The reason is the parser's error, which can include text from the message, so it is cut down to
// printable ASCII on a single line before it is sent in the response.
func invalidHeadersError(err error) error {
	reason := strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\r' || r == '\n' || r < 0x20 || r == 0x7f:
			return ' '
		case r > 0x7e:
			return '?'
		}

		return r
	}, err.Error())

	reason = strings.Join(strings.Fields(reason), " ")
	if reason == "" {
		reason = "unknown error"
	}
	if len(reason) > maxReasonLength {
		reason = reason[:maxReasonLength] + "..."
	}

	return errors.New("550 5.6.0 Message headers could not be parsed: " + reason)
}

func authHandler(remoteAddr net.Addr, mechanism string, username []byte, password []byte, _ []byte) (bool, error) {
	allow := auth.SMTPCredentials.Match(string(username), string(password))
	if allow {
		logger.Log().Debugf("[smtpd] allow %s login:%q from:%s", mechanism, string(username), cleanIP(remoteAddr))
	} else {
		logger.Log().Warnf("[smtpd] deny %s login:%q from:%s", mechanism, string(username), cleanIP(remoteAddr))
	}

	return allow, nil
}

// Allow any username and password
func authHandlerAny(remoteAddr net.Addr, mechanism string, username []byte, _ []byte, _ []byte) (bool, error) {
	logger.Log().Debugf("[smtpd] allow %s login %q from %s", mechanism, string(username), cleanIP(remoteAddr))

	return true, nil
}

// HandlerRcpt used to optionally restrict recipients based on `--smtp-allowed-recipients`
func handlerRcpt(remoteAddr net.Addr, from string, to string) bool {
	if config.SMTPAllowedRecipientsRegexp == nil {
		return true
	}

	result := config.SMTPAllowedRecipientsRegexp.MatchString(to)

	if !result {
		logger.Log().Warnf("[smtpd] rejected message to %s from %s (%s)", to, from, cleanIP(remoteAddr))
		stats.LogSMTPRejected()
	}

	return result
}

// Listen starts the SMTPD server
func Listen() error {
	if config.SMTPAuthAllowInsecure {
		if auth.SMTPCredentials != nil {
			logger.Log().Info("[smtpd] enabling login authentication (insecure)")
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtpd] enabling any authentication (insecure)")
		}
	} else {
		if auth.SMTPCredentials != nil {
			logger.Log().Info("[smtpd] enabling login authentication")
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtpd] enabling any authentication")
		}
	}

	return listenAndServe(config.SMTPListen, mailHandler, authHandler)
}

// Translate the smtpd verb from READ/WRITE
func verbLogTranslator(verb string) string {
	if verb == "READ" {
		return "received"
	}

	return "response"
}

func listenAndServe(addr string, handler MsgIDHandler, authHandler AuthHandler) error {

	socketAddr, perm, isSocket := tools.UnixSocket(addr)

	Debug = true // to enable Mailpit logging
	srv := &Server{
		Addr:                     addr,
		MsgIDHandler:             handler,
		HandlerRcpt:              handlerRcpt,
		AppName:                  "Mailpit",
		Hostname:                 "",
		AuthHandler:              nil,
		AuthRequired:             false,
		MaxRecipients:            config.SMTPMaxRecipients,
		IgnoreRejectedRecipients: config.SMTPIgnoreRejectedRecipients,
		DisableReverseDNS:        DisableReverseDNS,
		LogRead: func(remoteIP, verb, line string) {
			logger.Log().Debugf("[smtpd] %s (%s) %s", verbLogTranslator(verb), remoteIP, line)
		},
		LogWrite: func(remoteIP, verb, line string) {
			if warningResponse.MatchString(line) {
				logger.Log().Warnf("[smtpd] %s (%s) %s", verbLogTranslator(verb), remoteIP, line)
				websockets.BroadCastClientError("warning", "smtpd", remoteIP, line)
			} else if errorResponse.MatchString(line) {
				logger.Log().Errorf("[smtpd] %s (%s) %s", verbLogTranslator(verb), remoteIP, line)
				websockets.BroadCastClientError("error", "smtpd", remoteIP, line)
			} else {
				logger.Log().Debugf("[smtpd] %s (%s) %s", verbLogTranslator(verb), remoteIP, line)
			}
		},
	}

	if config.MaxMessageSize > 0 {
		srv.MaxSize = config.MaxMessageSize * 1024 * 1024
	}

	if config.Label != "" {
		srv.AppName = fmt.Sprintf("Mailpit (%s)", config.Label)
	}

	if config.SMTPAuthAllowInsecure {
		srv.AuthMechs = map[string]bool{
			"CRAM-MD5": false,
			"PLAIN":    true,
			"LOGIN":    true,
		}
	}

	if auth.SMTPCredentials != nil {
		srv.AuthMechs = map[string]bool{
			"CRAM-MD5": false,
			"PLAIN":    true,
			"LOGIN":    true,
		}
		srv.AuthHandler = authHandler
		srv.AuthRequired = true
	} else if config.SMTPAuthAcceptAny {
		srv.AuthMechs = map[string]bool{
			"CRAM-MD5": false,
			"PLAIN":    true,
			"LOGIN":    true,
		}
		srv.AuthHandler = authHandlerAny
	}

	if config.SMTPTLSCert != "" {
		srv.TLSRequired = config.SMTPRequireSTARTTLS
		srv.TLSListener = config.SMTPRequireTLS // if true overrules srv.TLSRequired
		if err := srv.ConfigureTLS(config.SMTPTLSCert, config.SMTPTLSKey); err != nil {
			return err
		}
	}

	if isSocket {
		srv.Addr = socketAddr
		srv.Protocol = "unix"
		srv.SocketPerm = perm

		if err := tools.PrepareSocket(srv.Addr); err != nil {
			storage.Close()
			return err
		}

		// delete the Unix socket file on exit
		storage.AddTempFile(srv.Addr)

		logger.Log().Infof("[smtpd] starting on %s", config.SMTPListen)
	} else {
		smtpType := "no encryption"

		if config.SMTPTLSCert != "" {
			if config.SMTPRequireTLS {
				smtpType = "SSL/TLS required"
			} else if config.SMTPRequireSTARTTLS {
				smtpType = "STARTTLS required"
			} else {
				smtpType = "STARTTLS optional"
				if !config.SMTPAuthAllowInsecure && auth.SMTPCredentials != nil {
					smtpType = "STARTTLS required"
				}
			}
		}

		logger.Log().Infof("[smtpd] starting on %s (%s)", config.SMTPListen, smtpType)
	}

	return srv.ListenAndServe()
}

func cleanIP(i net.Addr) string {
	parts := strings.Split(i.String(), ":")

	return parts[0]
}

// addBccAddresses sets the Bcc header to include the addresses, appending them
// to any existing Bcc value.
func addBccAddresses(data []byte, h mail.Header, hasBccHeader bool, addresses []string) ([]byte, error) {
	bccVal := strings.Join(addresses, ", ")
	if hasBccHeader {
		bccVal = h.Get("Bcc") + ", " + bccVal
	}

	return tools.SetMessageHeader(data, "Bcc", bccVal)
}

// Returns a list of all lowercased emails found in To, Cc and Bcc,
// as well as whether there is a Bcc field
func scanAddressesInHeader(h mail.Header) (map[string]bool, bool) {
	emails := make(map[string]bool)
	hasBccHeader := false

	if recipients, err := tools.AddressList(h, "To"); err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}
	}

	if recipients, err := tools.AddressList(h, "Cc"); err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}
	}

	recipients, err := tools.AddressList(h, "Bcc")
	if err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}

		hasBccHeader = true
	}

	return emails, hasBccHeader
}
