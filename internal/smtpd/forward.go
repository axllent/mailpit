package smtpd

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
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

func createForwardingSMTPClient(config config.SMTPForwardConfigStruct, addr string) (*smtp.Client, error) {
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

	if config.STARTTLS {
		tlsConf := &tls.Config{ServerName: config.Host} // #nosec
		tlsConf.InsecureSkipVerify = config.AllowInsecure

		if err = client.StartTLS(tlsConf); err != nil {
			_ = client.Close()
			return nil, fmt.Errorf("error creating StartTLS config: %v", err)
		}
	}

	// Set the hostname for HELO/EHLO
	if hostname, err := os.Hostname(); err == nil {
		if err := client.Hello(hostname); err != nil {
			return nil, fmt.Errorf("error saying HELO/EHLO to %s: %v", addr, err)
		}
	}

	// Note: The caller is responsible for closing the client
	return client, nil
}

// Forward will connect to a pre-configured SMTP server and send a message to one or more recipients.
func forward(from string, msg []byte) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPForwardConfig.Host, config.SMTPForwardConfig.Port)

	c, err := createForwardingSMTPClient(config.SMTPForwardConfig, addr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

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
