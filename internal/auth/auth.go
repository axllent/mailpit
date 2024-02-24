// Package auth handles the web UI and SMTP authentication
package auth

import (
	"regexp"
	"strings"

	"github.com/tg123/go-htpasswd"
)

var (
	// UICredentials passwords
	UICredentials *htpasswd.File
	// SMTPCredentials passwords
	SMTPCredentials *htpasswd.File
	// POP3Credentials passwords
	POP3Credentials *htpasswd.File
)

// SetUIAuth will set Basic Auth credentials required for the UI & API
func SetUIAuth(s string) error {
	var err error

	credentials := credentialsFromString(s)
	if len(credentials) == 0 {
		return nil
	}

	r := strings.NewReader(strings.Join(credentials, "\n"))

	UICredentials, err = htpasswd.NewFromReader(r, htpasswd.DefaultSystems, nil)
	if err != nil {
		return err
	}

	return nil
}

// SetSMTPAuth will set SMTP credentials
func SetSMTPAuth(s string) error {
	var err error

	credentials := credentialsFromString(s)
	if len(credentials) == 0 {
		return nil
	}

	r := strings.NewReader(strings.Join(credentials, "\n"))

	SMTPCredentials, err = htpasswd.NewFromReader(r, htpasswd.DefaultSystems, nil)
	if err != nil {
		return err
	}

	return nil
}

// SetPOP3Auth will set POP3 server credentials
func SetPOP3Auth(s string) error {
	var err error

	credentials := credentialsFromString(s)
	if len(credentials) == 0 {
		return nil
	}

	r := strings.NewReader(strings.Join(credentials, "\n"))

	POP3Credentials, err = htpasswd.NewFromReader(r, htpasswd.DefaultSystems, nil)
	if err != nil {
		return err
	}

	return nil
}

func credentialsFromString(s string) []string {
	// split string by any whitespace character
	re := regexp.MustCompile(`\s+`)

	words := re.Split(s, -1)
	credentials := []string{}
	for _, w := range words {
		if w != "" {
			credentials = append(credentials, w)
		}
	}

	return credentials
}
