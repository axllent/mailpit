package config

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/tg123/go-htpasswd"
)

var (
	// SMTPListen to listen on <interface>:<port>
	SMTPListen = "0.0.0.0:1025"

	// HTTPListen to listen on <interface>:<port>
	HTTPListen = "0.0.0.0:8025"

	// DataDir for mail (optional)
	DataDir string

	// MaxMessages is the maximum number of messages a mailbox can have (auto-pruned every minute)
	MaxMessages = 500

	// VerboseLogging for console output
	VerboseLogging = false

	// NoLogging for tests
	NoLogging = false

	// SSLCert file
	SSLCert string

	// SSLKey file
	SSLKey string

	// AuthFile for basic authentication
	AuthFile string

	// Auth used for euthentication
	Auth *htpasswd.File
)

// VerifyConfig wil do some basic checking
func VerifyConfig() error {
	re := regexp.MustCompile(`^[a-zA-Z0-9\.\-]{3,}:\d{2,}$`)
	if !re.MatchString(SMTPListen) {
		return errors.New("SMTP bind should be in the format of <ip>:<port>")
	}
	if !re.MatchString(HTTPListen) {
		return errors.New("HTTP bind should be in the format of <ip>:<port>")
	}

	if AuthFile != "" {
		if !isFile(AuthFile) {
			return fmt.Errorf("password file not found: %s", AuthFile)
		}

		a, err := htpasswd.New(AuthFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		Auth = a
	}

	if SSLCert != "" && SSLKey == "" || SSLCert == "" && SSLKey != "" {
		return errors.New("you must provide both an SSL certificate and a key")
	}

	if SSLCert != "" {
		if !isFile(SSLCert) {
			return fmt.Errorf("SSL certificate not found: %s", SSLCert)
		}

		if !isFile(SSLKey) {
			return fmt.Errorf("SSL key not found: %s", SSLKey)
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
