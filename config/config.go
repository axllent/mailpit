// Package config handles the application configuration
package config

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/tg123/go-htpasswd"
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

	// VerboseLogging for console output
	VerboseLogging = false

	// QuietLogging for console output (errors only)
	QuietLogging = false

	// NoLogging for tests
	NoLogging = false

	// UITLSCert file
	UITLSCert string

	// UITLSKey file
	UITLSKey string

	// UIAuthFile for basic authentication
	UIAuthFile string

	// UIAuth used for euthentication
	UIAuth *htpasswd.File

	// Webroot to define the base path for the UI and API
	Webroot = "/"

	// SMTPTLSCert file
	SMTPTLSCert string

	// SMTPTLSKey file
	SMTPTLSKey string

	// SMTPAuthFile for SMTP authentication
	SMTPAuthFile string

	// SMTPAuth used for euthentication
	SMTPAuth *htpasswd.File

	// SMTPAuthAllowInsecure allows PLAIN & LOGIN unencrypted authentication
	SMTPAuthAllowInsecure bool

	// SMTPAuthAcceptAny accepts any username/password including none
	SMTPAuthAcceptAny bool

	// SMTPCLITags is used to map the CLI args
	SMTPCLITags string

	// TagRegexp is the allowed tag characters
	TagRegexp = regexp.MustCompile(`^([a-zA-Z0-9\-\ \_]){3,}$`)

	// SMTPTags are expressions to apply tags to new mail
	SMTPTags []Tag

	// ContentSecurityPolicy for HTTP server
	ContentSecurityPolicy = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; frame-src 'self'; img-src * data: blob:; font-src 'self' data:; media-src 'self'; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self';"

	// Version is the default application version, updated on release
	Version = "dev"

	// Repo on Github for updater
	Repo = "axllent/mailpit"

	// RepoBinaryName on Github for updater
	RepoBinaryName = "mailpit"
)

// Tag struct
type Tag struct {
	Tag   string
	Match string
}

// VerifyConfig wil do some basic checking
func VerifyConfig() error {
	if DataFile != "" && isDir(DataFile) {
		DataFile = filepath.Join(DataFile, "mailpit.db")
	}

	re := regexp.MustCompile(`^((\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})|(\[([\da-fA-F:])+\])):\d+$`)
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

		a, err := htpasswd.New(UIAuthFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		UIAuth = a
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

		if SMTPAuthAcceptAny {
			return errors.New("SMTP authentication can either use --smtp-auth-file or --smtp-auth-accept-any")
		}

		a, err := htpasswd.New(SMTPAuthFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		SMTPAuth = a
	}

	if SMTPTLSCert == "" && (SMTPAuthFile != "" || SMTPAuthAcceptAny) && !SMTPAuthAllowInsecure {
		return errors.New("SMTP authentication requires TLS encryption, run with `--smtp-auth-allow-insecure` to allow insecure authentication")
	}

	validWebrootRe := regexp.MustCompile(`[^0-9a-zA-Z\/\-\_\.]`)
	if validWebrootRe.MatchString(Webroot) {
		return fmt.Errorf("Invalid characters in Webroot (%s). Valid chars include: [a-z A-Z 0-9 _ . - /]", Webroot)
	}

	s := strings.TrimRight(path.Join("/", Webroot, "/"), "/") + "/"
	Webroot = s

	SMTPTags = []Tag{}

	p := shellwords.NewParser()

	if SMTPCLITags != "" {
		args, err := p.Parse(SMTPCLITags)
		if err != nil {
			return fmt.Errorf("Error parsing tags (%s)", err)
		}

		for _, a := range args {
			t := strings.Split(a, "=")
			if len(t) > 1 {
				tag := strings.TrimSpace(t[0])
				if !TagRegexp.MatchString(tag) || len(tag) == 0 {
					return fmt.Errorf("Invalid tag (%s) - can only contain spaces, letters, numbers, - & _", tag)
				}
				match := strings.TrimSpace(strings.ToLower(strings.Join(t[1:], "=")))
				if len(match) == 0 {
					return fmt.Errorf("Invalid tag match (%s) - no search detected", tag)
				}
				SMTPTags = append(SMTPTags, Tag{Tag: tag, Match: match})
			} else {
				return fmt.Errorf("Error parsing tags (%s)", a)
			}
		}

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
