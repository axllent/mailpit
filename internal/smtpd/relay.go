package smtpd

import (
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
	"os"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/pkg/errors"
)

// relayRecipient is a recipient as it is matched (the parsed address) and as it is sent (the envelope address).
type relayRecipient struct {
	match string
	send  string
}

// parseRelayRecipients parses each recipient which was received. What was received can have comments, a name or
// quotes (which the SMTP server and the send API accept), so the block list and the matching expression are matched
// against the address as the parser reads it, and that address (and not what was received) is what is sent on: another
// server could read what was received as another address than the one which was matched. A recipient which does not
// parse is not relayed.
func parseRelayRecipients(to []string) []relayRecipient {
	recipients := make([]relayRecipient, 0, len(to))

	for _, address := range to {
		parsed, err := mail.ParseAddress(address)
		if err != nil {
			logger.Log().Warnf("[relay] ignoring auto-relay to %q: invalid address", address)
			continue
		}

		recipients = append(recipients, relayRecipient{match: parsed.Address, send: tools.EnvelopeAddress(parsed)})
	}

	return recipients
}

// Wrapper to auto relay messages if configured
func autoRelayMessage(from string, received []string, data *[]byte) error {
	recipients := parseRelayRecipients(received)

	if config.SMTPRelayConfig.BlockedRecipientsRegexp != nil {
		filtered := []relayRecipient{}
		for _, r := range recipients {
			if config.SMTPRelayConfig.BlockedRecipientsRegexp.MatchString(r.match) {
				logger.Log().Debugf("[relay] ignoring auto-relay to %s: found in blocklist", r.match)
				continue
			}

			filtered = append(filtered, r)
		}
		recipients = filtered
	}

	to := make([]string, 0, len(recipients))
	for _, r := range recipients {
		to = append(to, r.send)
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
	} else if config.SMTPRelayMatchingRegexp != nil {
		filtered := []string{}
		for _, r := range recipients {
			if config.SMTPRelayMatchingRegexp.MatchString(r.match) {
				filtered = append(filtered, r.send)
			}
		}

		if len(filtered) == 0 {
			return nil
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

		client, err := dialSMTP(addr, config.Host, true, tlsConf)
		if err != nil {
			return nil, err
		}

		// Set the hostname for HELO/EHLO
		if hostname, err := os.Hostname(); err == nil {
			if err := client.Hello(hostname); err != nil {
				_ = client.Close()
				return nil, fmt.Errorf("error saying HELO/EHLO to %s: %v", addr, err)
			}
		}

		// Note: The caller is responsible for closing the client
		return client, nil
	}

	client, err := dialSMTP(addr, config.Host, false, nil)
	if err != nil {
		return nil, err
	}

	// Set the hostname for HELO/EHLO
	if hostname, err := os.Hostname(); err == nil {
		if err := client.Hello(hostname); err != nil {
			_ = client.Close()
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

	// Whatever is sent has <CR><LF> line breaks only, and no whitespace before the colon of a header, as the receiving
	// server may read others differently.
	// A <CR> on its own is not changed by the SMTP client, and could end the message for the server.
	msg, _ = tools.NormaliseForSending(msg)

	if config.SMTPRelayConfig.OverrideFrom != "" {
		msg, err = tools.OverrideFromHeader(msg, config.SMTPRelayConfig.OverrideFrom)
		if err != nil {
			return fmt.Errorf("error overriding From header: %s", err.Error())
		}

		from = config.SMTPRelayConfig.OverrideFrom
	}

	// after override-from, so the return-path is the envelope sender
	if config.SMTPRelayConfig.ReturnPath != "" {
		msg, err = setReturnPath(msg, config.SMTPRelayConfig.ReturnPath)
		if err != nil {
			return err
		}

		from = config.SMTPRelayConfig.ReturnPath
	}

	if config.SMTPRelayConfig.SubjectPrefix != "" {
		msg, err = tools.PrefixSubject(msg, config.SMTPRelayConfig.SubjectPrefix)
		if err != nil {
			return fmt.Errorf("error prefixing Subject header: %s", err.Error())
		}
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
