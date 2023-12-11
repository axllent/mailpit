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
	"github.com/axllent/mailpit/internal/storage"
	"github.com/google/uuid"
	"github.com/mhale/smtpd"
)

func mailHandler(origin net.Addr, from string, to []string, data []byte) error {
	if !config.SMTPStrictRFCHeaders {
		// replace all <CR><CR><LF> (\r\r\n) with <CR><LF> (\r\n)
		// @see https://github.com/axllent/mailpit/issues/87 & https://github.com/axllent/mailpit/issues/153
		data = bytes.ReplaceAll(data, []byte("\r\r\n"), []byte("\r\n"))
	}

	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		logger.Log().Errorf("[smtpd] error parsing message: %s", err.Error())

		return err
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
		messageID = uuid.New().String() + "@mailpit"
		// add unique ID
		data = append([]byte("Message-Id: <"+messageID+">\r\n"), data...)
	} else if config.IgnoreDuplicateIDs {
		if storage.MessageIDExists(messageID) {
			logger.Log().Debugf("[smtpd] duplicate message found, ignoring %s", messageID)
			return nil
		}
	}

	// if enabled, this will route the email 1:1 through to the preconfigured smtp server
	if config.SMTPRelayAllIncoming {
		if err := Send(from, to, data); err != nil {
			logger.Log().Warnf("[smtp] error relaying message: %s", err.Error())
		} else {
			logger.Log().Debugf("[smtp] relayed message from %s via %s:%d", from, config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)
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

	_, err = storage.Store(data)
	if err != nil {
		logger.Log().Errorf("[db] error storing message: %s", err.Error())

		return err
	}

	subject := msg.Header.Get("Subject")
	logger.Log().Debugf("[smtpd] received (%s) from:%s subject:%q", cleanIP(origin), from, subject)

	return nil
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

// Listen starts the SMTPD server
func Listen() error {
	if config.SMTPAuthAllowInsecure {
		if auth.SMTPCredentials != nil {
			logger.Log().Info("[smtpd] enabling login auth (insecure)")
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtpd] enabling all auth (insecure)")
		}
	} else {
		if auth.SMTPCredentials != nil {
			logger.Log().Info("[smtpd] enabling login auth (TLS)")
		} else if config.SMTPAuthAcceptAny {
			logger.Log().Info("[smtpd] enabling any auth (TLS)")
		}
	}

	logger.Log().Infof("[smtpd] starting on %s", config.SMTPListen)

	return listenAndServe(config.SMTPListen, mailHandler, authHandler)
}

func listenAndServe(addr string, handler smtpd.Handler, authHandler smtpd.AuthHandler) error {
	srv := &smtpd.Server{
		Addr:          addr,
		Handler:       handler,
		Appname:       "Mailpit",
		Hostname:      "",
		AuthHandler:   nil,
		AuthRequired:  false,
		MaxRecipients: config.SMTPMaxRecipients,
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
		if err := srv.ConfigureTLS(config.SMTPTLSCert, config.SMTPTLSKey); err != nil {
			return err
		}
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
