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
	Use:   "sendmail",
	Short: "A sendmail command replacement",
	Long: `A sendmail command replacement.
	
You can optionally create a symlink called 'sendmail' to the main binary.`,
	Run: func(_ *cobra.Command, _ []string) {
		sendmail.Run()
	},
}

func init() {
	rootCmd.AddCommand(sendmailCmd)

	// these are simply repeated for cli consistency
	sendmailCmd.Flags().StringVar(&smtpAddr, "smtp-addr", smtpAddr, "SMTP server address")
	sendmailCmd.Flags().StringVarP(&fromAddr, "from", "f", "", "SMTP sender")
	sendmailCmd.Flags().BoolP("long-i", "i", false, "Ignored. This flag exists for sendmail compatibility.")
	sendmailCmd.Flags().BoolP("long-t", "t", false, "Ignored. This flag exists for sendmail compatibility.")
}
