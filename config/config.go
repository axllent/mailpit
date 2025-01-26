// Package config handles the application configuration
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/smtpd/chaos"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/tools"
)

var (
	// SMTPListen to listen on <interface>:<port>
	SMTPListen = "[::]:1025"

	// HTTPListen to listen on <interface>:<port>
	HTTPListen = "[::]:8025"

	// Database for mail (optional)
	Database string

	// TenantID is an optional prefix to be applied to all database tables,
	// allowing multiple isolated instances of Mailpit to share a database.
	TenantID string

	// Label to identify this Mailpit instance (optional).
	// This gets applied to web UI, SMTP and optional POP3 server.
	Label string

	// MaxMessages is the maximum number of messages a mailbox can have (auto-pruned every minute)
	MaxMessages = 500

	// MaxAge is the maximum age of messages (auto-pruned every hour).
	// Value can be either <int>h for hours or <int>d for days
	MaxAge string

	// MaxAgeInHours is the maximum age of messages in hours, set with parseMaxAge() using MaxAge value
	MaxAgeInHours int

	// UseMessageDates sets the Created date using the message date, not the delivered date
	UseMessageDates bool

	// UITLSCert file
	UITLSCert string

	// UITLSKey file
	UITLSKey string

	// UIAuthFile for UI & API authentication
	UIAuthFile string

	// Webroot to define the base path for the UI and API
	Webroot = "/"

	// SMTPTLSCert file
	SMTPTLSCert string

	// SMTPTLSKey file
	SMTPTLSKey string

	// SMTPRequireSTARTTLS to enforce the use of STARTTLS
	// The only allowed commands are NOOP, EHLO, STARTTLS and QUIT (as specified in RFC 3207) until
	// the connection is upgraded to TLS i.e. until STARTTLS is issued.
	SMTPRequireSTARTTLS bool

	// SMTPRequireTLS to allow only SSL/TLS connections for all connections
	//
	SMTPRequireTLS bool

	// SMTPAuthFile for SMTP authentication
	SMTPAuthFile string

	// SMTPAuthAllowInsecure allows PLAIN & LOGIN unencrypted authentication
	SMTPAuthAllowInsecure bool

	// SMTPAuthAcceptAny accepts any username/password including none
	SMTPAuthAcceptAny bool

	// SMTPMaxRecipients is the maximum number of recipients a message may have.
	// The SMTP RFC states that an server must handle a minimum of 100 recipients
	// however some servers accept more.
	SMTPMaxRecipients = 100

	// IgnoreDuplicateIDs will skip messages with the same ID
	IgnoreDuplicateIDs bool

	// BlockRemoteCSSAndFonts used to disable remote CSS & fonts
	BlockRemoteCSSAndFonts = false

	// CLITagsArg is used to map the CLI args
	CLITagsArg string

	// ValidTagRegexp represents a valid tag
	ValidTagRegexp = regexp.MustCompile(`^([a-zA-Z0-9\-\ \_\.]){1,}$`)

	// TagsConfig is a yaml file to pre-load tags
	TagsConfig string

	// TagFilters are used to apply tags to new mail
	TagFilters []autoTag

	// TagsDisable accepts a comma-separated list of tag types to disable
	// including x-tags & plus-addresses
	TagsDisable string

	// SMTPRelayConfigFile to parse a yaml file and store config of the relay SMTP server
	SMTPRelayConfigFile string

	// SMTPRelayConfig to parse a yaml file and store config of the the relay SMTP server
	SMTPRelayConfig SMTPRelayConfigStruct

	// ReleaseEnabled is whether message releases are enabled, requires a valid SMTPRelayConfigFile
	ReleaseEnabled = false

	// SMTPRelayAll is whether to relay all incoming messages via pre-configured SMTP server.
	// Use with extreme caution!
	SMTPRelayAll = false

	// SMTPRelayMatching if set, will auto-release to recipients matching this regular expression
	SMTPRelayMatching string

	// SMTPRelayMatchingRegexp is the compiled version of SMTPRelayMatching
	SMTPRelayMatchingRegexp *regexp.Regexp

	// SMTPForwardConfigFile to parse a yaml file and store config of the forwarding SMTP server
	SMTPForwardConfigFile string

	// SMTPForwardConfig to parse a yaml file and store config of the forwarding SMTP server
	SMTPForwardConfig SMTPForwardConfigStruct

	// SMTPStrictRFCHeaders will return an error if the email headers contain <CR><CR><LF> (\r\r\n)
	// @see https://github.com/axllent/mailpit/issues/87 & https://github.com/axllent/mailpit/issues/153
	SMTPStrictRFCHeaders bool

	// SMTPAllowedRecipients if set, will only accept recipients matching this regular expression
	SMTPAllowedRecipients string

	// SMTPAllowedRecipientsRegexp is the compiled version of SMTPAllowedRecipients
	SMTPAllowedRecipientsRegexp *regexp.Regexp

	// POP3Listen address - if set then Mailpit will start the POP3 server and listen on this address
	POP3Listen = "[::]:1110"

	// POP3AuthFile for POP3 authentication
	POP3AuthFile string

	// POP3TLSCert TLS certificate
	POP3TLSCert string

	// POP3TLSKey TLS certificate key
	POP3TLSKey string

	// EnableSpamAssassin must be either <host>:<port> or "postmark"
	EnableSpamAssassin string

	// WebhookURL for calling
	WebhookURL string

	// ContentSecurityPolicy for HTTP server - set via VerifyConfig()
	ContentSecurityPolicy string

	// AllowUntrustedTLS allows untrusted HTTPS connections link checking & screenshot generation
	AllowUntrustedTLS bool

	// Version is the default application version, updated on release
	Version = "dev"

	// Repo on Github for updater
	Repo = "axllent/mailpit"

	// RepoBinaryName on Github for updater
	RepoBinaryName = "mailpit"

	// ChaosTriggers are parsed and set in the chaos module
	ChaosTriggers string

	// DisableHTMLCheck DEPRECATED 2024/04/13 - kept here to display console warning only
	DisableHTMLCheck = false

	// DemoMode disables SMTP relay, link checking & HTTP send functionality
	DemoMode = false
)

