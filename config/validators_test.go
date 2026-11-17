package config

import (
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/smtpd/chaos"
)

func TestParseMaxAge(t *testing.T) {
	tests := []struct {
		value     string
		wantHours int
		wantErr   string
	}{
		{"", 0, ""},
		{"24h", 24, ""},
		{"7d", 168, ""},
		{"0d", 0, ""},
		{"abc", 0, "must be either"},
		{"7w", 0, "must be either"},

		// largest values that fit in a time.Duration (controls for the cases below)
		{"99999d", 2399976, ""},
		{"2562047h", 2562047, ""},
		{"106751d", 2562024, ""},

		// one past the limit, and values that previously overflowed in the prune job
		// and deleted every message (or wrapped negative and disabled pruning)
		{"2562048h", 0, "too large"},
		{"106752d", 0, "too large"},
		{"999999d", 0, "too large"},
		{"9999999h", 0, "too large"},
		{"384307168202282326d", 0, "too large"},
		{"99999999999999999999h", 0, "too large"},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			origAge, origHours := MaxAge, MaxAgeInHours
			t.Cleanup(func() { MaxAge, MaxAgeInHours = origAge, origHours })

			MaxAge, MaxAgeInHours = tc.value, 0

			err := parseMaxAge()

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error containing %q", err, tc.wantErr)
			}
			if MaxAgeInHours != tc.wantHours {
				t.Errorf("MaxAgeInHours = %d, want %d", MaxAgeInHours, tc.wantHours)
			}
		})
	}
}

func TestValidateRelayAndForwardConfig(t *testing.T) {
	tests := []struct {
		name           string
		port           int
		returnPath     string
		wantReturnPath string
		wantErr        string
	}{
		{"defaults", 0, "", "", ""},
		{"lowest port", 1, "", "", ""},
		{"highest port", 65535, "", "", ""},
		{"port too high", 65536, "", "", "port must be between"},
		{"negative port", -25, "", "", "port must be between"},
		{"plain return-path", 25, "bounce@example.com", "bounce@example.com", ""},
		{"bracketed return-path", 25, "<bounce@example.com>", "bounce@example.com", ""},
		{"named return-path", 25, "Bounce <bounce@example.com>", "bounce@example.com", ""},
		// the quotes are kept, as the address is sent as it is in MAIL FROM (it used to be "a b@example.com")
		{"quoted return-path", 25, `"a b"@example.com`, `"a b"@example.com`, ""},
		{"bracketed quoted return-path", 25, `<"a b"@example.com>`, `"a b"@example.com`, ""},
		{"return-path with line break", 25, "bounce@example.com\r\nX-Injected: 1", "", "return-path is not a valid"},
		{"return-path not an address", 25, "bounce", "", "return-path is not a valid"},
	}

	origRelay, origForward, origRelease := SMTPRelayConfig, SMTPForwardConfig, ReleaseEnabled
	t.Cleanup(func() {
		SMTPRelayConfig, SMTPForwardConfig, ReleaseEnabled = origRelay, origForward, origRelease
	})

	for _, tc := range tests {
		t.Run("relay "+tc.name, func(t *testing.T) {
			SMTPRelayConfig = SMTPRelayConfigStruct{Host: "localhost", Port: tc.port, ReturnPath: tc.returnPath}
			checkValidateResult(t, validateRelayConfig(), tc.wantErr, SMTPRelayConfig.ReturnPath, tc.wantReturnPath)
		})
		t.Run("forward "+tc.name, func(t *testing.T) {
			SMTPForwardConfig = SMTPForwardConfigStruct{Host: "localhost", Port: tc.port, ReturnPath: tc.returnPath, To: "to@example.com"}
			checkValidateResult(t, validateForwardConfig(), tc.wantErr, SMTPForwardConfig.ReturnPath, tc.wantReturnPath)
		})
	}
}

func checkValidateResult(t *testing.T, err error, wantErr, returnPath, wantReturnPath string) {
	t.Helper()
	if wantErr == "" {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if returnPath != wantReturnPath {
			t.Errorf("ReturnPath = %q, want %q", returnPath, wantReturnPath)
		}
		return
	}
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		t.Fatalf("error = %v, want error containing %q", err, wantErr)
	}
}

func TestParseChaosTriggers(t *testing.T) {
	origTriggers, origEnabled := ChaosTriggers, chaos.Enabled
	t.Cleanup(func() { ChaosTriggers, chaos.Enabled = origTriggers, origEnabled })

	tests := []struct {
		value   string
		wantErr string
	}{
		// probability 0 so the control does not leave an active trigger behind
		{"sender:451:0", ""},
		{"Sender:451:0,Recipients:451:0", ""},
		{"sender:451:0.5", "invalid argument"},
		{"sender:45:0", "invalid argument"},
		{"sender:451:101", "probability must be between"},
		{"unknown:451:0", "unknown key"},
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			ChaosTriggers = tc.value
			err := parseChaosTriggers()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := map[string]bool{
		"http://example.com/hook":  true,
		"https://example.com/hook": true,
		"HTTPS://example.com/hook": true,
		"httpfoo://example.com":    false,
		"https2://example.com":     false,
		"ftp://example.com":        false,
		"/relative/path":           false,
		"":                         false,
	}

	for in, want := range tests {
		if got := isValidURL(in); got != want {
			t.Errorf("isValidURL(%q) = %v, want %v", in, got, want)
		}
	}
}
