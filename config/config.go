// Package config handles the application configuration
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/auth"
	"github.com/axllent/mailpit/internal/logger"
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

	// ReleaseEnabled is whether message releases are enabled, requires a valid SMTPRelayConfigFile
	ReleaseEnabled = false

	// SMTPRelayAllIncoming is whether to relay all incoming messages via pre-configured SMTP server.
	// Use with extreme caution!
	SMTPRelayAllIncoming = false

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
		return errors.New("SMTP bind should be in the format of <ip>:<port>")
	}
	if !re.MatchString(HTTPListen) {
		return errors.New("HTTP bind should be in the format of <ip>:<port>")
	}

	if UIAuthFile != "" {
		if !isFile(UIAuthFile) {
			return fmt.Errorf("HTTP password file not found: %s", UIAuthFile)
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
		return errors.New("You must provide both a UI TLS certificate and a key")
	}

	if UITLSCert != "" {
		if !isFile(UITLSCert) {
			return fmt.Errorf("TLS certificate not found: %s", UITLSCert)
		}

		if !isFile(UITLSKey) {
			return fmt.Errorf("TLS key not found: %s", UITLSKey)
		}
	}

	if SMTPTLSCert != "" && SMTPTLSKey == "" || SMTPTLSCert == "" && SMTPTLSKey != "" {
		return errors.New("You must provide both an SMTP TLS certificate and a key")
	}

	if SMTPTLSCert != "" {
		if !isFile(SMTPTLSCert) {
			return fmt.Errorf("SMTP TLS certificate not found: %s", SMTPTLSCert)
		}

		if !isFile(SMTPTLSKey) {
			return fmt.Errorf("SMTP TLS key not found: %s", SMTPTLSKey)
		}
	}

	if SMTPAuthFile != "" {
		if !isFile(SMTPAuthFile) {
			return fmt.Errorf("SMTP password file not found: %s", SMTPAuthFile)
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
		return errors.New("SMTP authentication cannot use both credentials and --smtp-auth-accept-any")
	}

	if SMTPTLSCert == "" && (auth.SMTPCredentials != nil || SMTPAuthAcceptAny) && !SMTPAuthAllowInsecure {
		return errors.New("SMTP authentication requires TLS encryption, run with `--smtp-auth-allow-insecure` to allow insecure authentication")
	}

	validWebrootRe := regexp.MustCompile(`[^0-9a-zA-Z\/\-\_\.@]`)
	if validWebrootRe.MatchString(Webroot) {
		return fmt.Errorf("Invalid characters in Webroot (%s). Valid chars include: [a-z A-Z 0-9 _ . - / @]", Webroot)
	}

	s := strings.TrimRight(path.Join("/", Webroot, "/"), "/") + "/"
	Webroot = s

	if WebhookURL != "" && !isValidURL(WebhookURL) {
		return fmt.Errorf("Webhook URL does not appear to be a valid URL (%s)", WebhookURL)
	}

	SMTPTags = []AutoTag{}

	if SMTPCLITags != "" {
		args := tools.ArgsParser(SMTPCLITags)

		for _, a := range args {
			t := strings.Split(a, "=")
			if len(t) > 1 {
				tag := tools.CleanTag(t[0])
				if !ValidTagRegexp.MatchString(tag) || len(tag) == 0 {
					return fmt.Errorf("Invalid tag (%s) - can only contain spaces, letters, numbers, - & _", tag)
				}
				match := strings.TrimSpace(strings.ToLower(strings.Join(t[1:], "=")))
				if len(match) == 0 {
					return fmt.Errorf("Invalid tag match (%s) - no search detected", tag)
				}
				SMTPTags = append(SMTPTags, AutoTag{Tag: tag, Match: match})
			} else {
				return fmt.Errorf("Error parsing tags (%s)", a)
			}
		}
	}

	if err := parseRelayConfig(SMTPRelayConfigFile); err != nil {
		return err
	}

	if !ReleaseEnabled && SMTPRelayAllIncoming {
		return errors.New("SMTP relay config must be set to relay all messages")
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

	if !isFile(c) {
		return fmt.Errorf("SMTP relay configuration not found: %s", SMTPRelayConfigFile)
	}

	data, err := os.ReadFile(c)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &SMTPRelayConfig); err != nil {
		return err
	}

	if SMTPRelayConfig.Host == "" {
		return errors.New("SMTP relay host not set")
	}

	if SMTPRelayConfig.Port == 0 {
		SMTPRelayConfig.Port = 25 // default
	}

	SMTPRelayConfig.Auth = strings.ToLower(SMTPRelayConfig.Auth)

	if SMTPRelayConfig.Auth == "" || SMTPRelayConfig.Auth == "none" || SMTPRelayConfig.Auth == "false" {
		SMTPRelayConfig.Auth = "none"
	} else if SMTPRelayConfig.Auth == "plain" {
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("SMTP relay host username or password not set for PLAIN authentication (%s)", c)
		}
	} else if SMTPRelayConfig.Auth == "login" {
		SMTPRelayConfig.Auth = "login"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Password == "" {
			return fmt.Errorf("SMTP relay host username or password not set for LOGIN authentication (%s)", c)
		}
	} else if strings.HasPrefix(SMTPRelayConfig.Auth, "cram") {
		SMTPRelayConfig.Auth = "cram-md5"
		if SMTPRelayConfig.Username == "" || SMTPRelayConfig.Secret == "" {
			return fmt.Errorf("SMTP relay host username or secret not set for CRAM-MD5 authentication (%s)", c)
		}
	} else {
		return fmt.Errorf("SMTP relay authentication method not supported: %s", SMTPRelayConfig.Auth)
	}

	ReleaseEnabled = true

	logger.Log().Infof("[smtp] enabling message relaying via %s:%d", SMTPRelayConfig.Host, SMTPRelayConfig.Port)

	allowlistRegexp, err := regexp.Compile(SMTPRelayConfig.RecipientAllowlist)

	if SMTPRelayConfig.RecipientAllowlist != "" {
		if err != nil {
			return fmt.Errorf("failed to compile recipient allowlist regexp: %e", err)
		}

		SMTPRelayConfig.RecipientAllowlistRegexp = allowlistRegexp
		logger.Log().Infof("[smtp] recipient allowlist is active with the following regexp: %s", SMTPRelayConfig.RecipientAllowlist)

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
