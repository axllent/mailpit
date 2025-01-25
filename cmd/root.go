// Package cmd is the main application
package cmd

import (
	"os"
	"strconv"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/smtpd"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server"
	"github.com/axllent/mailpit/server/webhook"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mailpit",
	Short: "Mailpit is an email testing tool for developers",
	Long: `Mailpit is an email testing tool for developers.

It acts as an SMTP server, and provides a web interface to view all captured emails.

Documentation:
  https://github.com/axllent/mailpit
  https://mailpit.axllent.org/docs/`,
	Run: func(_ *cobra.Command, _ []string) {
		if err := config.VerifyConfig(); err != nil {
			logger.Log().Error(err.Error())
			os.Exit(1)
		}
		if err := storage.InitDB(); err != nil {
			logger.Log().Fatal(err.Error())
			os.Exit(1)
		}

		go server.Listen()

		if err := smtpd.Listen(); err != nil {
			storage.Close()
			logger.Log().Fatal(err.Error())
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

	// load environment variables
	initConfigFromEnv()

	rootCmd.Flags().StringVarP(&config.Database, "database", "d", config.Database, "Database to store persistent data")
	rootCmd.Flags().StringVar(&config.Label, "label", config.Label, "Optional label identify this Mailpit instance")
	rootCmd.Flags().StringVar(&config.TenantID, "tenant-id", config.TenantID, "Database tenant ID to isolate data")
	rootCmd.Flags().IntVarP(&config.MaxMessages, "max", "m", config.MaxMessages, "Max number of messages to store")
	rootCmd.Flags().StringVar(&config.MaxAge, "max-age", config.MaxAge, "Max age of messages in either (h)ours or (d)ays (eg: 3d)")
	rootCmd.Flags().BoolVar(&config.UseMessageDates, "use-message-dates", config.UseMessageDates, "Use message dates as the received dates")
	rootCmd.Flags().BoolVar(&config.IgnoreDuplicateIDs, "ignore-duplicate-ids", config.IgnoreDuplicateIDs, "Ignore duplicate messages (by Message-Id)")
	rootCmd.Flags().StringVar(&logger.LogFile, "log-file", logger.LogFile, "Log output to file instead of stdout")
	rootCmd.Flags().BoolVarP(&logger.QuietLogging, "quiet", "q", logger.QuietLogging, "Quiet logging (errors only)")
	rootCmd.Flags().BoolVarP(&logger.VerboseLogging, "verbose", "v", logger.VerboseLogging, "Verbose logging")

	// Web UI / API
	rootCmd.Flags().StringVarP(&config.HTTPListen, "listen", "l", config.HTTPListen, "HTTP bind interface & port for UI")
	rootCmd.Flags().StringVar(&config.Webroot, "webroot", config.Webroot, "Set the webroot for web UI & API")
	rootCmd.Flags().StringVar(&config.UIAuthFile, "ui-auth-file", config.UIAuthFile, "A password file for web UI & API authentication")
	rootCmd.Flags().StringVar(&config.UITLSCert, "ui-tls-cert", config.UITLSCert, "TLS certificate for web UI (HTTPS) - requires ui-tls-key")
	rootCmd.Flags().StringVar(&config.UITLSKey, "ui-tls-key", config.UITLSKey, "TLS key for web UI (HTTPS) - requires ui-tls-cert")
	rootCmd.Flags().StringVar(&server.AccessControlAllowOrigin, "api-cors", server.AccessControlAllowOrigin, "Set API CORS Access-Control-Allow-Origin header")
	rootCmd.Flags().BoolVar(&config.BlockRemoteCSSAndFonts, "block-remote-css-and-fonts", config.BlockRemoteCSSAndFonts, "Block access to remote CSS & fonts")
	rootCmd.Flags().StringVar(&config.EnableSpamAssassin, "enable-spamassassin", config.EnableSpamAssassin, "Enable integration with SpamAssassin")
	rootCmd.Flags().BoolVar(&config.AllowUntrustedTLS, "allow-untrusted-tls", config.AllowUntrustedTLS, "Do not verify HTTPS certificates (link checker & screenshots)")

	// SMTP server
	rootCmd.Flags().StringVarP(&config.SMTPListen, "smtp", "s", config.SMTPListen, "SMTP bind interface and port")
	rootCmd.Flags().StringVar(&config.SMTPAuthFile, "smtp-auth-file", config.SMTPAuthFile, "A password file for SMTP authentication")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAcceptAny, "smtp-auth-accept-any", config.SMTPAuthAcceptAny, "Accept any SMTP username and password, including none")
	rootCmd.Flags().StringVar(&config.SMTPTLSCert, "smtp-tls-cert", config.SMTPTLSCert, "TLS certificate for SMTP (STARTTLS) - requires smtp-tls-key")
	rootCmd.Flags().StringVar(&config.SMTPTLSKey, "smtp-tls-key", config.SMTPTLSKey, "TLS key for SMTP (STARTTLS) - requires smtp-tls-cert")
	rootCmd.Flags().BoolVar(&config.SMTPRequireSTARTTLS, "smtp-require-starttls", config.SMTPRequireSTARTTLS, "Require SMTP client use STARTTLS")
	rootCmd.Flags().BoolVar(&config.SMTPRequireTLS, "smtp-require-tls", config.SMTPRequireTLS, "Require client use SSL/TLS")
	rootCmd.Flags().BoolVar(&config.SMTPAuthAllowInsecure, "smtp-auth-allow-insecure", config.SMTPAuthAllowInsecure, "Allow insecure PLAIN & LOGIN SMTP authentication")
	rootCmd.Flags().BoolVar(&config.SMTPStrictRFCHeaders, "smtp-strict-rfc-headers", config.SMTPStrictRFCHeaders, "Return SMTP error if message headers contain <CR><CR><LF>")
	rootCmd.Flags().IntVar(&config.SMTPMaxRecipients, "smtp-max-recipients", config.SMTPMaxRecipients, "Maximum SMTP recipients allowed")
	rootCmd.Flags().StringVar(&config.SMTPAllowedRecipients, "smtp-allowed-recipients", config.SMTPAllowedRecipients, "Only allow SMTP recipients matching a regular expression (default allow all)")
	rootCmd.Flags().BoolVar(&smtpd.DisableReverseDNS, "smtp-disable-rdns", smtpd.DisableReverseDNS, "Disable SMTP reverse DNS lookups")

	// SMTP relay
	rootCmd.Flags().StringVar(&config.SMTPRelayConfigFile, "smtp-relay-config", config.SMTPRelayConfigFile, "SMTP relay configuration file to allow releasing messages")
	rootCmd.Flags().BoolVar(&config.SMTPRelayAll, "smtp-relay-all", config.SMTPRelayAll, "Auto-relay all new messages via external SMTP server (caution!)")
	rootCmd.Flags().StringVar(&config.SMTPRelayMatching, "smtp-relay-matching", config.SMTPRelayMatching, "Auto-relay new messages to only matching recipients (regular expression)")

	// SMTP forwarding
	rootCmd.Flags().StringVar(&config.SMTPForwardConfigFile, "smtp-forward-config", config.SMTPForwardConfigFile, "SMTP forwarding configuration file for all messages")

	// Chaos
	rootCmd.Flags().BoolVar(&chaos.Enabled, "enable-chaos", chaos.Enabled, "Enable Chaos functionality (API / web UI)")
	rootCmd.Flags().StringVar(&config.ChaosTriggers, "chaos-triggers", config.ChaosTriggers, "Enable Chaos & set the triggers for SMTP server")

	// POP3 server
	rootCmd.Flags().StringVar(&config.POP3Listen, "pop3", config.POP3Listen, "POP3 server bind interface and port")
	rootCmd.Flags().StringVar(&config.POP3AuthFile, "pop3-auth-file", config.POP3AuthFile, "A password file for POP3 server authentication (enables POP3 server)")
	rootCmd.Flags().StringVar(&config.POP3TLSCert, "pop3-tls-cert", config.POP3TLSCert, "Optional TLS certificate for POP3 server - requires pop3-tls-key")
	rootCmd.Flags().StringVar(&config.POP3TLSKey, "pop3-tls-key", config.POP3TLSKey, "Optional TLS key for POP3 server - requires pop3-tls-cert")

	// Tagging
	rootCmd.Flags().StringVarP(&config.CLITagsArg, "tag", "t", config.CLITagsArg, "Tag new messages matching filters")
	rootCmd.Flags().StringVar(&config.TagsConfig, "tags-config", config.TagsConfig, "Load tags filters from yaml configuration file")
	rootCmd.Flags().BoolVar(&tools.TagsTitleCase, "tags-title-case", tools.TagsTitleCase, "TitleCase new tags generated from plus-addresses and X-Tags")
	rootCmd.Flags().StringVar(&config.TagsDisable, "tags-disable", config.TagsDisable, "Disable auto-tagging, comma separated (eg: plus-addresses,x-tags)")

	// Webhook
	rootCmd.Flags().StringVar(&config.WebhookURL, "webhook-url", config.WebhookURL, "Send a webhook request for new messages")
	rootCmd.Flags().IntVar(&webhook.RateLimit, "webhook-limit", webhook.RateLimit, "Limit webhook requests per second")

	// DEPRECATED FLAG 2024/04/12 - but will not be removed to maintain backwards compatibility
	rootCmd.Flags().StringVar(&config.Database, "db-file", config.Database, "Database file to store persistent data")
	rootCmd.Flags().Lookup("db-file").Hidden = true

	// DEPRECATED FLAGS 2023/03/12
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

	// DEPRECATED FLAGS 2024/03/16
	rootCmd.Flags().BoolVar(&config.SMTPRequireSTARTTLS, "smtp-tls-required", config.SMTPRequireSTARTTLS, "smtp-require-starttls")
	rootCmd.Flags().Lookup("smtp-tls-required").Hidden = true
	rootCmd.Flags().Lookup("smtp-tls-required").Deprecated = "use --smtp-require-starttls"

	// DEPRECATED FLAG 2024/04/13 - no longer used
	rootCmd.Flags().BoolVar(&config.DisableHTMLCheck, "disable-html-check", config.DisableHTMLCheck, "Disable the HTML check functionality (web UI & API)")
	rootCmd.Flags().Lookup("disable-html-check").Hidden = true
}

// Load settings from environment
func initConfigFromEnv() {
	// General
	if len(os.Getenv("MP_DATABASE")) > 0 {
		config.Database = os.Getenv("MP_DATABASE")
	}

	config.TenantID = os.Getenv("MP_TENANT_ID")

	config.Label = os.Getenv("MP_LABEL")

	if len(os.Getenv("MP_MAX_MESSAGES")) > 0 {
		config.MaxMessages, _ = strconv.Atoi(os.Getenv("MP_MAX_MESSAGES"))
	}
	if len(os.Getenv("MP_MAX_AGE")) > 0 {
		config.MaxAge = os.Getenv("MP_MAX_AGE")
	}
	if getEnabledFromEnv("MP_USE_MESSAGE_DATES") {
		config.UseMessageDates = true
	}
	if getEnabledFromEnv("MP_IGNORE_DUPLICATE_IDS") {
		config.IgnoreDuplicateIDs = true
	}
	if len(os.Getenv("MP_LOG_FILE")) > 0 {
		logger.LogFile = os.Getenv("MP_LOG_FILE")
	}
	if getEnabledFromEnv("MP_QUIET") {
		logger.QuietLogging = true
	}
	if getEnabledFromEnv("MP_VERBOSE") {
		logger.VerboseLogging = true
	}

	// Web UI & API
	if len(os.Getenv("MP_UI_BIND_ADDR")) > 0 {
		config.HTTPListen = os.Getenv("MP_UI_BIND_ADDR")
	}
	if len(os.Getenv("MP_WEBROOT")) > 0 {
		config.Webroot = os.Getenv("MP_WEBROOT")
	}
	config.UIAuthFile = os.Getenv("MP_UI_AUTH_FILE")
	if err := auth.SetUIAuth(os.Getenv("MP_UI_AUTH")); err != nil {
		logger.Log().Error(err.Error())
	}
	config.UITLSCert = os.Getenv("MP_UI_TLS_CERT")
	config.UITLSKey = os.Getenv("MP_UI_TLS_KEY")
	if len(os.Getenv("MP_API_CORS")) > 0 {
		server.AccessControlAllowOrigin = os.Getenv("MP_API_CORS")
	}
	if getEnabledFromEnv("MP_BLOCK_REMOTE_CSS_AND_FONTS") {
		config.BlockRemoteCSSAndFonts = true
	}
	if len(os.Getenv("MP_ENABLE_SPAMASSASSIN")) > 0 {
		config.EnableSpamAssassin = os.Getenv("MP_ENABLE_SPAMASSASSIN")
	}
	if getEnabledFromEnv("MP_ALLOW_UNTRUSTED_TLS") {
		config.AllowUntrustedTLS = true
	}

	// SMTP server
	if len(os.Getenv("MP_SMTP_BIND_ADDR")) > 0 {
		config.SMTPListen = os.Getenv("MP_SMTP_BIND_ADDR")
	}
	config.SMTPAuthFile = os.Getenv("MP_SMTP_AUTH_FILE")
	if err := auth.SetSMTPAuth(os.Getenv("MP_SMTP_AUTH")); err != nil {
		logger.Log().Error(err.Error())
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ACCEPT_ANY") {
		config.SMTPAuthAcceptAny = true
	}
	config.SMTPTLSCert = os.Getenv("MP_SMTP_TLS_CERT")
	config.SMTPTLSKey = os.Getenv("MP_SMTP_TLS_KEY")
	if getEnabledFromEnv("MP_SMTP_REQUIRE_STARTTLS") {
		config.SMTPRequireSTARTTLS = true
	}
	if getEnabledFromEnv("MP_SMTP_REQUIRE_TLS") {
		config.SMTPRequireTLS = true
	}
	if getEnabledFromEnv("MP_SMTP_AUTH_ALLOW_INSECURE") {
		config.SMTPAuthAllowInsecure = true
	}
	if getEnabledFromEnv("MP_SMTP_STRICT_RFC_HEADERS") {
		config.SMTPStrictRFCHeaders = true
	}
	if len(os.Getenv("MP_SMTP_MAX_RECIPIENTS")) > 0 {
		config.SMTPMaxRecipients, _ = strconv.Atoi(os.Getenv("MP_SMTP_MAX_RECIPIENTS"))
	}
	if len(os.Getenv("MP_SMTP_ALLOWED_RECIPIENTS")) > 0 {
		config.SMTPAllowedRecipients = os.Getenv("MP_SMTP_ALLOWED_RECIPIENTS")
	}
	if getEnabledFromEnv("MP_SMTP_DISABLE_RDNS") {
		smtpd.DisableReverseDNS = true
	}

	// SMTP relay
	config.SMTPRelayConfigFile = os.Getenv("MP_SMTP_RELAY_CONFIG")
	if getEnabledFromEnv("MP_SMTP_RELAY_ALL") {
		config.SMTPRelayAll = true
	}
	config.SMTPRelayMatching = os.Getenv("MP_SMTP_RELAY_MATCHING")
	config.SMTPRelayConfig = config.SMTPRelayConfigStruct{}
	config.SMTPRelayConfig.Host = os.Getenv("MP_SMTP_RELAY_HOST")
	if len(os.Getenv("MP_SMTP_RELAY_PORT")) > 0 {
		config.SMTPRelayConfig.Port, _ = strconv.Atoi(os.Getenv("MP_SMTP_RELAY_PORT"))
	}
	config.SMTPRelayConfig.STARTTLS = getEnabledFromEnv("MP_SMTP_RELAY_STARTTLS")
	config.SMTPRelayConfig.AllowInsecure = getEnabledFromEnv("MP_SMTP_RELAY_ALLOW_INSECURE")
	config.SMTPRelayConfig.Auth = os.Getenv("MP_SMTP_RELAY_AUTH")
	config.SMTPRelayConfig.Username = os.Getenv("MP_SMTP_RELAY_USERNAME")
	config.SMTPRelayConfig.Password = os.Getenv("MP_SMTP_RELAY_PASSWORD")
	config.SMTPRelayConfig.Secret = os.Getenv("MP_SMTP_RELAY_SECRET")
	config.SMTPRelayConfig.ReturnPath = os.Getenv("MP_SMTP_RELAY_RETURN_PATH")
	config.SMTPRelayConfig.OverrideFrom = os.Getenv("MP_SMTP_RELAY_OVERRIDE_FROM")
	config.SMTPRelayConfig.AllowedRecipients = os.Getenv("MP_SMTP_RELAY_ALLOWED_RECIPIENTS")
	config.SMTPRelayConfig.BlockedRecipients = os.Getenv("MP_SMTP_RELAY_BLOCKED_RECIPIENTS")

	// SMTP forwarding
	config.SMTPForwardConfigFile = os.Getenv("MP_SMTP_FORWARD_CONFIG")
	config.SMTPForwardConfig = config.SMTPForwardConfigStruct{}
	config.SMTPForwardConfig.Host = os.Getenv("MP_SMTP_FORWARD_HOST")
	if len(os.Getenv("MP_SMTP_FORWARD_PORT")) > 0 {
		config.SMTPForwardConfig.Port, _ = strconv.Atoi(os.Getenv("MP_SMTP_FORWARD_PORT"))
	}
	config.SMTPForwardConfig.STARTTLS = getEnabledFromEnv("MP_SMTP_FORWARD_STARTTLS")
	config.SMTPForwardConfig.AllowInsecure = getEnabledFromEnv("MP_SMTP_FORWARD_ALLOW_INSECURE")
	config.SMTPForwardConfig.Auth = os.Getenv("MP_SMTP_FORWARD_AUTH")
	config.SMTPForwardConfig.Username = os.Getenv("MP_SMTP_FORWARD_USERNAME")
	config.SMTPForwardConfig.Password = os.Getenv("MP_SMTP_FORWARD_PASSWORD")
	config.SMTPForwardConfig.Secret = os.Getenv("MP_SMTP_FORWARD_SECRET")
	config.SMTPForwardConfig.ReturnPath = os.Getenv("MP_SMTP_FORWARD_RETURN_PATH")
	config.SMTPForwardConfig.OverrideFrom = os.Getenv("MP_SMTP_FORWARD_OVERRIDE_FROM")
	config.SMTPForwardConfig.To = os.Getenv("MP_SMTP_FORWARD_TO")

	// Chaos
	chaos.Enabled = getEnabledFromEnv("MP_ENABLE_CHAOS")
	config.ChaosTriggers = os.Getenv("MP_CHAOS_TRIGGERS")

	// POP3 server
	if len(os.Getenv("MP_POP3_BIND_ADDR")) > 0 {
		config.POP3Listen = os.Getenv("MP_POP3_BIND_ADDR")
	}
	config.POP3AuthFile = os.Getenv("MP_POP3_AUTH_FILE")
	if err := auth.SetPOP3Auth(os.Getenv("MP_POP3_AUTH")); err != nil {
		logger.Log().Error(err.Error())
	}
	config.POP3TLSCert = os.Getenv("MP_POP3_TLS_CERT")
	config.POP3TLSKey = os.Getenv("MP_POP3_TLS_KEY")

	// Tagging
	config.CLITagsArg = os.Getenv("MP_TAG")
	config.TagsConfig = os.Getenv("MP_TAGS_CONFIG")
	tools.TagsTitleCase = getEnabledFromEnv("MP_TAGS_TITLE_CASE")
	config.TagsDisable = os.Getenv("MP_TAGS_DISABLE")

	// Webhook
	if len(os.Getenv("MP_WEBHOOK_URL")) > 0 {
		config.WebhookURL = os.Getenv("MP_WEBHOOK_URL")
	}
	if len(os.Getenv("MP_WEBHOOK_LIMIT")) > 0 {
		webhook.RateLimit, _ = strconv.Atoi(os.Getenv("MP_WEBHOOK_LIMIT"))
	}

	// Demo mode
	config.DemoMode = getEnabledFromEnv("MP_DEMO_MODE")
}

// load deprecated settings from environment and warn
func initDeprecatedConfigFromEnv() {
	// deprecated 2024/04/12 - but will not be removed to maintain backwards compatibility
	if len(os.Getenv("MP_DATA_FILE")) > 0 {
		config.Database = os.Getenv("MP_DATA_FILE")
	}

	// deprecated 2023/03/12
	if len(os.Getenv("MP_UI_SSL_CERT")) > 0 {
		logger.Log().Warn("ENV MP_UI_SSL_CERT has been deprecated, use MP_UI_TLS_CERT")
		config.UITLSCert = os.Getenv("MP_UI_SSL_CERT")
	}
	// deprecated 2023/03/12
	if len(os.Getenv("MP_UI_SSL_KEY")) > 0 {
		logger.Log().Warn("ENV MP_UI_SSL_KEY has been deprecated, use MP_UI_TLS_KEY")
		config.UITLSKey = os.Getenv("MP_UI_SSL_KEY")
	}
	// deprecated 2023/03/12
	if len(os.Getenv("MP_SMTP_SSL_CERT")) > 0 {
		logger.Log().Warn("ENV MP_SMTP_CERT has been deprecated, use MP_SMTP_TLS_CERT")
		config.SMTPTLSCert = os.Getenv("MP_SMTP_SSL_CERT")
	}
	// deprecated 2023/03/12
	if len(os.Getenv("MP_SMTP_SSL_KEY")) > 0 {
		logger.Log().Warn("ENV MP_SMTP_KEY has been deprecated, use MP_SMTP_TLS_KEY")
		config.SMTPTLSKey = os.Getenv("MP_SMTP_SMTP_KEY")
	}
	// deprecated 2023/12/10
	if getEnabledFromEnv("MP_STRICT_RFC_HEADERS") {
		logger.Log().Warn("ENV MP_STRICT_RFC_HEADERS has been deprecated, use MP_SMTP_STRICT_RFC_HEADERS")
		config.SMTPStrictRFCHeaders = true
	}
	// deprecated 2024/03.16
	if getEnabledFromEnv("MP_SMTP_TLS_REQUIRED") {
		logger.Log().Warn("ENV MP_SMTP_TLS_REQUIRED has been deprecated, use MP_SMTP_REQUIRE_STARTTLS")
		config.SMTPRequireSTARTTLS = true
	}
	if getEnabledFromEnv("MP_DISABLE_HTML_CHECK") {
		logger.Log().Warn("ENV MP_DISABLE_HTML_CHECK has been deprecated and is no longer used")
		config.DisableHTMLCheck = true
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
