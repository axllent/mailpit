// Package cmd is the sendmail cli
package cmd

/**
 * Bare bones sendmail drop-in replacement borrowed from MailHog
 */

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/mail"
	"net/smtp"
	"os"
	"os/user"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
	flag "github.com/spf13/pflag"
)

var (
	// Verbose flag
	Verbose bool

	fromAddr string
)

// Run the Mailpit sendmail replacement.
func Run() {
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	}

	username := "nobody"
	user, err := user.Current()
	if err == nil && user != nil && len(user.Username) > 0 {
		username = user.Username
	}

	if fromAddr == "" {
		fromAddr = username + "@" + host
	}

	smtpAddr := "localhost:1025"
	var recip []string

	// defaults from envars if provided
	if len(os.Getenv("MP_SENDMAIL_SMTP_ADDR")) > 0 {
		smtpAddr = os.Getenv("MP_SENDMAIL_SMTP_ADDR")
	}
	if len(os.Getenv("MP_SENDMAIL_FROM")) > 0 {
		fromAddr = os.Getenv("MP_SENDMAIL_FROM")
	}

	// override defaults from cli flags
	flag.StringVarP(&fromAddr, "from", "f", fromAddr, "SMTP sender address")
	flag.StringVarP(&smtpAddr, "smtp-addr", "S", smtpAddr, "SMTP server address")
	flag.BoolVarP(&Verbose, "verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	flag.BoolP("long-b", "b", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-o", "o", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-s", "s", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.CommandLine.SortFlags = false

	// set the default help
	flag.Usage = func() {
		fmt.Printf("A sendmail command replacement for Mailpit (%s).\n\n", config.Version)
		fmt.Printf("Usage:\n %s [flags] [recipients]\n", os.Args[0])
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
	}

	flag.Parse()

	// allow recipient to be passed as an argument
	recip = flag.Args()

	if Verbose {
		fmt.Fprintln(os.Stdout, smtpAddr, fromAddr)
	}

	body, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading stdin")
		os.Exit(11)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("error parsing message body: %s", err))
		os.Exit(11)
	}

	addresses := []string{}

	if len(recip) > 0 {
		addresses = recip
	} else {
		// get all recipients in To, Cc and Bcc
		if to, err := msg.Header.AddressList("To"); err == nil {
			for _, a := range to {
				addresses = append(addresses, a.Address)
			}
		}
		if cc, err := msg.Header.AddressList("Cc"); err == nil {
			for _, a := range cc {
				addresses = append(addresses, a.Address)
			}
		}
		if bcc, err := msg.Header.AddressList("Bcc"); err == nil {
			for _, a := range bcc {
				addresses = append(addresses, a.Address)
			}
		}
	}

	err = smtp.SendMail(smtpAddr, nil, fromAddr, addresses, body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}
}