// AutoTag struct for auto-tagging
type autoTag struct {
	Match string
	Tags  []string
}

// SMTPRelayConfigStruct struct for parsing yaml & storing variables
type SMTPRelayConfigStruct struct {
	Host                    string         `yaml:"host"`
	Port                    int            `yaml:"port"`
	STARTTLS                bool           `yaml:"starttls"`
	AllowInsecure           bool           `yaml:"allow-insecure"`
	Auth                    string         `yaml:"auth"`               // none, plain, login, cram-md5
	Username                string         `yaml:"username"`           // plain & cram-md5
	Password                string         `yaml:"password"`           // plain
	Secret                  string         `yaml:"secret"`             // cram-md5
	ReturnPath              string         `yaml:"return-path"`        // allow overriding the bounce address
	OverrideFrom            string         `yaml:"override-from"`      // allow overriding of the from address
	AllowedRecipients       string         `yaml:"allowed-recipients"` // regex, if set needs to match for mails to be relayed
	AllowedRecipientsRegexp *regexp.Regexp // compiled regexp using AllowedRecipients
	BlockedRecipients       string         `yaml:"blocked-recipients"` // regex, if set prevents relating to these addresses
	BlockedRecipientsRegexp *regexp.Regexp // compiled regexp using BlockedRecipients

	// DEPRECATED 2024/03/12
	RecipientAllowlist string `yaml:"recipient-allowlist"`
}

// SMTPForwardConfigStruct struct for parsing yaml & storing variables
type SMTPForwardConfigStruct struct {
	To            string `yaml:"to"`             // comma-separated list of email addresses
	Host          string `yaml:"host"`           // SMTP host
	Port          int    `yaml:"port"`           // SMTP port
	STARTTLS      bool   `yaml:"starttls"`       // whether to use STARTTLS
	AllowInsecure bool   `yaml:"allow-insecure"` // allow insecure authentication
	Auth          string `yaml:"auth"`           // none, plain, login, cram-md5
	Username      string `yaml:"username"`       // plain & cram-md5
	Password      string `yaml:"password"`       // plain
	Secret        string `yaml:"secret"`         // cram-md5
	ReturnPath    string `yaml:"return-path"`    // allow overriding the bounce address
	OverrideFrom  string `yaml:"override-from"`  // allow overriding of the from address
}

