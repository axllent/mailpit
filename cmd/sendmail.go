package cmd

import (
	"os"

	sendmail "github.com/axllent/mailpit/sendmail/cmd"
	"github.com/spf13/cobra"
)

// sendmailCmd represents the sendmail command
var sendmailCmd = &cobra.Command{
	Use:   "sendmail [flags] [recipients]",
	Short: "A sendmail command replacement for Mailpit",
	Run: func(_ *cobra.Command, _ []string) {

		sendmail.Run()
	},
}

func init() {
	rootCmd.AddCommand(sendmailCmd)

	// print out manual help screen
	rootCmd.SetHelpTemplate(sendmail.HelpTemplate([]string{os.Args[0], "sendmail"}))

	// these are simply repeated for cli consistency as cobra/viper does not allow
	// multi-letter single-dash variables (-bs)
	sendmailCmd.Flags().StringVarP(&sendmail.FromAddr, "from", "f", sendmail.FromAddr, "SMTP sender")
	sendmailCmd.Flags().StringVarP(&sendmail.SMTPAddr, "smtp-addr", "S", sendmail.SMTPAddr, "SMTP server address")
	sendmailCmd.Flags().BoolVarP(&sendmail.UseB, "long-b", "b", false, "Handle SMTP commands on standard input (use as -bs)")
	sendmailCmd.Flags().BoolVarP(&sendmail.UseS, "long-s", "s", false, "Handle SMTP commands on standard input (use as -bs)")
	sendmailCmd.Flags().BoolP("verbose", "v", false, "Verbose mode (sends debug output to stderr)")
	sendmailCmd.Flags().BoolP("i", "i", false, "Ignored")
	sendmailCmd.Flags().BoolP("o", "o", false, "Ignored")
	sendmailCmd.Flags().BoolP("t", "t", false, "Ignored")
}
