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

	"github.com/axllent/mailpit/utils/logger"
	flag "github.com/spf13/pflag"
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

	fromAddr := username + "@" + host
	smtpAddr := "localhost:1025"
	var recip []string

	// defaults from envars if provided
	if len(os.Getenv("MP_SENDMAIL_SMTP_ADDR")) > 0 {
		smtpAddr = os.Getenv("MP_SENDMAIL_SMTP_ADDR")
	}
	if len(os.Getenv("MP_SENDMAIL_FROM")) > 0 {
		fromAddr = os.Getenv("MP_SENDMAIL_FROM")
	}

	var verbose bool

	// override defaults from cli flags
	flag.StringVarP(&fromAddr, "from", "f", fromAddr, "SMTP sender")
	flag.StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	flag.BoolVarP(&verbose, "verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	flag.BoolP("long-b", "b", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-o", "o", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-s", "s", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
	flag.CommandLine.SortFlags = false
	flag.Parse()

	// allow recipient to be passed as an argument
	recip = flag.Args()

	if verbose {
		fmt.Fprintln(os.Stderr, smtpAddr, fromAddr)
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

	if len(recip) == 0 {
		// We only need to parse the message to get a recipient if none where
		// provided on the command line.
		recip = append(recip, msg.Header.Get("To"))
	}

	err = smtp.SendMail(smtpAddr, nil, fromAddr, recip, body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}
}