// VerifyConfig wil do some basic checking
func VerifyConfig() error {
	cssFontRestriction := "*"
	if BlockRemoteCSSAndFonts {
		cssFontRestriction = "'self'"
	}

	// The default Content Security Policy is updates on every application page load to replace script-src 'self'
	// with a random nonce ID to prevent XSS. This applies to the Mailpit app & API.
	// See server.middleWareFunc()
	ContentSecurityPolicy = fmt.Sprintf("default-src 'self'; script-src 'self'; style-src %s 'unsafe-inline'; frame-src 'self'; img-src * data: blob:; font-src %s data:; media-src 'self'; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self';",
		cssFontRestriction, cssFontRestriction,
	)

	if Database != "" && isDir(Database) {
		Database = filepath.Join(Database, "mailpit.db")
	}

	Label = tools.Normalize(Label)

	if err := parseMaxAge(); err != nil {
		return err
	}

	TenantID = DBTenantID(TenantID)
	if TenantID != "" {
		logger.Log().Infof("[db] using tenant \"%s\"", TenantID)
	}

	re := regexp.MustCompile(`.*:\d+$`)
	if _, _, isSocket := tools.UnixSocket(SMTPListen); !isSocket && !re.MatchString(SMTPListen) {
		return errors.New("[smtp] bind should be in the format of <ip>:<port>")
	}
	if _, _, isSocket := tools.UnixSocket(HTTPListen); !isSocket && !re.MatchString(HTTPListen) {
		return errors.New("[ui] HTTP bind should be in the format of <ip>:<port>")
	}

	if UIAuthFile != "" {
		UIAuthFile = filepath.Clean(UIAuthFile)

		if !isFile(UIAuthFile) {
			return fmt.Errorf("[ui] HTTP password file not found or readable: %s", UIAuthFile)
		}

		b, err := os.ReadFile(UIAuthFile)
		if err != nil {
			return err
		}

		if err := auth.SetUIAuth(string(b)); err != nil {
			return err
		}
	}

	if UITLSCert != "" && UITLSKey == "" || UITLSCert == "" && UITLSKey != "" {
		return errors.New("[ui] you must provide both a UI TLS certificate and a key")
	}

	if UITLSCert != "" {
		UITLSCert = filepath.Clean(UITLSCert)
		UITLSKey = filepath.Clean(UITLSKey)

		if !isFile(UITLSCert) {
			return fmt.Errorf("[ui] TLS certificate not found or readable: %s", UITLSCert)
		}

		if !isFile(UITLSKey) {
			return fmt.Errorf("[ui] TLS key not found or readable: %s", UITLSKey)
		}
	}

	if SMTPTLSCert != "" && SMTPTLSKey == "" || SMTPTLSCert == "" && SMTPTLSKey != "" {
		return errors.New("[smtp] You must provide both an SMTP TLS certificate and a key")
	}

	if SMTPTLSCert != "" {
		SMTPTLSCert = filepath.Clean(SMTPTLSCert)
		SMTPTLSKey = filepath.Clean(SMTPTLSKey)

		if !isFile(SMTPTLSCert) {
			return fmt.Errorf("[smtp] TLS certificate not found or readable: %s", SMTPTLSCert)
		}

		if !isFile(SMTPTLSKey) {
			return fmt.Errorf("[smtp] TLS key not found or readable: %s", SMTPTLSKey)
		}
	} else if SMTPRequireTLS {
		return errors.New("[smtp] TLS cannot be required without an SMTP TLS certificate and key")
	} else if SMTPRequireSTARTTLS {
		return errors.New("[smtp] STARTTLS cannot be required without an SMTP TLS certificate and key")
	}
	if SMTPRequireSTARTTLS && SMTPAuthAllowInsecure || SMTPRequireTLS && SMTPAuthAllowInsecure {
		return errors.New("[smtp] TLS cannot be required with --smtp-auth-allow-insecure")
	}
	if SMTPRequireSTARTTLS && SMTPRequireTLS {
		return errors.New("[smtp] TLS & STARTTLS cannot be required together")
	}

	if SMTPAuthFile != "" {
		SMTPAuthFile = filepath.Clean(SMTPAuthFile)

		if !isFile(SMTPAuthFile) {
			return fmt.Errorf("[smtp] password file not found or readable: %s", SMTPAuthFile)
		}

		b, err := os.ReadFile(SMTPAuthFile)
		if err != nil {
			return err
		}

		if err := auth.SetSMTPAuth(string(b)); err != nil {
			return err
		}

		if !SMTPAuthAllowInsecure {
			// https://www.rfc-editor.org/rfc/rfc4954
			// A server implementation MUST implement a configuration in which
			// it does NOT permit any plaintext password mechanisms, unless either
			// the STARTTLS [SMTP-TLS] command has been negotiated or some other
			// mechanism that protects the session from password snooping has been
			// provided.  Server sites SHOULD NOT use any configuration which
			// permits a plaintext password mechanism without such a protection
			// mechanism against password snooping.
			SMTPRequireSTARTTLS = true
		}
	}

	if auth.SMTPCredentials != nil && SMTPAuthAcceptAny {
		return errors.New("[smtp] authentication cannot use both credentials and --smtp-auth-accept-any")
	}

	if SMTPTLSCert == "" && (auth.SMTPCredentials != nil || SMTPAuthAcceptAny) && !SMTPAuthAllowInsecure {
		return errors.New("[smtp] authentication requires STARTTLS or TLS encryption, run with `--smtp-auth-allow-insecure` to allow insecure authentication")
	}

	if err := parseChaosTriggers(); err != nil {
		return fmt.Errorf("[chaos] %s", err.Error())
	}

	if chaos.Enabled {
		logger.Log().Info("[chaos] is enabled")
	}

	// POP3 server
	if POP3TLSCert != "" {
		POP3TLSCert = filepath.Clean(POP3TLSCert)
		POP3TLSKey = filepath.Clean(POP3TLSKey)

		if !isFile(POP3TLSCert) {
			return fmt.Errorf("[pop3] TLS certificate not found or readable: %s", POP3TLSCert)
		}

		if !isFile(POP3TLSKey) {
			return fmt.Errorf("[pop3] TLS key not found or readable: %s", POP3TLSKey)
		}
	}
	if POP3TLSCert != "" && POP3TLSKey == "" || POP3TLSCert == "" && POP3TLSKey != "" {
		return errors.New("[pop3] You must provide both a POP3 TLS certificate and a key")
	}
	if POP3Listen != "" {
		_, err := net.ResolveTCPAddr("tcp", POP3Listen)
		if err != nil {
			return fmt.Errorf("[pop3] %s", err.Error())
		}
	}
	if POP3AuthFile != "" {
		POP3AuthFile = filepath.Clean(POP3AuthFile)

		if !isFile(POP3AuthFile) {
			return fmt.Errorf("[pop3] password file not found or readable: %s", POP3AuthFile)
		}

		b, err := os.ReadFile(POP3AuthFile)
		if err != nil {
			return err
		}

		if err := auth.SetPOP3Auth(string(b)); err != nil {
			return err
		}
	}

	// Web root
	validWebrootRe := regexp.MustCompile(`[^0-9a-zA-Z\/\-\_\.@]`)
	if validWebrootRe.MatchString(Webroot) {
		return fmt.Errorf("invalid characters in Webroot (%s). Valid chars include: [a-z A-Z 0-9 _ . - / @]", Webroot)
	}

	s := strings.TrimRight(path.Join("/", Webroot, "/"), "/") + "/"
	Webroot = s

	if WebhookURL != "" && !isValidURL(WebhookURL) {
		return fmt.Errorf("webhook URL does not appear to be a valid URL (%s)", WebhookURL)
	}

	// DEPRECATED 2024/04/13
	if DisableHTMLCheck {
		logger.Log().Warn("--disable-html-check has been deprecated and is no longer used")
	}

	if EnableSpamAssassin != "" {
		spamassassin.SetService(EnableSpamAssassin)
		logger.Log().Infof("[spamassassin] enabled via %s", EnableSpamAssassin)

		if err := spamassassin.Ping(); err != nil {
			logger.Log().Warnf("[spamassassin] ping: %s", err.Error())
		}
	}

	// load tag filters & options
	TagFilters = []autoTag{}
	if err := loadTagsFromArgs(CLITagsArg); err != nil {
		return err
	}
	if err := loadTagsFromConfig(TagsConfig); err != nil {
		return err
	}
	if err := parseTagsDisable(TagsDisable); err != nil {
		return err
	}

	if SMTPAllowedRecipients != "" {
		restrictRegexp, err := regexp.Compile(SMTPAllowedRecipients)
		if err != nil {
			return fmt.Errorf("[smtp] failed to compile smtp-allowed-recipients regexp: %s", err.Error())
		}

		SMTPAllowedRecipientsRegexp = restrictRegexp
		logger.Log().Infof("[smtp] only allowing recipients matching regexp: %s", SMTPAllowedRecipients)
	}

	if err := parseRelayConfig(SMTPRelayConfigFile); err != nil {
		return err
	}

	// separate relay config validation to account for environment variables
	if err := validateRelayConfig(); err != nil {
		return err
	}

	if !ReleaseEnabled && SMTPRelayAll || !ReleaseEnabled && SMTPRelayMatching != "" {
		return errors.New("[relay] a relay configuration must be set to auto-relay any messages")
	}

	if SMTPRelayMatching != "" {
		if SMTPRelayAll {
			logger.Log().Warnf("[relay] ignoring smtp-relay-matching when smtp-relay-all is enabled")
		} else {
			re, err := regexp.Compile(SMTPRelayMatching)
			if err != nil {
				return fmt.Errorf("[relay] failed to compile smtp-relay-matching regexp: %s", err.Error())
			}

			SMTPRelayMatchingRegexp = re
			logger.Log().Infof("[relay] auto-relaying new messages to recipients matching \"%s\" via %s:%d",
				SMTPRelayMatching, SMTPRelayConfig.Host, SMTPRelayConfig.Port)
		}
	}

	if SMTPRelayAll {
		// this deserves a warning
		logger.Log().Warnf("[relay] auto-relaying all new messages via %s:%d", SMTPRelayConfig.Host, SMTPRelayConfig.Port)
	}

	if err := parseForwardConfig(SMTPForwardConfigFile); err != nil {
		return err
	}

	// separate forwarding config validation to account for environment variables
	if err := validateForwardConfig(); err != nil {
		return err
	}

	if DemoMode {
		MaxMessages = 1000
		// this deserves a warning
		logger.Log().Info("demo mode enabled")
	}

	return nil
}
