package smtpd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
)

// Send will connect to a pre-configured SMTP server and send a message to one or more recipients.
func Send(from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %s", addr, err.Error())
	}

	defer c.Close()

	if config.SMTPRelayConfig.STARTTLS {
		conf := &tls.Config{ServerName: config.SMTPRelayConfig.Host} // #nosec

		conf.InsecureSkipVerify = config.SMTPRelayConfig.AllowInsecure

		if err = c.StartTLS(conf); err != nil {
			return fmt.Errorf("error creating StartTLS config: %s", err.Error())
		}
	}

	auth := relayAuthFromConfig()

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("error response to AUTH command: %s", err.Error())
		}
	}

	if err = c.Mail(from); err != nil {
		return fmt.Errorf("error response to MAIL command: %s", err.Error())
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			logger.Log().Warnf("error response to RCPT command for %s: %s", addr, err.Error())
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

// Return the SMTP relay authentication based on config
func relayAuthFromConfig() smtp.Auth {
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

	return a
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

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
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
