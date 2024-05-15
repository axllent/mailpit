// Package cmd is the sendmail cli
package cmd

/**
 * Bare bones sendmail drop-in replacement borrowed from MailHog
 *
 * It uses a bit of a hack for flag parsing in order to be compatible
 * with the cobra sendmail subcommand, as sendmail uses `-bc` which
 * is not POSIX compatible.
 *
 * The -bs command-line switch causes sendmail to run a single SMTP session in the
 * foreground over its standard input and output, and then exit. The SMTP session
 * is exactly like a network SMTP session. Usually, one or more messages are
 * submitted to sendmail for delivery.
 */
import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"net/smtp"
	"os"
	"os/user"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/reiver/go-telnet"
	flag "github.com/spf13/pflag"
)

var (
	// SMTPAddr address
	SMTPAddr = "localhost:1025"
	// FromAddr email address
	FromAddr string

	// UseB - used to set from `-bs`
	UseB bool
	// UseS - used to set from `-bs`
	UseS bool
)

func init() {
	// ensure only valid characters are used, ie: windows
	re := regexp.MustCompile(`[^a-zA-Z\-\.\_]`)
	host, err := os.Hostname()
	if err != nil {
		host = "localhost"
	} else {
		host = re.ReplaceAllString(host, "-")
	}

	username := "nobody"
	user, err := user.Current()
	if err == nil && user != nil && len(user.Username) > 0 {
		username = re.ReplaceAllString(user.Username, "-")
	}

	if FromAddr == "" {
		FromAddr = username + "@" + host
	}
}

// Run the Mailpit sendmail replacement.
func Run() {
	var recipients []string

	// defaults from env vars if provided
	if len(os.Getenv("MP_SENDMAIL_SMTP_ADDR")) > 0 {
		SMTPAddr = os.Getenv("MP_SENDMAIL_SMTP_ADDR")
	}
	if len(os.Getenv("MP_SENDMAIL_FROM")) > 0 {
		FromAddr = os.Getenv("MP_SENDMAIL_FROM")
	}

	flag.StringVarP(&FromAddr, "from", "f", FromAddr, "SMTP sender")
	flag.StringVarP(&SMTPAddr, "smtp-addr", "S", SMTPAddr, "SMTP server address")
	flag.BoolVarP(&UseB, "long-b", "b", false, "Handle SMTP commands on standard input (use as -bs)")
	flag.BoolVarP(&UseS, "long-s", "s", false, "Handle SMTP commands on standard input (use as -bs)")
	flag.BoolP("verbose", "v", false, "Ignored")
	flag.BoolP("long-i", "i", false, "Ignored")
	flag.BoolP("long-o", "o", false, "Ignored")
	flag.BoolP("long-t", "t", false, "Ignored")

	// set the default help
	flag.Usage = func() {
		fmt.Println(HelpTemplate(os.Args[0:1]))
	}

	var showHelp bool
	// avoid 'pflag: help requested' error
	flag.BoolVarP(&showHelp, "help", "h", false, "")

	flag.Parse()

	// allow recipients to be passed as an argument
	recipients = flag.Args()

	// if run via `mailpit sendmail ...` then remove `sendmail` from "recipients"
	if len(recipients) > 0 && recipients[0] == "sendmail" {
		recipients = recipients[1:]
	}

	if showHelp {
		flag.Usage()
		os.Exit(0)
	}

	// ensure -bs is set
	if UseB && !UseS || !UseB && UseS {
		fmt.Printf("error: use -bs")
		os.Exit(1)
	}

	// handles `sendmail -bs`
	if UseB && UseS {
		var caller telnet.Caller = telnet.StandardCaller

		// telnet directly to SMTP
		if err := telnet.DialToAndCall(SMTPAddr, caller); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		return
	}

	body, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error reading stdin")
		os.Exit(11)
	}

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing message body: %si\n", err)
		os.Exit(11)
	}

	addresses := []string{}

	if len(recipients) > 0 {
		addresses = recipients
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

	from, err := mail.ParseAddress(FromAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid from address")
		os.Exit(11)
	}

	err = smtp.SendMail(SMTPAddr, nil, from.Address, addresses, body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}
}

// HelpTemplate returns a string of the help
func HelpTemplate(args []string) string {
	return fmt.Sprintf(`A sendmail command replacement for Mailpit (%s)

Usage: %s [flags] [recipients] < message

See: https://github.com/axllent/mailpit

Flags:
  -S  string  SMTP server address (default "localhost:1025")
  -f  string  Set the envelope sender address (default "%s")
  -bs         Handle SMTP commands on standard input
  -t          Ignored
  -i          Ignored
  -o          Ignored
  -v          Ignored
`, config.Version, strings.Join(args, " "), FromAddr)
}
