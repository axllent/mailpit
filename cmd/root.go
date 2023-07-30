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

	// load and warn deprecated ENV vars
	initDeprecatedConfigFromEnv()

	// load ENV vars
	initConfigFromEnv()

	rootCmd.Flags().StringVarP(&config.DataFile, "db-file", "d", config.DataFile, "Database file to store persistent data")
	rootCmd.Flags().StringVarP(&config.SMTPListen, "smtp", "s", config.SMTPListen, "SMTP bind interface and port")
	rootCmd.Flags().StringVarP(&config.HTTPListen, "listen", "l", config.HTTPListen, "HTTP bind interface and port for UI")
	rootCmd.Flags().IntVarP(&config.MaxMessages, "max", "m", config.MaxMessages, "Max number of messages to store")
	rootCmd.Flags().StringVar(&config.Webroot, "webroot", config.Webroot, "Set the webroot for web UI & API")
	rootCmd.Flags().StringVar(&server.AccessControlAllowOrigin, "api-cors", server.AccessControlAllowOrigin, "Set API CORS Access-Control-Allow-Origin header")
	rootCmd.Flags().BoolVar(&config.UseMessageDates, "use-message-dates", config.UseMessageDates, "Use message dates as the received dates")
	rootCmd.Flags().BoolVar(&config.IgnoreDuplicateIDs, "ignore-duplicate-ids", config.IgnoreDuplicateIDs, "Ignore duplicate messages (by Message-Id)")
	rootCmd.Flags().BoolVar(&config.DisableHTMLCheck, "disable-html-check", config.DisableHTMLCheck, "Disable the HTML check functionality (web UI & API)")
	rootCmd.Flags().BoolVar(&config.BlockRemoteCSSAndFonts, "block-remote-css-and-fonts", config.BlockRemoteCSSAndFonts, "Block access to remote CSS & fonts")

	rootCmd.Flags().StringVar(&config.UIAuthFile, "ui-auth-file", config.UIAuthFile, "A password file for web UI authentication")
	rootCmd.Flags().StringVar(&config.UITLSCert, "ui-tls-cert", config.UITLSCert, "TLS certificate for web UI (HTTPS) - requires ui-tls-key")
	rootCmd.Flags().StringVar(&config.UITLSKey, "ui-tls-key", config.UITLSKey, "TLS key for web UI (HTTPS) - requires ui-tls-cert")

	rootCmd.Flags().StringVar(&config.SMTPAuthFile, "smtp-auth-file", config.SMTPAuthFile, "A password file for SMTP authentication")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAcceptAny, "smtp-auth-accept-any", config.SMTPAuthAcceptAny, "Accept any SMTP username and password, including none")
	rootCmd.Flags().StringVar(&config.SMTPTLSCert, "smtp-tls-cert", config.SMTPTLSCert, "TLS certificate for SMTP (STARTTLS) - requires smtp-tls-key")
	rootCmd.Flags().StringVar(&config.SMTPTLSKey, "smtp-tls-key", config.SMTPTLSKey, "TLS key for SMTP (STARTTLS) - requires smtp-tls-cert")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAllowInsecure, "smtp-auth-allow-insecure", config.SMTPAuthAllowInsecure, "Enable insecure PLAIN & LOGIN authentication")

	rootCmd.Flags().StringVar(&config.SMTPRelayConfigFile, "smtp-relay-config", config.SMTPRelayConfigFile, "SMTP configuration file to allow releasing messages")
	rootCmd.Flags().BoolVar(&config.SMTPRelayAllIncoming, "smtp-relay-all", config.SMTPRelayAllIncoming, "Relay all incoming messages via external SMTP server (caution!)")

	rootCmd.Flags().StringVarP(&config.SMTPCLITags, "tag", "t", config.SMTPCLITags, "Tag new messages matching filters")
	rootCmd.Flags().BoolVarP(&logger.QuietLogging, "quiet", "q", logger.QuietLogging, "Quiet logging (errors only)")
	rootCmd.Flags().BoolVarP(&logger.VerboseLogging, "verbose", "v", logger.VerboseLogging, "Verbose logging")

	// deprecated flags 2022/08/06
	rootCmd.Flags().StringVarP(&config.UIAuthFile, "auth-file", "a", config.UIAuthFile, "A password file for web UI authentication")
	rootCmd.Flags().StringVar(&config.UITLSCert, "ssl-cert", config.UITLSCert, "SSL certificate - requires ssl-key")
	rootCmd.Flags().StringVar(&config.UITLSKey, "ssl-key", config.UITLSKey, "SSL key - requires ssl-cert")
	rootCmd.Flags().Lookup("auth-file").Hidden = true
	rootCmd.Flags().Lookup("auth-file").Deprecated = "use --ui-auth-file"
	rootCmd.Flags().Lookup("ssl-cert").Hidden = true
	rootCmd.Flags().Lookup("ssl-cert").Deprecated = "use --ui-tls-cert"
	rootCmd.Flags().Lookup("ssl-key").Hidden = true
	rootCmd.Flags().Lookup("ssl-key").Deprecated = "use --ui-tls-key"

	// deprecated flags 2022/08/30
	rootCmd.Flags().StringVar(&config.DataFile, "data", config.DataFile, "Database file to store persistent data")
	rootCmd.Flags().Lookup("data").Hidden = true
	rootCmd.Flags().Lookup("data").Deprecated = "use --db-file"

	// deprecated flags 2023/03/12
	rootCmd.Flags().StringVar(&config.UITLSCert, "ui-ssl-cert", config.UITLSCert, "SSL certificate for web UI - requires ui-ssl-key")
	rootCmd.Flags().StringVar(&config.UITLSKey, "ui-ssl-key", config.UITLSKey, "SSL key for web UI - requires ui-ssl-cert")
	rootCmd.Flags().StringVar(&config.SMTPTLSCert, "smtp-ssl-cert", config.SMTPTLSCert, "SSL certificate for SMTP - requires smtp-ssl-key")
	rootCmd.Flags().StringVar(&config.SMTPTLSKey, "smtp-ssl-key", config.SMTPTLSKey, "SSL key for SMTP - requires smtp-ssl-cert")
	rootCmd.Flags().Lookup("ui-ssl-cert").Hidden = true
	rootCmd.Flags().Lookup("ui-ssl-cert").Deprecated = "use --ui-tls-cert"
	rootCmd.Flags().Lookup("ui-ssl-key").Hidden = true
	rootCmd.Flags().Lookup("ui-ssl-key").Deprecated = "use --ui-tls-key"
	rootCmd.Flags().Lookup("smtp-ssl-cert").Hidden = true
	rootCmd.Flags().Lookup("smtp-ssl-cert").Deprecated = "use --smtp-tls-cert"
	rootCmd.Flags().Lookup("smtp-ssl-key").Hidden = true
	rootCmd.Flags().Lookup("smtp-ssl-key").Deprecated = "use --smtp-tls-key"
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
	if len(os.Getenv("MP_UI_TLS_CERT")) > 0 {
		config.UITLSCert = os.Getenv("MP_UI_TLS_CERT")
	}
	if len(os.Getenv("MP_UI_TLS_KEY")) > 0 {
		config.UITLSKey = os.Getenv("MP_UI_TLS_KEY")
	}

	// SMTP
	if len(os.Getenv("MP_SMTP_AUTH_FILE")) > 0 {
		config.SMTPAuthFile = os.Getenv("MP_SMTP_AUTH_FILE")
	}
	if len(os.Getenv("MP_SMTP_TLS_CERT")) > 0 {
		config.SMTPTLSCert = os.Getenv("MP_SMTP_TLS_CERT")
	}
	if len(os.Getenv("MP_SMTP_TLS_KEY")) > 0 {
		config.SMTPTLSKey = os.Getenv("MP_SMTP_TLS_KEY")
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ACCEPT_ANY") {
		config.SMTPAuthAcceptAny = true
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ALLOW_INSECURE") {
		config.SMTPAuthAllowInsecure = true
	}

	// Relay server config
	if len(os.Getenv("MP_SMTP_RELAY_CONFIG")) > 0 {
		config.SMTPRelayConfigFile = os.Getenv("MP_SMTP_RELAY_CONFIG")
	}
	if getEnabledFromEnv("MP_SMTP_RELAY_ALL") {
		config.SMTPRelayAllIncoming = true
	}

	// Misc options
	if len(os.Getenv("MP_WEBROOT")) > 0 {
		config.Webroot = os.Getenv("MP_WEBROOT")
	}
	if len(os.Getenv("MP_API_CORS")) > 0 {
		server.AccessControlAllowOrigin = os.Getenv("MP_API_CORS")
	}
	if getEnabledFromEnv("MP_USE_MESSAGE_DATES") {
		config.UseMessageDates = true
	}
	if getEnabledFromEnv("MP_IGNORE_DUPLICATE_IDS") {
		config.IgnoreDuplicateIDs = true
	}
	if getEnabledFromEnv("MP_DISABLE_HTML_CHECK") {
		config.DisableHTMLCheck = true
	}
	if getEnabledFromEnv("MP_BLOCK_REMOTE_CSS_AND_FONTS") {
		config.BlockRemoteCSSAndFonts = true
	}
	if getEnabledFromEnv("MP_QUIET") {
		logger.QuietLogging = true
	}
	if getEnabledFromEnv("MP_VERBOSE") {
		logger.VerboseLogging = true
	}
}

