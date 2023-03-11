// Package cmd is the main application
package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server"
	"github.com/axllent/mailpit/server/smtpd"
	"github.com/axllent/mailpit/storage"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/spf13/cobra"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mailpit",
	Short: "Mailpit is an email testing tool for developers",
	Long: `Mailpit is an email testing tool for developers.

It acts as an SMTP server, and provides a web interface to view all captured emails.

Documentation:
  https://github.com/axllent/mailpit
  https://github.com/axllent/mailpit/wiki`,
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
	rootCmd.Flags().SortFlags = false
	// hide help command
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	// hide help flag
	rootCmd.PersistentFlags().BoolP("help", "h", false, "This help")
	rootCmd.PersistentFlags().Lookup("help").Hidden = true

	initConfigFromEnv()

	// deprecated 2022/08/06
	if len(os.Getenv("MP_AUTH_FILE")) > 0 {
		fmt.Println("MP_AUTH_FILE has been deprecated, use MP_UI_AUTH_FILE")
		config.UIAuthFile = os.Getenv("MP_AUTH_FILE")
	}
	// deprecated 2022/08/06
	if len(os.Getenv("MP_SSL_CERT")) > 0 {
		fmt.Println("MP_SSL_CERT has been deprecated, use MP_UI_SSL_CERT")
		config.UISSLCert = os.Getenv("MP_SSL_CERT")
	}
	// deprecated 2022/08/06
	if len(os.Getenv("MP_SSL_KEY")) > 0 {
		fmt.Println("MP_SSL_KEY has been deprecated, use MP_UI_SSL_KEY")
		config.UISSLKey = os.Getenv("MP_SSL_KEY")
	}
	// deprecated 2022/08/28
	if len(os.Getenv("MP_DATA_DIR")) > 0 {
		fmt.Println("MP_DATA_DIR has been deprecated, use MP_DATA_FILE")
		config.DataFile = os.Getenv("MP_DATA_DIR")
	}

	rootCmd.Flags().StringVarP(&config.DataFile, "db-file", "d", config.DataFile, "Database file to store persistent data")
	rootCmd.Flags().StringVarP(&config.SMTPListen, "smtp", "s", config.SMTPListen, "SMTP bind interface and port")
	rootCmd.Flags().StringVarP(&config.HTTPListen, "listen", "l", config.HTTPListen, "HTTP bind interface and port for UI")
	rootCmd.Flags().IntVarP(&config.MaxMessages, "max", "m", config.MaxMessages, "Max number of messages to store")
	rootCmd.Flags().StringVar(&config.Webroot, "webroot", config.Webroot, "Set the webroot for web UI & API")
	rootCmd.Flags().BoolVar(&config.UseMessageDates, "use-message-dates", false, "Use message dates as the received dates")

	rootCmd.Flags().StringVar(&config.UIAuthFile, "ui-auth-file", config.UIAuthFile, "A password file for web UI authentication")
	rootCmd.Flags().StringVar(&config.UISSLCert, "ui-ssl-cert", config.UISSLCert, "SSL certificate for web UI - requires ui-ssl-key")
	rootCmd.Flags().StringVar(&config.UISSLKey, "ui-ssl-key", config.UISSLKey, "SSL key for web UI - requires ui-ssl-cert")

	rootCmd.Flags().StringVar(&config.SMTPAuthFile, "smtp-auth-file", config.SMTPAuthFile, "A password file for SMTP authentication")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAcceptAny, "smtp-auth-accept-any", false, "Accept any SMTP username and password, including none")
	rootCmd.Flags().StringVar(&config.SMTPSSLCert, "smtp-ssl-cert", config.SMTPSSLCert, "SSL certificate for SMTP - requires smtp-ssl-key")
	rootCmd.Flags().StringVar(&config.SMTPSSLKey, "smtp-ssl-key", config.SMTPSSLKey, "SSL key for SMTP - requires smtp-ssl-cert")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAllowInsecure, "smtp-auth-allow-insecure", false, "Enable insecure PLAIN & LOGIN authentication")
	rootCmd.Flags().StringVarP(&config.SMTPCLITags, "tag", "t", config.SMTPCLITags, "Tag new messages matching filters")

	rootCmd.Flags().BoolVarP(&config.QuietLogging, "quiet", "q", false, "Quiet logging (errors only)")
	rootCmd.Flags().BoolVarP(&config.VerboseLogging, "verbose", "v", false, "Verbose logging")

	// deprecated 2022/08/06
	rootCmd.Flags().StringVarP(&config.UIAuthFile, "auth-file", "a", config.UIAuthFile, "A password file for web UI authentication")
	rootCmd.Flags().StringVar(&config.UISSLCert, "ssl-cert", config.UISSLCert, "SSL certificate - requires ssl-key")
	rootCmd.Flags().StringVar(&config.UISSLKey, "ssl-key", config.UISSLKey, "SSL key - requires ssl-cert")
	rootCmd.Flags().Lookup("auth-file").Hidden = true
	rootCmd.Flags().Lookup("auth-file").Deprecated = "use --ui-auth-file"
	rootCmd.Flags().Lookup("ssl-cert").Hidden = true
	rootCmd.Flags().Lookup("ssl-cert").Deprecated = "use --ui-ssl-cert"
	rootCmd.Flags().Lookup("ssl-key").Hidden = true
	rootCmd.Flags().Lookup("ssl-key").Deprecated = "use --ui-ssl-key"

	// deprecated 2022/08/30
	rootCmd.Flags().StringVar(&config.DataFile, "data", config.DataFile, "Database file to store persistent data")
	rootCmd.Flags().Lookup("data").Hidden = true
	rootCmd.Flags().Lookup("data").Deprecated = "use --db-file"
}

