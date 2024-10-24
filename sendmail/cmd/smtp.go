// Package cmd is a wrapper library to send mail
package cmd

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"

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
		return smtp.SendMail(addr, nil, fromAddress.Address, to, msg)
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
