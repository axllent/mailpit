// Package config handles the application configuration
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/spamassassin"
	"github.com/axllent/mailpit/internal/tools"
	"gopkg.in/yaml.v3"
)

var (
	// SMTPListen to listen on <interface>:<port>
	SMTPListen = "[::]:1025"

	// HTTPListen to listen on <interface>:<port>
	HTTPListen = "[::]:8025"

	// DataFile for mail (optional)
	DataFile string

	// MaxMessages is the maximum number of messages a mailbox can have (auto-pruned every minute)
	MaxMessages = 500

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

	// SMTPTLSRequired to enforce TLS
	// The only allowed commands are NOOP, EHLO, STARTTLS and QUIT (as specified in RFC 3207) until
	// the connection is upgraded to TLS i.e. until STARTTLS is issued.
	SMTPTLSRequired bool

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

	// DisableHTMLCheck used to disable the HTML check in bother the API and web UI
	DisableHTMLCheck = false

	// BlockRemoteCSSAndFonts used to disable remote CSS & fonts
	BlockRemoteCSSAndFonts = false

	// SMTPCLITags is used to map the CLI args
	SMTPCLITags string

	// ValidTagRegexp represents a valid tag
	ValidTagRegexp = regexp.MustCompile(`^([a-zA-Z0-9\-\ \_]){3,}$`)

	// SMTPTags are expressions to apply tags to new mail
	SMTPTags []AutoTag

	// SMTPRelayConfigFile to parse a yaml file and store config of relay SMTP server
	SMTPRelayConfigFile string

	// SMTPRelayConfig to parse a yaml file and store config of relay SMTP server
	SMTPRelayConfig smtpRelayConfigStruct

	// SMTPStrictRFCHeaders will return an error if the email headers contain <CR><CR><LF> (\r\r\n)
	// @see https://github.com/axllent/mailpit/issues/87 & https://github.com/axllent/mailpit/issues/153
	SMTPStrictRFCHeaders bool

	// SMTPAllowedRecipients if set, will only accept recipients matching this regular expression
	SMTPAllowedRecipients string

	// SMTPAllowedRecipientsRegexp is the compiled version of SMTPAllowedRecipients
	SMTPAllowedRecipientsRegexp *regexp.Regexp

	// ReleaseEnabled is whether message releases are enabled, requires a valid SMTPRelayConfigFile
	ReleaseEnabled = false

	// SMTPRelayAllIncoming is whether to relay all incoming messages via pre-configured SMTP server.
	// Use with extreme caution!
	SMTPRelayAllIncoming = false

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
)

// AutoTag struct for auto-tagging
type AutoTag struct {
	Tag   string
	Match string
}

// SMTPRelayConfigStruct struct for parsing yaml & storing variables
type smtpRelayConfigStruct struct {
	Host                     string `yaml:"host"`
	Port                     int    `yaml:"port"`
	STARTTLS                 bool   `yaml:"starttls"`
	AllowInsecure            bool   `yaml:"allow-insecure"`
	Auth                     string `yaml:"auth"`                // none, plain, login, cram-md5
	Username                 string `yaml:"username"`            // plain & cram-md5
	Password                 string `yaml:"password"`            // plain
	Secret                   string `yaml:"secret"`              // cram-md5
	ReturnPath               string `yaml:"return-path"`         // allow overriding the bounce address
	RecipientAllowlist       string `yaml:"recipient-allowlist"` // regex, if set needs to match for mails to be relayed
	RecipientAllowlistRegexp *regexp.Regexp
}

