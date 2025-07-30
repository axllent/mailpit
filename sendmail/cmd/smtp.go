// Package cmd is a wrapper library to send mail
package cmd

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
)

// Send is a wrapper for smtp.SendMail() which also supports sending via unix sockets.
// Unix sockets must be set as unix:/path/to/socket
// It does not support authentication.
func Send(addr string, from string, to []string, msg []byte) error {
	socketPath, isSocket := socketAddress(addr)

	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("invalid from address: %s", from)
	}

	if len(to) == 0 {
		return fmt.Errorf("no To addresses specified")
	}

	if !isSocket {
		return sendMail(addr, nil, fromAddress.Address, to, msg)
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("error connecting to %s", addr)
	}

	client, err := smtp.NewClient(conn, "")
	if err != nil {
		return err
	}

	// Set the sender
	if err := client.Mail(fromAddress.Address); err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}

	// Set the recipient
	for _, a := range to {
		if err := client.Rcpt(a); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}

	_, err = wc.Write(msg)
	if err != nil {
		return err
	}
	err = wc.Close()
	if err != nil {
		return err
	}

	return nil
}

func sendMail(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
	if err := validateLine(from); err != nil {
		return err
	}

	for _, recipient := range to {
		if err := validateLine(recipient); err != nil {
			return err
		}
	}

	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	if err = c.Hello(addr); err != nil {
		return err
	}

	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: addr, InsecureSkipVerify: true} // #nosec
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}

	if a != nil {
		if ok, _ := c.Extension("AUTH"); !ok {
			return errors.New("smtp: server doesn't support AUTH")
		}
		if err = c.Auth(a); err != nil {
			return err
		}
	}

	if err = c.Mail(from); err != nil {
		return err
	}

	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}

	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write(msg)
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}

// validateLine checks to see if a line has CR or LF as per RFC 5321.
func validateLine(line string) error {
	if strings.ContainsAny(line, "\n\r") {
		return errors.New("smtp: A line must not contain CR or LF")
	}
	return nil
}
