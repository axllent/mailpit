package smtpd

import (
	"crypto/tls"
	"fmt"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
	"net/mail"
	"net/smtp"
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
		return nil
	}

	addr := fmt.Sprintf("%s:%d", config.SMTPRelayConfig.Host, config.SMTPRelayConfig.Port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}

	defer c.Close()

	if config.SMTPRelayConfig.STARTTLS {
		conf := &tls.Config{ServerName: config.SMTPRelayConfig.Host}

		conf.InsecureSkipVerify = config.SMTPRelayConfig.AllowInsecure

		if err = c.StartTLS(conf); err != nil {
			return err
		}
	}

	var a smtp.Auth

	if config.SMTPRelayConfig.Auth == "plain" {
		a = smtp.PlainAuth("", config.SMTPRelayConfig.Username, config.SMTPRelayConfig.Password, config.SMTPRelayConfig.Host)
	}

	if config.SMTPRelayConfig.Auth == "cram-md5" {
		a = smtp.CRAMMD5Auth(config.SMTPRelayConfig.Username, config.SMTPRelayConfig.Secret)
	}

	if a != nil {
		if err = c.Auth(a); err != nil {
			return err
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range recipients {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	if _, err := w.Write(msg); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return c.Quit()
}
