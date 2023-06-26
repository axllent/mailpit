package smtpd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
)

func allowedRecipients(to []string) []string {
	if config.SMTPRelayConfig.RecipientAllowlistRegexp == nil {
		return to
	}

	var ar []string

	for _, recipient := range to {
		address, err := mail.ParseAddress(recipient)

		if err != nil {
			logger.Log().Warnf("ignoring invalid email address: %s", recipient)
			continue
		}

		if !config.SMTPRelayConfig.RecipientAllowlistRegexp.MatchString(address.Address) {
			logger.Log().Debugf("[smtp] not allowed to relay to %s: does not match the allowlist %s", recipient, config.SMTPRelayConfig.RecipientAllowlist)
		} else {
			ar = append(ar, recipient)
		}
	}

	return ar
}

// Send will connect to a pre-configured SMTP server and send a message to one or more recipients.
func Send(from string, to []string, msg []byte) error {
	recipients := allowedRecipients(to)

	if len(recipients) == 0 {
		return errors.New("no valid recipients")
	}

	addr := fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %s", addr, err.Error())
	}

	defer c.Close()

	if config.SMTPRelayConfig.STARTTLS {
		conf := &tls.Config{ServerName: config.SMTPRelayConfig.Host}

		conf.InsecureSkipVerify = config.SMTPRelayConfig.AllowInsecure

		if err = c.StartTLS(conf); err != nil {
			return fmt.Errorf("error creating StartTLS config: %s", err.Error())
		}
	}

	var a smtp.Auth

	if config.SMTPRelayConfig.Auth == "plain" {
		a = smtp.PlainAuth("", config.SMTPRelayConfig.Username, config.SMTPRelayConfig.Password, config.SMTPRelayConfig.Host)
	}

	if config.SMTPRelayConfig.Auth == "login" {
		a = LoginAuth(config.SMTPRelayConfig.Username, config.SMTPRelayConfig.Password)
	}

	if config.SMTPRelayConfig.Auth == "cram-md5" {
		a = smtp.CRAMMD5Auth(config.SMTPRelayConfig.Username, config.SMTPRelayConfig.Secret)
	}

	if a != nil {
		if err = c.Auth(a); err != nil {
			return fmt.Errorf("error authenticating: %s", err.Error())
		}
	}
	if err = c.Mail(from); err != nil {
		return fmt.Errorf("error response to MAIL command: %s", err.Error())
	}

	for _, addr := range recipients {
		if err = c.Rcpt(addr); err != nil {
			return fmt.Errorf("error response to RCPT command for %s: %s", addr, err.Error())
		}
	}

	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("error response to DATA command: %s", err.Error())
	}

	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("error sending message: %s", err.Error())
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("error closing connection: %s", err.Error())
	}

	return c.Quit()
}

// Custom implementation of LOGIN SMTP authentication
// @see https://gist.github.com/andelf/5118732
type loginAuth struct {
	username, password string
}

// LoginAuth authentication
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown fromServer")
		}
	}

	return nil, nil
}
