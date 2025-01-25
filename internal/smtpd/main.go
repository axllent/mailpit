// Package smtpd is the SMTP daemon
package smtpd

import (
	"bytes"
	"fmt"
	"net"
	"net/mail"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/stats"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/lithammer/shortuuid/v4"
)

var (
	// DisableReverseDNS allows rDNS to be disabled
	DisableReverseDNS bool

	warningResponse = regexp.MustCompile(`^4\d\d `)
	errorResponse   = regexp.MustCompile(`^5\d\d `)
)

// MailHandler handles the incoming message to store in the database
func mailHandler(origin net.Addr, from string, to []string, data []byte) (string, error) {
	return SaveToDatabase(origin, from, to, data)
}

// SaveToDatabase will attempt to save a message to the database
func SaveToDatabase(origin net.Addr, from string, to []string, data []byte) (string, error) {
	if !config.SMTPStrictRFCHeaders {
		// replace all <CR><CR><LF> (\r\r\n) with <CR><LF> (\r\n)
		// @see https://github.com/axllent/mailpit/issues/87 & https://github.com/axllent/mailpit/issues/153
		data = bytes.ReplaceAll(data, []byte("\r\r\n"), []byte("\r\n"))
	}

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		logger.Log().Warnf("[smtpd] error parsing message: %s", err.Error())
		stats.LogSMTPRejected()
		return "", err
	}

	// check / set the Return-Path based on SMTP from
	returnPath := strings.Trim(msg.Header.Get("Return-Path"), "<>")
	if returnPath != from {
		if returnPath != "" {
			// replace Return-Path
			re := regexp.MustCompile(`(?i)(^|\n)(Return\-Path: .*\n)`)
			replaced := false
			data = re.ReplaceAllFunc(data, func(r []byte) []byte {
				if replaced {
					return r
				}
				replaced = true // only replace first occurrence

				return re.ReplaceAll(r, []byte("${1}Return-Path: <"+from+">\r\n"))
			})
		} else {
			// add Return-Path
			data = append([]byte("Return-Path: <"+from+">\r\n"), data...)
		}
	}

	messageID := strings.Trim(msg.Header.Get("Message-Id"), "<>")

	// add a message ID if not set
	if messageID == "" {
		// generate unique ID
		messageID = shortuuid.New() + "@mailpit"
		// add unique ID
		data = append([]byte("Message-Id: <"+messageID+">\r\n"), data...)
	} else if config.IgnoreDuplicateIDs {
		if storage.MessageIDExists(messageID) {
			logger.Log().Debugf("[smtpd] duplicate message found, ignoring %s", messageID)
			stats.LogSMTPIgnored()
			return "", nil
		}
	}

	// if enabled, this may conditionally relay the email through to the preconfigured smtp server
	autoRelayMessage(from, to, &data)

	// if enabled, this will forward a copy to preconfigured addresses
	autoForwardMessage(from, &data)

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
		if hasBccHeader {
			// email already has Bcc header, add to existing addresses
			re := regexp.MustCompile(`(?i)(^|\n)(Bcc: )`)
			replaced := false
			data = re.ReplaceAllFunc(data, func(r []byte) []byte {
				if replaced {
					return r
				}
				replaced = true // only replace first occurrence

				return re.ReplaceAll(r, []byte("${1}Bcc: "+strings.Join(missingAddresses, ", ")+", "))
			})

		} else {
			// prepend new Bcc header
			bcc := []byte(fmt.Sprintf("Bcc: %s\r\n", strings.Join(missingAddresses, ", ")))
			data = append(bcc, data...)
		}

		logger.Log().Debugf("[smtpd] added missing addresses to Bcc header: %s", strings.Join(missingAddresses, ", "))
	}

	id, err := storage.Store(&data)
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
		Addr:              addr,
		MsgIDHandler:      handler,
		HandlerRcpt:       handlerRcpt,
		AppName:           "Mailpit",
		Hostname:          "",
		AuthHandler:       nil,
		AuthRequired:      false,
		MaxRecipients:     config.SMTPMaxRecipients,
		DisableReverseDNS: DisableReverseDNS,
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

	if config.Label != "" {
		srv.AppName = fmt.Sprintf("Mailpit (%s)", config.Label)
	}

	if config.SMTPAuthAllowInsecure {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
	}

	if auth.SMTPCredentials != nil {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
		srv.AuthHandler = authHandler
		srv.AuthRequired = true
	} else if config.SMTPAuthAcceptAny {
		srv.AuthMechs = map[string]bool{"CRAM-MD5": false, "PLAIN": true, "LOGIN": true}
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
			if config.SMTPRequireSTARTTLS {
				smtpType = "STARTTLS required"
			} else if config.SMTPRequireTLS {
				smtpType = "SSL/TLS required"
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

// Returns a list of all lowercased emails found in To, Cc and Bcc,
// as well as whether there is a Bcc field
func scanAddressesInHeader(h mail.Header) (map[string]bool, bool) {
	emails := make(map[string]bool)
	hasBccHeader := false

	if recipients, err := h.AddressList("To"); err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}
	}

	if recipients, err := h.AddressList("Cc"); err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}
	}

	recipients, err := h.AddressList("Bcc")
	if err == nil {
		for _, r := range recipients {
			emails[strings.ToLower(r.Address)] = true
		}

		hasBccHeader = true
	}

	return emails, hasBccHeader
}
