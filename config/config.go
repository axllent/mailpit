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
	SMTPListen = "0.0.0.0:1025"

	// HTTPListen to listen on <interface>:<port>
	HTTPListen = "0.0.0.0:8025"

	// DataFile for mail (optional)
	DataFile string

	// MaxMessages is the maximum number of messages a mailbox can have (auto-pruned every minute)
	MaxMessages = 500

	// VerboseLogging for console output
	VerboseLogging = false

	// QuietLogging for console output (errors only)
	QuietLogging = false

	// NoLogging for tests
	NoLogging = false

	// UISSLCert file
	UISSLCert string

	// UISSLKey file
	UISSLKey string

	// UIAuthFile for basic authentication
	UIAuthFile string

	// UIAuth used for euthentication
	UIAuth *htpasswd.File

	// Webroot to define the base path for the UI and API
	Webroot = "/"

	// SMTPSSLCert file
	SMTPSSLCert string

	// SMTPSSLKey file
	SMTPSSLKey string

	// SMTPAuthFile for SMTP authentication
	SMTPAuthFile string

	// SMTPAuth used for euthentication
	SMTPAuth *htpasswd.File

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

	re := regexp.MustCompile(`^[a-zA-Z0-9\.\-]{3,}:\d{2,}$`)
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

	if UISSLCert != "" && UISSLKey == "" || UISSLCert == "" && UISSLKey != "" {
		return errors.New("you must provide both a UI SSL certificate and a key")
	}

	if UISSLCert != "" {
		if !isFile(UISSLCert) {
			return fmt.Errorf("SSL certificate not found: %s", UISSLCert)
		}

		if !isFile(UISSLKey) {
			return fmt.Errorf("SSL key not found: %s", UISSLKey)
		}
	}

	if SMTPSSLCert != "" && SMTPSSLKey == "" || SMTPSSLCert == "" && SMTPSSLKey != "" {
		return errors.New("you must provide both an SMTP SSL certificate and a key")
	}

	if SMTPSSLCert != "" {
		if !isFile(SMTPSSLCert) {
			return fmt.Errorf("SMTP SSL certificate not found: %s", SMTPSSLCert)
		}

		if !isFile(SMTPSSLKey) {
			return fmt.Errorf("SMTP SSL key not found: %s", SMTPSSLKey)
		}
	}

	if SMTPAuthFile != "" {
		if !isFile(SMTPAuthFile) {
			return fmt.Errorf("SMTP password file not found: %s", SMTPAuthFile)
		}

		if SMTPSSLCert == "" {
			return errors.New("SMTP authentication requires SMTP encryption")
		}

		a, err := htpasswd.New(SMTPAuthFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		SMTPAuth = a
	}

	if strings.Contains(Webroot, " ") {
		return fmt.Errorf("Webroot cannot contain spaces (%s)", Webroot)
	}

	s := path.Join("/", Webroot, "/")
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
