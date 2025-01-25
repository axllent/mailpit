package smtpd

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
)

// Wrapper to forward messages if configured
func autoForwardMessage(from string, data *[]byte) {
	if config.SMTPForwardConfig.Host == "" {
		return
	}

	if err := forward(from, *data); err != nil {
		logger.Log().Errorf("[forward] error: %s", err.Error())
	} else {
		logger.Log().Debugf("[forward] message from %s to %s via %s:%d",
			from, config.SMTPForwardConfig.To, config.SMTPForwardConfig.Host, config.SMTPForwardConfig.Port)
	}
}

// Forward will connect to a pre-configured SMTP server and send a message to one or more recipients.
func forward(from string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPForwardConfig.Host, config.SMTPForwardConfig.Port)

	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("error connecting to %s: %s", addr, err.Error())
	}

	defer c.Close()

	if config.SMTPForwardConfig.STARTTLS {
		conf := &tls.Config{ServerName: config.SMTPForwardConfig.Host} // #nosec

		conf.InsecureSkipVerify = config.SMTPForwardConfig.AllowInsecure

		if err = c.StartTLS(conf); err != nil {
			return fmt.Errorf("error creating StartTLS config: %s", err.Error())
		}
	}

	auth := forwardAuthFromConfig()

	if auth != nil {
		if err = c.Auth(auth); err != nil {
			return fmt.Errorf("error response to AUTH command: %s", err.Error())
		}
	}

	if config.SMTPForwardConfig.OverrideFrom != "" {
		msg, err = tools.OverrideFromHeader(msg, config.SMTPForwardConfig.OverrideFrom)
		if err != nil {
			return fmt.Errorf("error overriding From header: %s", err.Error())
		}

		from = config.SMTPForwardConfig.OverrideFrom
	}

	if err = c.Mail(from); err != nil {
		return fmt.Errorf("error response to MAIL command: %s", err.Error())
	}

	to := strings.Split(config.SMTPForwardConfig.To, ",")

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

// Return the SMTP forwarding authentication based on config
func forwardAuthFromConfig() smtp.Auth {
	var a smtp.Auth

	if config.SMTPForwardConfig.Auth == "plain" {
		a = smtp.PlainAuth("", config.SMTPForwardConfig.Username, config.SMTPForwardConfig.Password, config.SMTPForwardConfig.Host)
	}

	if config.SMTPForwardConfig.Auth == "login" {
		a = LoginAuth(config.SMTPForwardConfig.Username, config.SMTPForwardConfig.Password)
	}

	if config.SMTPForwardConfig.Auth == "cram-md5" {
		a = smtp.CRAMMD5Auth(config.SMTPForwardConfig.Username, config.SMTPForwardConfig.Secret)
	}

	return a
}
