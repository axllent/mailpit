package auth

import (
	"strings"

	"github.com/tg123/go-htpasswd"
)

// loadUnvalidated builds Credentials without load's entry validation.
func loadUnvalidated(s string) (*Credentials, error) {
	f, err := htpasswd.NewFromReader(strings.NewReader(s), htpasswd.DefaultSystems, nil)
	if err != nil {
		return nil, err
	}

	return &Credentials{file: f}, nil
}
