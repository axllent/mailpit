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
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"time"

	"net"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
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
	flag.StringP("from-name", "F", "", "Ignored")
	flag.StringP("bits", "B", "", "Ignored")
	flag.StringP("errors", "e", "", "Ignored")

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

	socketAddr, isSocket := socketAddress(SMTPAddr)

	// handles `sendmail -bs`
	// relay stdin/stdout to SMTP connection
	if UseB && UseS {
		network := "tcp"
		addr := SMTPAddr
		if isSocket {
			network = "unix"
			addr = socketAddr
		}

		conn, err := net.DialTimeout(network, addr, dialTimeout)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		defer func() { _ = conn.Close() }()

		done := make(chan struct{})
		go func() {
			_, _ = io.Copy(os.Stdout, conn)
			close(done)
		}()
		_, _ = io.Copy(conn, os.Stdin)
		if cw, ok := conn.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
		select {
		case <-done:
		case <-time.After(30 * time.Second):
		}

		return
	}

	body, msg, err := readMessage(os.Stdin)
	if err != nil {
		var tooLarge messageTooLargeError

		switch {
		case errors.As(err, &tooLarge):
			fmt.Fprintf(os.Stderr, "message exceeds %d MiB size cap\n", tooLarge.limit/(1024*1024))
		case errors.Is(err, errReadingInput):
			fmt.Fprintln(os.Stderr, "error reading stdin")
		default:
			fmt.Fprintf(os.Stderr, "error parsing message body: %s\n", err)
		}
		os.Exit(11)
	}

	addresses := recipients
	if len(addresses) == 0 {
		addresses = HeaderRecipients(msg.Header)
	}

	from, err := envelopeSender(FromAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid from address")
		os.Exit(11)
	}

	if err := Send(SMTPAddr, from, addresses, body); err != nil {
		fmt.Fprintln(os.Stderr, "error sending mail")
		logger.Log().Fatal(err)
	}
}

// maxMessageSize is the largest message which is read. It is a variable for the tests.
var maxMessageSize int64 = 1000 * 1024 * 1024

var errReadingInput = errors.New("error reading the message")

// messageTooLargeError is returned for a message which is larger than maxMessageSize.
type messageTooLargeError struct{ limit int64 }

func (e messageTooLargeError) Error() string {
	return fmt.Sprintf("message exceeds %d MiB size cap", e.limit/(1024*1024))
}

// readMessage reads the message, which is at most maxMessageSize, and returns it as it is to be sent with the headers
// of it read. The line breaks of the message are all <CR><LF> (and there is no space between the name of a header and
// its colon), as they are for a message which is relayed: a <CR> on its own is not changed by the SMTP client, and could
// end the message for a server which takes it as a line break, with what follows read as commands. The message is
// changed before the headers are read, so that the recipients are the ones which a server would see.
func readMessage(r io.Reader) ([]byte, *mail.Message, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxMessageSize+1))
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %w", errReadingInput, err)
	}
	if int64(len(body)) > maxMessageSize {
		return nil, nil, messageTooLargeError{limit: maxMessageSize}
	}

	body, _ = tools.NormaliseForSending(body)

	msg, err := mail.ReadMessage(bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	return body, msg, nil
}

// envelopeSender returns the sender for the SMTP command from the address which is given (by -f or the
// environment), with the quotes which its local part needs (see tools.EnvelopeAddress).
func envelopeSender(from string) (string, error) {
	a, err := mail.ParseAddress(from)
	if err != nil {
		return "", err
	}

	return tools.EnvelopeAddress(a), nil
}

// HeaderRecipients returns the recipients of a message from its To, Cc and Bcc headers, as they are used in an SMTP
// command (see tools.EnvelopeAddress).
func HeaderRecipients(h mail.Header) []string {
	addresses := []string{}

	for _, key := range []string{"To", "Cc", "Bcc"} {
		if list, err := tools.AddressList(h, key); err == nil {
			for _, a := range list {
				addresses = append(addresses, tools.EnvelopeAddress(a))
			}
		}
	}

	return addresses
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
  -F  string  Ignored
  -B  string  Ignored
  -e  string  Ignored
`, config.Version, strings.Join(args, " "), FromAddr)
}

// SocketAddress returns a path and a FileMode if the address is in
// the format of unix:<path>
func socketAddress(address string) (string, bool) {
	re := regexp.MustCompile(`^unix:(.*)$`)

	if !re.MatchString(address) {
		return "", false
	}

	m := re.FindAllStringSubmatch(address, 1)

	return path.Clean(m[0][1]), true
}
