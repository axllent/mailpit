package cmd

import (
	"os"
	"strconv"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server"
	"github.com/axllent/mailpit/smtpd"
	"github.com/axllent/mailpit/storage"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mailpit",
	Short: "Mailpit is an email testing tool for developers",
	Long: `Mailpit is an email testing tool for developers.

It acts as an SMTP server, and provides a web interface to view all captured emails.`,
	Run: func(_ *cobra.Command, _ []string) {
		if err := config.VerifyConfig(); err != nil {
			logger.Log().Error(err.Error())
			os.Exit(1)
		}
		if err := storage.InitDB(); err != nil {
			logger.Log().Error(err.Error())
			os.Exit(1)
		}

		go server.Listen()

		if err := smtpd.Listen(); err != nil {
			logger.Log().Error(err.Error())
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// SendmailExecute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func SendmailExecute() {
	args := []string{"mailpit", "sendmail"}

	rootCmd.Run(sendmailCmd, args)
}

func init() {
	// hide autocompletion
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	// rootCmd.Flags().SortFlags = false
	// hide help
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	// defaults from envars if provided
	if len(os.Getenv("MP_DATA_DIR")) > 0 {
		config.DataDir = os.Getenv("MP_DATA_DIR")
	}
	if len(os.Getenv("MP_SMTP_BIND_ADDR")) > 0 {
		config.SMTPListen = os.Getenv("MP_SMTP_BIND_ADDR")
	}
	if len(os.Getenv("MP_UI_BIND_ADDR")) > 0 {
		config.HTTPListen = os.Getenv("MP_UI_BIND_ADDR")
	}
	if len(os.Getenv("MP_MAX_MESSAGES")) > 0 {
		config.MaxMessages, _ = strconv.Atoi(os.Getenv("MP_MAX_MESSAGES"))
	}

	rootCmd.Flags().StringVarP(&config.DataDir, "data", "d", config.DataDir, "Optional path to store peristent data")
	rootCmd.Flags().StringVarP(&config.SMTPListen, "smtp", "s", config.SMTPListen, "SMTP bind interface and port")
	rootCmd.Flags().StringVarP(&config.HTTPListen, "listen", "l", config.HTTPListen, "HTTP bind interface and port for UI")
	rootCmd.Flags().IntVarP(&config.MaxMessages, "max", "m", config.MaxMessages, "Max number of messages per mailbox")
	rootCmd.Flags().BoolVarP(&config.VerboseLogging, "verbose", "v", false, "Verbose logging")
}
