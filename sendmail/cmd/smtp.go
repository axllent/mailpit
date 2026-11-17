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
	"time"

	"github.com/axllent/mailpit/internal/tools"
)

var (
	// dialTimeout is how long connecting to the server may take. It is a variable for the tests.
	dialTimeout = 10 * time.Second

	// idleTimeout is how long the server may take to answer (or to take what is sent) once connected. It is not for
	// the whole of the conversation, so a large message is still sent as long as it moves. It is a variable for the
	// tests.
	idleTimeout = 30 * time.Second

	// replyTimeout is how long the server may take to answer once the message has been sent. Mailpit does not answer
	// until the message has been stored, which can wait up to two minutes for its turn when large messages are being
	// received. Giving up before then would report an error for a message which is stored, and be sent again. It is
	// a variable for the tests.
	replyTimeout = 3 * time.Minute
)

// Send is a wrapper for smtp.SendMail() which also supports sending via unix sockets.
// Unix sockets must be set as unix:/path/to/socket
// It does not support authentication.
//
// Connecting has a time limit, and so has each time the server is waited for (a server which accepts the connection
// and then says nothing would otherwise hold the program for ever).
func Send(addr string, from string, to []string, msg []byte) error {
	socketPath, isSocket := socketAddress(addr)

	fromAddress, err := mail.ParseAddress(from)
	if err != nil {
		return fmt.Errorf("invalid from address: %s", from)
	}

	if len(to) == 0 {
		return fmt.Errorf("no To addresses specified")
	}

	// the sender as it is used in the command, which keeps the quotes of a local part which has to have them
	sender := tools.EnvelopeAddress(fromAddress)

	if err := validateLine(sender); err != nil {
		return err
	}
	for _, recipient := range to {
		if err := validateLine(recipient); err != nil {
			return err
		}
	}

	network, address := "tcp", addr
	if isSocket {
		network, address = "unix", socketPath
	}

	client, idle, err := dial(network, address)
	if err != nil {
		if isSocket {
			return fmt.Errorf("error connecting to %s: %w", addr, err)
		}

		return err
	}
	defer func() { _ = client.Close() }()

	// A server which is not on a socket is also told who we are, and offered TLS (it is not checked, as this is
	// for a server which is local)
	if !isSocket {
		// Use the local hostname for EHLO/HELO as required by RFC 5321.
		// Fall back to "localhost" if the hostname cannot be determined.
		localHostname, err := os.Hostname()
		if err != nil {
			localHostname = "localhost"
		}
		if err = client.Hello(localHostname); err != nil {
			return err
		}

		if ok, _ := client.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: addr, InsecureSkipVerify: true} // #nosec
			if err = client.StartTLS(config); err != nil {
				return err
			}
		}
	}

	if err := client.Mail(sender); err != nil {
		return fmt.Errorf("error setting sender: %w", err)
	}

	for _, a := range to {
		if err := client.Rcpt(a); err != nil {
			return err
		}
	}

	wc, err := client.Data()
	if err != nil {
		return err
	}

	if _, err := wc.Write(msg); err != nil {
		return err
	}

	// closing sends the end of the message, and waits for the server to say that it has it
	idle.SetTimeout(replyTimeout)
	err = wc.Close()
	idle.SetTimeout(idleTimeout)
	if err != nil {
		return err
	}

	// The server has the message, so it is sent whatever it says to QUIT (which is only polite): an error here would
	// be taken for a message which was not, and it would be sent again.
	_ = client.Quit()

	return nil
}

// dial connects to the server (a TCP address, or the path of a Unix socket) and reads its greeting. It returns the
// connection as well, so that the time limit on it can be changed.
func dial(network, address string) (*smtp.Client, *tools.IdleConn, error) {
	conn, err := net.DialTimeout(network, address, dialTimeout)
	if err != nil {
		return nil, nil, err
	}

	// the name of the server is its host, as smtp.Dial has it (there is none for a socket)
	var host string
	if network == "tcp" {
		host, _, _ = net.SplitHostPort(address)
	}

	idle := tools.NewIdleConn(conn, idleTimeout)

	client, err := smtp.NewClient(idle, host)
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}

	return client, idle, nil
}

// validateLine checks to see if a line has CR or LF as per RFC 5321.
func validateLine(line string) error {
	if strings.ContainsAny(line, "\n\r") {
		return errors.New("smtp: A line must not contain CR or LF")
	}
	return nil
}
