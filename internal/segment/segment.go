// Package segment splits messages into isolated segments, one per SMTP username.
//
// Segments are opt-in via --segment-domain. When enabled, a message is stored in the
// segment named after the (normalized) username the client authenticated with, and the
// web UI and API serve a segment on <segment>.<segment-domain>. Any other host is served
// the default segment, which holds messages received without authentication.
package segment

import (
	"crypto/subtle"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/axllent/mailpit/config"
)

const (
	// Default is the segment for messages received without a username
	Default = ""

	// All is an internal value used to address all segments at once (eg: global statistics).
	// It can never be the result of Normalize() or FromHost().
	All = "*"

	// maxLength is the maximum length of a DNS label
	maxLength = 63
)

var invalidCharsRe = regexp.MustCompile(`[^a-z0-9\-]+`)

// Enabled returns whether segments are enabled
func Enabled() bool {
	return config.SegmentDomain != ""
}

// Normalize converts a username to a segment name which is safe to use as a DNS label:
// lowercase, a-z, 0-9 and dashes only, no leading or trailing dashes, max 63 characters.
// Every other character (including dots and underscores) is replaced with a dash.
func Normalize(username string) string {
	s := invalidCharsRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(username)), "-")
	s = strings.Trim(s, "-")
	if len(s) > maxLength {
		s = strings.Trim(s[:maxLength], "-")
	}

	return s
}

// FromUsername returns the segment for an authenticated username (nil or empty for none).
// It always returns the default segment when segments are disabled.
func FromUsername(username *string) string {
	if !Enabled() || username == nil {
		return Default
	}

	return Normalize(*username)
}

// FromHost returns the segment for an HTTP Host header value, eg: "alpha.mail.example.com:8025"
// returns "alpha" when the segment domain is "mail.example.com". A host that is not exactly one
// valid label below the segment domain returns the default segment.
func FromHost(host string) string {
	if !Enabled() {
		return Default
	}

	h := strings.ToLower(strings.TrimSpace(host))
	if hostOnly, _, err := net.SplitHostPort(h); err == nil {
		h = hostOnly
	}
	h = strings.TrimSuffix(h, ".")

	label, found := strings.CutSuffix(h, "."+config.SegmentDomain)
	if !found || label == "" || strings.Contains(label, ".") {
		return Default
	}

	// only accept labels that are already in their normalized form
	if Normalize(label) != label {
		return Default
	}

	return label
}

// FromRequest returns the segment for an HTTP request based on its Host header
func FromRequest(r *http.Request) string {
	return FromHost(r.Host)
}

// HostAllowed returns whether a host is the segment domain or a subdomain of it.
// This is used to admit segment hosts when --allowed-hosts is set.
func HostAllowed(host string) bool {
	if !Enabled() {
		return false
	}

	h := strings.ToLower(strings.TrimSpace(host))
	if hostOnly, _, err := net.SplitHostPort(h); err == nil {
		h = hostOnly
	}
	h = strings.TrimSuffix(h, ".")

	return h == config.SegmentDomain || strings.HasSuffix(h, "."+config.SegmentDomain)
}

// AuthEnabled returns whether segment authentication is active (--segment-password-env).
// SMTP and web UI logins are then the segment name plus one shared password.
func AuthEnabled() bool {
	return Enabled() && config.SegmentPassword != ""
}

func passwordMatches(password string) bool {
	return subtle.ConstantTimeCompare([]byte(password), []byte(config.SegmentPassword)) == 1
}

// AuthenticateUser checks an SMTP login: the username must be a valid segment name
// (lowercase letters, digits and dashes) and the password must be the shared password.
func AuthenticateUser(username, password string) bool {
	if !AuthEnabled() || username == "" || Normalize(username) != username {
		return false
	}

	return passwordMatches(password)
}

// AuthenticateRequest checks the basic auth login of an HTTP request: the username must
// be the segment of the request host, so a login for one segment never opens another.
// Hosts that are not a segment host (the default segment) are always refused.
func AuthenticateRequest(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || !AuthEnabled() {
		return false
	}

	seg := FromRequest(r)
	if seg == Default || subtle.ConstantTimeCompare([]byte(user), []byte(seg)) != 1 {
		return false
	}

	return passwordMatches(pass)
}
