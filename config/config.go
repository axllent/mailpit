package config

import (
	"errors"
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

	// SSLCert @TODO
	SSLCert string
	// SSLKey @TODO
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
		a, err := htpasswd.New(AuthFile, htpasswd.DefaultSystems, nil)
		if err != nil {
			return err
		}
		Auth = a
	}

	return nil
}