// VerifyConfig wil do some basic checking
func VerifyConfig() error {
	cssFontRestriction := "*"
	if BlockRemoteCSSAndFonts {
		cssFontRestriction = "'self'"
	}

	ContentSecurityPolicy = fmt.Sprintf("default-src 'self'; script-src 'self'; style-src %s 'unsafe-inline'; frame-src 'self'; img-src * data: blob:; font-src %s data:; media-src 'self'; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self';",
		cssFontRestriction, cssFontRestriction,
	)

	if DataFile != "" && isDir(DataFile) {
		DataFile = filepath.Join(DataFile, "mailpit.db")
	}

	re := regexp.MustCompile(`.*:\d+$`)
	if !re.MatchString(SMTPListen) {
		return errors.New("[smtp] bind should be in the format of <ip>:<port>")
	}
	if !re.MatchString(HTTPListen) {
		return errors.New("[ui] HTTP bind should be in the format of <ip>:<port>")
	}

	if UIAuthFile != "" {
		UIAuthFile = filepath.Clean(UIAuthFile)

		if !isFile(UIAuthFile) {
			return fmt.Errorf("[ui] HTTP password file not found: %s", UIAuthFile)
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
			return fmt.Errorf("[ui] TLS certificate not found: %s", UITLSCert)
		}

		if !isFile(UITLSKey) {
			return fmt.Errorf("[ui] TLS key not found: %s", UITLSKey)
		}
	}

	if SMTPTLSCert != "" && SMTPTLSKey == "" || SMTPTLSCert == "" && SMTPTLSKey != "" {
		return errors.New("[smtp] You must provide both an SMTP TLS certificate and a key")
	}

	if SMTPTLSCert != "" {
		SMTPTLSCert = filepath.Clean(SMTPTLSCert)
		SMTPTLSKey = filepath.Clean(SMTPTLSKey)

		if !isFile(SMTPTLSCert) {
			return fmt.Errorf("[smtp] TLS certificate not found: %s", SMTPTLSCert)
		}

		if !isFile(SMTPTLSKey) {
			return fmt.Errorf("[smtp] TLS key not found: %s", SMTPTLSKey)
		}
	} else if SMTPTLSRequired {
		return errors.New("[smtp] TLS cannot be required without an SMTP TLS certificate and key")
	}

	if SMTPTLSRequired && SMTPAuthAllowInsecure {
		return errors.New("[smtp] TLS cannot be required while also allowing insecure authentication")
	}

	if SMTPAuthFile != "" {
		SMTPAuthFile = filepath.Clean(SMTPAuthFile)

		if !isFile(SMTPAuthFile) {
			return fmt.Errorf("[smtp] password file not found: %s", SMTPAuthFile)
		}

		b, err := os.ReadFile(SMTPAuthFile)
		if err != nil {
			return err
		}

		if err := auth.SetSMTPAuth(string(b)); err != nil {
			return err
		}
	}

	if auth.SMTPCredentials != nil && SMTPAuthAcceptAny {
		return errors.New("[smtp] authentication cannot use both credentials and --smtp-auth-accept-any")
	}

	if SMTPTLSCert == "" && (auth.SMTPCredentials != nil || SMTPAuthAcceptAny) && !SMTPAuthAllowInsecure {
		return errors.New("[smtp] authentication requires TLS encryption, run with `--smtp-auth-allow-insecure` to allow insecure authentication")
	}

	// POP3 server
	if POP3TLSCert != "" {
		POP3TLSCert = filepath.Clean(POP3TLSCert)
		POP3TLSKey = filepath.Clean(POP3TLSKey)

		if !isFile(POP3TLSCert) {
			return fmt.Errorf("[pop3] TLS certificate not found: %s", POP3TLSCert)
		}

		if !isFile(POP3TLSKey) {
			return fmt.Errorf("[pop3] TLS key not found: %s", POP3TLSKey)
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
			return fmt.Errorf("[pop3] password file not found: %s", POP3AuthFile)
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

	if EnableSpamAssassin != "" {
		spamassassin.SetService(EnableSpamAssassin)
		logger.Log().Infof("[spamassassin] enabled via %s", EnableSpamAssassin)

		if err := spamassassin.Ping(); err != nil {
			logger.Log().Warnf("[spamassassin] ping: %s", err.Error())
		}
	}

	SMTPTags = []AutoTag{}

	if SMTPCLITags != "" {
		args := tools.ArgsParser(SMTPCLITags)

		for _, a := range args {
			t := strings.Split(a, "=")
			if len(t) > 1 {
				tag := tools.CleanTag(t[0])
				if !ValidTagRegexp.MatchString(tag) || len(tag) == 0 {
					return fmt.Errorf("[tag] invalid tag (%s) - can only contain spaces, letters, numbers, - & _", tag)
				}
				match := strings.TrimSpace(strings.ToLower(strings.Join(t[1:], "=")))
				if len(match) == 0 {
					return fmt.Errorf("[tag] invalid tag match (%s) - no search detected", tag)
				}
				SMTPTags = append(SMTPTags, AutoTag{Tag: tag, Match: match})
			} else {
				return fmt.Errorf("[tag] error parsing tags (%s)", a)
			}
		}
	}

	if SMTPAllowedRecipients != "" {
		restrictRegexp, err := regexp.Compile(SMTPAllowedRecipients)
		if err != nil {
			return fmt.Errorf("[smtp] failed to compile smtp-allowed-recipients regexp: %s", err.Error())
		}

		SMTPAllowedRecipientsRegexp = restrictRegexp
		logger.Log().Infof("[smtp] only allowing recipients matching the following regexp: %s", SMTPAllowedRecipients)
	}

	if err := parseRelayConfig(SMTPRelayConfigFile); err != nil {
		return err
	}

	if !ReleaseEnabled && SMTPRelayAllIncoming {
		return errors.New("[smtp] relay config must be set to relay all messages")
	}

	if SMTPRelayAllIncoming {
		// this deserves a warning
		logger.Log().Warnf("[smtp] enabling automatic relay of all new messages via %s:%d", SMTPRelayConfig.Host, SMTPRelayConfig.Port)
	}

	return nil
}

// Parse & validate the SMTPRelayConfigFile (if set)
func parseRelayConfig(c string) error {
	if c == "" {
		return nil
	}

	c = filepath.Clean(c)

	if !isFile(c) {
		return fmt.Errorf("[smtp] relay configuration not found: %s", c)
	}

	data, err := os.ReadFile(c)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &SMTPRelayConfig); err != nil {
		return err
	}

	if SMTPRelayConfig.Host == "" {
		return errors.New("[smtp] relay host not set")
	}

	if SMTPRelayConfig.Port == 0 {
		SMTPRelayConfig.Port = 25 // default
	}

	SMTPRelayConfig.Auth = strings.ToLower(SMTPRelayConfig.Auth)

	if SMTPRelayConfig.Auth == "" || SMTPRelayConfig.Auth == "none" || SMTPRelayConfig.Auth == "false" {
		SMTPRelayConfig.Auth = "none"
	} else if SMTPRelayConfig.Auth == "plain" {
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("[smtp] relay host username or password not set for PLAIN authentication (%s)", c)
		}
	} else if SMTPRelayConfig.Auth == "login" {
		SMTPRelayConfig.Auth = "login"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("[smtp] relay host username or password not set for LOGIN authentication (%s)", c)
		}
	} else if strings.HasPrefix(SMTPRelayConfig.Auth, "cram") {
		SMTPRelayConfig.Auth = "cram-md5"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Secret == "" {
			return fmt.Errorf("[smtp] relay host username or secret not set for CRAM-MD5 authentication (%s)", c)
		}
	} else {
		return fmt.Errorf("[smtp] relay authentication method not supported: %s", SMTPRelayConfig.Auth)
	}

	ReleaseEnabled = true

	logger.Log().Infof("[smtp] enabling message relaying via %s:%d", SMTPRelayConfig.Host, SMTPRelayConfig.Port)

	allowlistRegexp, err := regexp.Compile(SMTPRelayConfig.RecipientAllowlist)

	if SMTPRelayConfig.RecipientAllowlist != "" {
		if err != nil {
			return fmt.Errorf("[smtp] failed to compile relay recipient allowlist regexp: %s", err.Error())
		}

		SMTPRelayConfig.RecipientAllowlistRegexp = allowlistRegexp
		logger.Log().Infof("[smtp] relay recipient allowlist is active with the following regexp: %s", SMTPRelayConfig.RecipientAllowlist)

	}

	return nil
}

// IsFile returns if a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// IsDir returns whether a path is a directory
func isDir(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.IsDir() {
		return false
	}

	return true
}

func isValidURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}

	return strings.HasPrefix(u.Scheme, "http")
}
