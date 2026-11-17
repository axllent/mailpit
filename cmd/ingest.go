package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
	"github.com/spf13/cobra"
)

var (
	ingestRecent int
)

// ingestCmd represents the ingest command
var ingestCmd = &cobra.Command{
	Use:   "ingest <file|folder> ...[file|folder]",
	Short: "Ingest a file or folder of emails for testing",
	Long: `Ingest a file or folder of emails for testing.

This command will scan the folder for emails and deliver them via SMTP to a running 
Mailpit server. Each email must be a separate file (eg: Maildir format, not mbox).
The --recent flag will only consider files with a modification date within the last X days.`,
	// Hidden: true,
	Args: cobra.MinimumNArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		var count int
		var total int
		var per100start = time.Now()
		limit := int64(config.MaxMessageSize) * 1024 * 1024

		for _, a := range args {
			err := filepath.Walk(a,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						logger.Log().Error(err)
						return nil
					}
					if !info.Mode().IsRegular() {
						return nil
					}

					if ingestRecent > 0 && time.Since(info.ModTime()) > time.Duration(ingestRecent)*24*time.Hour {
						return nil
					}

					if config.MaxMessageSize > 0 && info.Size() > limit {
						logger.Log().Warnf("%s exceeds %d MiB size cap, skipping", path, config.MaxMessageSize)
						return nil
					}

					f, err := os.Open(filepath.Clean(path))
					if err != nil {
						logger.Log().Errorf("%s: %s", path, err.Error())
						return nil
					}
					defer func() { _ = f.Close() }()

					var reader io.Reader = f
					if config.MaxMessageSize > 0 {
						reader = io.LimitReader(f, limit+1)
					}
					body, err := io.ReadAll(reader)
					if err != nil {
						logger.Log().Errorf("%s: %s", path, err.Error())
						return nil
					}
					if config.MaxMessageSize > 0 && int64(len(body)) > limit {
						logger.Log().Warnf("%s exceeds %d MiB size cap, skipping", path, config.MaxMessageSize)
						return nil
					}

					msg, err := mail.ReadMessage(bytes.NewReader(body))
					if err != nil {
						logger.Log().Errorf("error parsing message body: %s", err.Error())
						return nil
					}

					returnPath, recipients := ingestEnvelope(msg.Header, sendmail.FromAddr)

					err = sendmail.Send(sendmail.SMTPAddr, returnPath, recipients, body)
					if err != nil {
						logger.Log().Errorf("error sending mail: %s (%s)", err.Error(), path)
						return nil
					}

					count++
					total++
					if count%100 == 0 {
						logger.Log().Infof("[%s] 100 messages in %s", format(total), time.Since(per100start))

						per100start = time.Now()
					}

					return nil
				})
			if err != nil {
				logger.Log().Error(err)
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(ingestCmd)

	ingestCmd.Flags().StringVarP(&sendmail.SMTPAddr, "smtp-addr", "S", sendmail.SMTPAddr, "SMTP server address")
	ingestCmd.Flags().IntVarP(&ingestRecent, "recent", "r", 0, "Only ingest messages from the last X days (default all)")
	ingestCmd.Flags().IntVar(&config.MaxMessageSize, "max-message-size", config.MaxMessageSize, "Maximum message size in MB (0 = unlimited)")
}

// ingestEnvelope returns the envelope sender and recipients for an ingested message.
// Recipients are taken from To, Cc and Bcc. The sender is the Return-Path, else the
// first From address. When the message has no usable sender or recipients, both fall
// back to defaultFrom (the CLI-configured sender) so the message can still be delivered.
// defaultFrom is passed in rather than written to, as sendmail.FromAddr is shared
// across every message ingested.
func ingestEnvelope(h mail.Header, defaultFrom string) (string, []string) {
	recipients := sendmail.HeaderRecipients(h)

	var msgFrom string
	if fromAddresses, err := tools.AddressList(h, "From"); err == nil && len(fromAddresses) > 0 {
		msgFrom = tools.EnvelopeAddress(fromAddresses[0])
	}

	sender := defaultFrom
	if msgFrom != "" {
		sender = msgFrom
	}

	// Return-Path per RFC 5321 is "<addr>" (or "<>" for null). Use mail.ParseAddress
	// so only the wrapping angle brackets are stripped, not stray "<"/">" inside.
	if rp, err := mail.ParseAddress(h.Get("Return-Path")); err == nil && rp.Address != "" {
		sender = tools.EnvelopeAddress(rp)
	}

	if len(recipients) == 0 {
		// eg: Bcc-only messages with the header removed
		if msgFrom != "" {
			recipients = []string{msgFrom}
		} else {
			recipients = []string{defaultFrom}
		}
	}

	return sender, recipients
}

// Format a an integer 10000 => 10,000
func format(n int) string {
	in := fmt.Sprintf("%d", n)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits-- // First character is the - sign (not a digit)
	}
	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}