// load deprecated settings from environment and warn
func initDeprecatedConfigFromEnv() {
	// deprecated 2022/08/06
	if len(os.Getenv("MP_AUTH_FILE")) > 0 {
		fmt.Println("ENV MP_AUTH_FILE has been deprecated, use MP_UI_AUTH_FILE")
		config.UIAuthFile = os.Getenv("MP_AUTH_FILE")
	}
	// deprecated 2022/08/06
	if len(os.Getenv("MP_SSL_CERT")) > 0 {
		fmt.Println("ENV MP_SSL_CERT has been deprecated, use MP_UI_TLS_CERT")
		config.UITLSCert = os.Getenv("MP_SSL_CERT")
	}
	// deprecated 2022/08/06
	if len(os.Getenv("MP_SSL_KEY")) > 0 {
		fmt.Println("ENV MP_SSL_KEY has been deprecated, use MP_UI_TLS_KEY")
		config.UITLSKey = os.Getenv("MP_TLS_KEY")
	}
	// deprecated 2022/08/28
	if len(os.Getenv("MP_DATA_DIR")) > 0 {
		fmt.Println("ENV MP_DATA_DIR has been deprecated, use MP_DATA_FILE")
		config.DataFile = os.Getenv("MP_DATA_DIR")
	}
	// deprecated 2023/03/12
	if len(os.Getenv("MP_UI_SSL_CERT")) > 0 {
		fmt.Println("ENV MP_UI_SSL_CERT has been deprecated, use MP_UI_TLS_CERT")
		config.UITLSCert = os.Getenv("MP_UI_SSL_CERT")
	}
	if len(os.Getenv("MP_UI_SSL_KEY")) > 0 {
		fmt.Println("ENV MP_UI_SSL_KEY has been deprecated, use MP_UI_TLS_KEY")
		config.UITLSKey = os.Getenv("MP_UI_SSL_KEY")
	}
	if len(os.Getenv("MP_SMTP_SSL_CERT")) > 0 {
		fmt.Println("ENV MP_SMTP_CERT has been deprecated, use MP_SMTP_TLS_CERT")
		config.SMTPTLSCert = os.Getenv("MP_SMTP_SSL_CERT")
	}
	if len(os.Getenv("MP_SMTP_SSL_KEY")) > 0 {
		fmt.Println("ENV MP_SMTP_KEY has been deprecated, use MP_SMTP_TLS_KEY")
		config.SMTPTLSKey = os.Getenv("MP_SMTP_SMTP_KEY")
	}
}

// Wrapper to get a boolean from an environment variable
func getEnabledFromEnv(k string) bool {
	if len(os.Getenv(k)) > 0 {
		v := strings.ToLower(os.Getenv(k))
		return v == "1" || v == "true" || v == "yes"
	}

	return false
}
