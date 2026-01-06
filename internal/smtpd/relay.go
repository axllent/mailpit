package smtpd

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net/smtp"
	"os"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/pkg/errors"
)

// Wrapper to auto relay messages if configured
func autoRelayMessage(from string, to []string, subject string, data *[]byte) error {
	if config.SMTPRelayConfig.BlockedRecipientsRegexp != nil {
		filteredTo := []string{}
		for _, address := range to {
			if config.SMTPRelayConfig.BlockedRecipientsRegexp.MatchString(address) {
				logger.Log().Debugf("[relay] ignoring auto-relay to %s: found in blocklist", address)
				continue
			}

			filteredTo = append(filteredTo, address)
		}
		to = filteredTo
	}

	if len(to) == 0 {
		return nil
	}

	if config.SMTPRelayAll {
		if err := Relay(from, to, *data); err != nil {
			return errors.WithMessage(err, "[relay] error")
		}

		logger.Log().Debugf(
			"[relay] sent message to %s from %s via %s:%d",
			strings.Join(to, ", "), from, config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port,
		)
	} else if config.SMTPRelayMatchingRegexp != nil || config.SMTPRelayMatchingSubjectRegexp != nil {
		filtered := []string{}
		for _, t := range to {
			if config.SMTPRelayMatchingRegexp != nil && !config.SMTPRelayMatchingRegexp.MatchString(t) {
				continue
			}

			filtered = append(filtered, t)
		}

		if len(filtered) == 0 {
			logger.Log().Debugf("[relay] Empty filter list")
			return nil
		}

		if config.SMTPRelayMatchingRegexp == nil && config.SMTPRelayMatchingSubjectRegexp != nil {
			decodedSubject, err := (&mime.WordDecoder{}).DecodeHeader(subject)
			if err != nil {
				return errors.WithMessage(err, "[relay] error")
			}

			if !config.SMTPRelayMatchingSubjectRegexp.MatchString(decodedSubject) {
				logger.Log().Debugf("[relay] ignoring auto-relay: subject does not match")
				return nil
			}
		}

		if err := Relay(from, filtered, *data); err != nil {
			return errors.WithMessage(err, "[relay] error")
		}

		logger.Log().Debugf(
			"[relay] auto-relay message to %s from %s via %s:%d",
			strings.Join(filtered, ", "), from, config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port,
		)
	}

	return nil
}

func createRelaySMTPClient(config config.SMTPRelayConfigStruct, addr string) (*smtp.Client, error) {
	if config.TLS {
		tlsConf := &tls.Config{ServerName: config.Host} // #nosec
		tlsConf.InsecureSkipVerify = config.AllowInsecure

		conn, err := tls.Dial("tcp", addr, tlsConf)
		if err != nil {
			return nil, fmt.Errorf("TLS dial error: %v", err)
		}

		client, err := smtp.NewClient(conn, tlsConf.ServerName)
		if err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("SMTP client error: %v", err)
		}

		// Note: The caller is responsible for closing the client
		return client, nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return nil, fmt.Errorf("error connecting to %s: %v", addr, err)
	}

	// Set the hostname for HELO/EHLO
	if hostname, err := os.Hostname(); err == nil {
		if err := client.Hello(hostname); err != nil {
			return nil, fmt.Errorf("error saying HELO/EHLO to %s: %v", addr, err)
		}
	}

	if config.STARTTLS {
		tlsConf := &tls.Config{ServerName: config.Host} // #nosec
		tlsConf.InsecureSkipVerify = config.AllowInsecure

		if err = client.StartTLS(tlsConf); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("error creating StartTLS config: %v", err)
		}
	}

	// Note: The caller is responsible for closing the client
	return client, nil
}

// Relay will connect to a pre-configured SMTP server and send a message to one or more recipients.
func Relay(from string, to []string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)

	c, err := createRelaySMTPClient(config.SMTPRelayConfig, addr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	auth := relayAuthFromConfig()

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("error response to AUTH command: %s", err.Error())
		}
	}

	if config.SMTPRelayConfig.OverrideFrom != "" {
		msg, err = tools.OverrideFromHeader(msg, config.SMTPRelayConfig.OverrideFrom)
		if err != nil {
			return fmt.Errorf("error overriding From header: %s", err.Error())
		}

		from = config.SMTPRelayConfig.OverrideFrom
	}

	if err = c.Mail(from); err != nil {
		return errors.WithMessage(err, "error sending MAIL command")
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			logger.Log().Warnf("error response to RCPT command for %s: %s", addr, err.Error())
			if config.SMTPRelayConfig.ForwardSMTPErrors {
				return errors.WithMessagef(err, "error response to RCPT command for %s", addr)
			}
		}
	}

	w, err := c.Data()
	if err != nil {
		return errors.WithMessage(err, "error response to DATA command")
	}

	if _, err := w.Write(msg); err != nil {
		return errors.WithMessage(err, "error sending message")
	}

	if err := w.Close(); err != nil {
		return errors.WithMessage(err, "error closing connection")
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
	return &loginAuth{
		username,
		password,
	}
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
			return nil, errors.New("unknown fromServer")
		}
	}

	return nil, nil
}
