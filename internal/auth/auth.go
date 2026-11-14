// Package auth handles the web UI and SMTP authentication
package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/tg123/go-htpasswd"
	"golang.org/x/crypto/bcrypt"
)

// maxCredentialLength is the maximum length in bytes of a username or password that
// will be checked. Some supported hash formats (SHA-crypt, APR1) have a cost that grows
// with the password length, so unauthenticated clients must not be able to choose it.
// Bcrypt only uses the first 72 bytes of a password, so real passwords are far shorter.
const maxCredentialLength = 1024

// ErrNoCredentials is returned when credentials were supplied but contain no entries.
var ErrNoCredentials = errors.New("no credentials found")

var (
	// UICredentials passwords
	UICredentials *Credentials
	// SendAPICredentials passwords
	SendAPICredentials *Credentials
	// SMTPCredentials passwords
	SMTPCredentials *Credentials
	// POP3Credentials passwords
	POP3Credentials *Credentials

	whitespaceRe = regexp.MustCompile(`\s+`)
)

// Credentials is a set of usernames and password hashes.
type Credentials struct {
	file *htpasswd.File
}

// Match returns true if the username and password are valid. It fails closed: it returns
// false for a nil receiver, for values over the length limit, and if the underlying
// hash implementation panics.
func (c *Credentials) Match(username, password string) (ok bool) {
	if c == nil || c.file == nil {
		return false
	}

	if len(username) > maxCredentialLength || len(password) > maxCredentialLength {
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Log().Errorf("[auth] recovered from panic checking credentials for user %q: %v", username, r)
			ok = false
		}
	}()

	return c.file.Match(username, password)
}

// SetUIAuth will set Basic Auth credentials required for the UI & API
func SetUIAuth(s string) error {
	c, err := load(s)
	if err != nil {
		return err
	}

	UICredentials = c

	return nil
}

// SetSendAPIAuth will set Send API credentials
func SetSendAPIAuth(s string) error {
	c, err := load(s)
	if err != nil {
		return err
	}

	SendAPICredentials = c

	return nil
}

// SetSMTPAuth will set SMTP credentials
func SetSMTPAuth(s string) error {
	c, err := load(s)
	if err != nil {
		return err
	}

	SMTPCredentials = c

	return nil
}

// SetPOP3Auth will set POP3 server credentials
func SetPOP3Auth(s string) error {
	c, err := load(s)
	if err != nil {
		return err
	}

	POP3Credentials = c

	return nil
}

// load parses whitespace-separated "user:password" entries. It returns ErrNoCredentials
// if there are none, and an error if any entry is malformed, so that a typo cannot
// silently drop a user. Errors never include the entry text as it may contain a password.
func load(s string) (*Credentials, error) {
	entries := credentialsFromString(s)
	if len(entries) == 0 {
		return nil, ErrNoCredentials
	}

	var errs []error

	for _, entry := range entries {
		if err := validateEntry(entry); err != nil {
			errs = append(errs, err)
		}
	}

	invalid := 0
	file, err := htpasswd.NewFromReader(strings.NewReader(strings.Join(entries, "\n")), htpasswd.DefaultSystems, func(error) {
		invalid++
	})
	if err != nil {
		return nil, err
	}

	if invalid > 0 {
		errs = append(errs, fmt.Errorf("%d invalid credential entries (expected user:password or user:hash, without spaces)", invalid))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Credentials{file: file}, nil
}

// validateEntry checks hash formats that the htpasswd library accepts without validating,
// but which can never match (bcrypt) or can panic when used (SHA-crypt).
func validateEntry(entry string) (err error) {
	user, hash, found := strings.Cut(entry, ":")
	if !found {
		return nil // reported by the parser as an invalid entry
	}

	switch {
	case strings.HasPrefix(hash, "$2"):
		if _, cerr := bcrypt.Cost([]byte(hash)); cerr != nil {
			return fmt.Errorf("invalid bcrypt hash for user %q", user)
		}
	case strings.HasPrefix(hash, "$5$"), strings.HasPrefix(hash, "$6$"):
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("invalid SHA-crypt hash for user %q", user)
			}
		}()

		f, nerr := htpasswd.NewFromReader(strings.NewReader(entry), htpasswd.DefaultSystems, nil)
		if nerr != nil {
			return fmt.Errorf("invalid SHA-crypt hash for user %q", user)
		}

		f.Match(user, "")
	}

	return nil
}

func credentialsFromString(s string) []string {
	// split string by any whitespace character
	words := whitespaceRe.Split(s, -1)
	credentials := []string{}
	for _, w := range words {
		if w != "" {
			credentials = append(credentials, w)
		}
	}

	return credentials
}
