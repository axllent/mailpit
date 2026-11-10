package segment

import (
	"net/http/httptest"
	"testing"

	"github.com/axllent/mailpit/config"
)

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"alpha":             "alpha",
		"Alpha":             "alpha",
		" Beta ":            "beta",
		"project.client":    "project-client",
		"user@example.com":  "user-example-com",
		"under_score":       "under-score",
		"--dashes--":        "dashes",
		"ünïcode":           "n-code",
		"":                  "",
		"!!!":               "",
		"a  b":              "a-b",
		"0123456789-abcdef": "0123456789-abcdef",
	}

	for in, expected := range tests {
		if got := Normalize(in); got != expected {
			t.Errorf("Normalize(%q) = %q, expected %q", in, got, expected)
		}
	}

	long := ""
	for range 70 {
		long += "a"
	}
	if got := Normalize(long); len(got) != 63 {
		t.Errorf("expected a max length of 63, got %d", len(got))
	}
}

func TestFromUsername(t *testing.T) {
	defer func() { config.SegmentDomain = "" }()

	user := "Alpha"

	config.SegmentDomain = ""
	if got := FromUsername(&user); got != Default {
		t.Errorf("expected default segment when disabled, got %q", got)
	}

	config.SegmentDomain = "mail.example.com"
	if got := FromUsername(&user); got != "alpha" {
		t.Errorf("expected \"alpha\", got %q", got)
	}
	if got := FromUsername(nil); got != Default {
		t.Errorf("expected default segment for no username, got %q", got)
	}
	empty := ""
	if got := FromUsername(&empty); got != Default {
		t.Errorf("expected default segment for empty username, got %q", got)
	}
}

func TestFromHost(t *testing.T) {
	defer func() { config.SegmentDomain = "" }()

	config.SegmentDomain = ""
	if got := FromHost("alpha.mail.example.com"); got != Default {
		t.Errorf("expected default segment when disabled, got %q", got)
	}

	config.SegmentDomain = "mail.example.com"

	tests := map[string]string{
		"alpha.mail.example.com":       "alpha",
		"ALPHA.Mail.Example.com":       "alpha",
		"alpha.mail.example.com:8925":  "alpha",
		"alpha.mail.example.com.":      "alpha",
		"beta-2.mail.example.com:443":  "beta-2",
		"mail.example.com":             Default,
		"mail.example.com:8925":        Default,
		"a.b.mail.example.com":         Default,
		"alphamail.example.com":        Default,
		"alpha.mail.example.com.evil":  Default,
		"alpha.example.com":            Default,
		"localhost:8025":               Default,
		"127.0.0.1:8025":               Default,
		"[::1]:8025":                   Default,
		"under_score.mail.example.com": Default,
		"-alpha.mail.example.com":      Default,
		"":                             Default,
	}

	for in, expected := range tests {
		if got := FromHost(in); got != expected {
			t.Errorf("FromHost(%q) = %q, expected %q", in, got, expected)
		}
	}

	r := httptest.NewRequest("GET", "http://alpha.mail.example.com:8925/api/v1/messages", nil)
	if got := FromRequest(r); got != "alpha" {
		t.Errorf("FromRequest() = %q, expected \"alpha\"", got)
	}
}

func TestHostAllowed(t *testing.T) {
	defer func() { config.SegmentDomain = "" }()

	config.SegmentDomain = "mail.example.com"

	tests := map[string]bool{
		"mail.example.com":            true,
		"alpha.mail.example.com:8925": true,
		"a.b.mail.example.com":        true,
		"evilmail.example.com":        false,
		"example.com":                 false,
	}

	for in, expected := range tests {
		if got := HostAllowed(in); got != expected {
			t.Errorf("HostAllowed(%q) = %v, expected %v", in, got, expected)
		}
	}
}
