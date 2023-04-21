package cmd

import (
	sendmail "github.com/axllent/mailpit/sendmail/cmd"
	"github.com/spf13/cobra"
)

var (
	smtpAddr = "localhost:1025"
	fromAddr string
)

// sendmailCmd represents the sendmail command
var sendmailCmd = &cobra.Command{
	Use:   "sendmail [flags] [recipients]",
	Short: "A sendmail command replacement for Mailpit",
	Long: `A sendmail command replacement for Mailpit.
	
You can optionally create a symlink called 'sendmail' to the Mailpit binary.`,
	Run: func(_ *cobra.Command, _ []string) {
		sendmail.Run()
	},
}

func init() {
	rootCmd.AddCommand(sendmailCmd)

	// these are simply repeated for cli consistency
	sendmailCmd.Flags().StringVarP(&fromAddr, "from", "f", fromAddr, "SMTP sender")
	sendmailCmd.Flags().StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	sendmailCmd.Flags().BoolVarP(&sendmail.Verbose, "verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	sendmailCmd.Flags().BoolP("long-b", "b", false, "Ignored. This flag exists for sendmail compatibility.")
	sendmailCmd.Flags().BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	sendmailCmd.Flags().BoolP("long-o", "o", false, "Ignored. This flag exists for sendmail compatibility.")
	sendmailCmd.Flags().BoolP("long-s", "s", false, "Ignored. This flag exists for sendmail compatibility.")
	sendmailCmd.Flags().BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
}
