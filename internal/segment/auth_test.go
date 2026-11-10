package segment

import (
	"net/http/httptest"
	"testing"

	"github.com/axllent/mailpit/config"
)

func enableAuth(t *testing.T) {
	t.Helper()

	config.SegmentDomain = "mail.example.com"
	config.SegmentPassword = "shared"
	t.Cleanup(func() {
		config.SegmentDomain = ""
		config.SegmentPassword = ""
	})
}

func TestAuthEnabled(t *testing.T) {
	if AuthEnabled() {
		t.Fatal("auth must be off by default")
	}

	config.SegmentPassword = "shared"
	defer func() { config.SegmentPassword = "" }()
	if AuthEnabled() {
		t.Fatal("auth must be off without a segment domain")
	}

	enableAuth(t)
	if !AuthEnabled() {
		t.Fatal("auth must be on with a segment domain and password")
	}
}

func TestAuthenticateUser(t *testing.T) {
	enableAuth(t)

	tests := []struct {
		user, pass string
		ok         bool
	}{
		{"alpha", "shared", true},
		{"my-project-2", "shared", true},
		{"alpha", "wrong", false},
		{"alpha", "", false},
		{"", "shared", false},
		{"Alpha", "shared", false},
		{"alpha.beta", "shared", false},
		{"under_score", "shared", false},
		{"-alpha", "shared", false},
		{"alpha ", "shared", false},
	}

	for _, tc := range tests {
		if got := AuthenticateUser(tc.user, tc.pass); got != tc.ok {
			t.Errorf("AuthenticateUser(%q, %q) = %v, expected %v", tc.user, tc.pass, got, tc.ok)
		}
	}
}

func TestAuthenticateRequest(t *testing.T) {
	enableAuth(t)

	tests := []struct {
		host, user, pass string
		ok               bool
	}{
		{"alpha.mail.example.com", "alpha", "shared", true},
		{"alpha.mail.example.com:8025", "alpha", "shared", true},
		{"alpha.mail.example.com", "beta", "shared", false},
		{"beta.mail.example.com", "alpha", "shared", false},
		{"alpha.mail.example.com", "alpha", "wrong", false},
		{"mail.example.com", "", "shared", false},
		{"mail.example.com", "alpha", "shared", false},
		{"a.b.mail.example.com", "a", "shared", false},
		{"other.example.org", "alpha", "shared", false},
	}

	for _, tc := range tests {
		r := httptest.NewRequest("GET", "http://"+tc.host+"/", nil)
		r.Host = tc.host
		r.SetBasicAuth(tc.user, tc.pass)
		if got := AuthenticateRequest(r); got != tc.ok {
			t.Errorf("AuthenticateRequest(%s, %q, %q) = %v, expected %v", tc.host, tc.user, tc.pass, got, tc.ok)
		}
	}

	r := httptest.NewRequest("GET", "http://alpha.mail.example.com/", nil)
	if AuthenticateRequest(r) {
		t.Error("a request without credentials must be refused")
	}
}