// Load settings from environment
func initConfigFromEnv() {
	// defaults from envars if provided
	if len(os.Getenv("MP_DATA_FILE")) > 0 {
		config.DataFile = os.Getenv("MP_DATA_FILE")
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
	if len(os.Getenv("MP_TAG")) > 0 {
		config.SMTPCLITags = os.Getenv("MP_TAG")
	}

	// UI
	if len(os.Getenv("MP_UI_AUTH_FILE")) > 0 {
		config.UIAuthFile = os.Getenv("MP_UI_AUTH_FILE")
	}
	if len(os.Getenv("MP_UI_SSL_CERT")) > 0 {
		config.UISSLCert = os.Getenv("MP_UI_SSL_CERT")
	}
	if len(os.Getenv("MP_UI_SSL_KEY")) > 0 {
		config.UISSLKey = os.Getenv("MP_UI_SSL_KEY")
	}

	// SMTP
	if len(os.Getenv("MP_SMTP_AUTH_FILE")) > 0 {
		config.SMTPAuthFile = os.Getenv("MP_SMTP_AUTH_FILE")
	}
	if len(os.Getenv("MP_SMTP_SSL_CERT")) > 0 {
		config.SMTPSSLCert = os.Getenv("MP_SMTP_SSL_CERT")
	}
	if len(os.Getenv("MP_SMTP_SSL_KEY")) > 0 {
		config.SMTPSSLKey = os.Getenv("MP_SMTP_SSL_KEY")
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ACCEPT_ANY") {
		config.SMTPAuthAcceptAny = true
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ALLOW_INSECURE") {
		config.SMTPAuthAllowInsecure = true
	}

	if len(os.Getenv("MP_WEBROOT")) > 0 {
		config.Webroot = os.Getenv("MP_WEBROOT")
	}
	if getEnabledFromEnv("MP_USE_MESSAGE_DATES") {
		config.UseMessageDates = true
	}
	if getEnabledFromEnv("MP_USE_MESSAGE_DATES") {
		config.UseMessageDates = true
	}
	if getEnabledFromEnv("MP_QUIET") {
		config.QuietLogging = true
	}
	if getEnabledFromEnv("MP_VERBOSE") {
		config.VerboseLogging = true
	}
}

func getEnabledFromEnv(k string) bool {
	if len(os.Getenv(k)) > 0 {
		v := strings.ToLower(os.Getenv(k))
		return v == "1" || v == "true" || v == "yes"
	}

	return false
}
