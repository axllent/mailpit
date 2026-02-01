package server

import (
	"net/http"
	"testing"
)

func TestExtractOrigins(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single hostname",
			input:    "example.com",
			expected: []string{"example.com"},
		},
		{
			name:     "multiple hostnames comma separated",
			input:    "example.com,foo.com",
			expected: []string{"example.com", "foo.com"},
		},
		{
			name:     "multiple hostnames space separated",
			input:    "example.com foo.com",
			expected: []string{"example.com", "foo.com"},
		},
		{
			name:     "wildcard",
			input:    "*",
			expected: []string{"*"},
		},
		{
			name:     "mixed protocols",
			input:    "http://example.com,https://foo.com:8080",
			expected: []string{"example.com", "foo.com"},
		},
		{

			name:     "embedded wildcard",
			input:    "http://example.com,*,https://test",
			expected: []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractOrigins(tt.input)

			if len(got) != len(tt.expected) {
				t.Errorf("expected %d origins, got %d", len(tt.expected), len(got))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("expected origin %q, got %q", tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestCorsOriginAccessControl(t *testing.T) {
	// Setup allowed origins
	AccessControlAllowOrigin = "example.com,foo.com,bar.com"
	setCORSOrigins()

	tests := []struct {
		name   string
		origin string
		host   string
		allow  bool
	}{
		{"no origin header", "", "example.com", true},
		{"allowed origin", "http://example.com:1234", "mailpit.local", true},
		{"not allowed origin", "http://notallowed.com", "mailpit.local", false},
		{"allowed by hostname", "http://foo.com", "mailpit.local", true},
		{"ascii fold: allowed origin uppercase", "HTTP://EXAMPLE.COM", "mailpit.local", true},
		{"ascii fold: allowed by hostname uppercase", "HTTP://FOO.COM", "mailpit.local", true},
		{"ascii fold: host uppercase", "http://example.com", "MAILPIT.LOCAL", true},
		{"ascii fold: not allowed origin uppercase", "HTTP://NOTALLOWED.COM", "mailpit.local", false},
		{"ascii fold: mixed case", "HtTp://ExAmPlE.CoM", "mailpit.local", true},
		{"non-ascii: allowed origin (unicode hostname)", "http://exámple.com", "mailpit.local", false},
		{"non-ascii: allowed by hostname (unicode)", "http://föö.com", "mailpit.local", false},
		{"non-ascii: host uppercase (unicode)", "http://exámple.com", "MAILPIT.LOCAL", false},
		{"non-ascii: mixed case (unicode)", "HtTp://ExÁmPlE.CoM", "mailpit.local", false},
	}

	// Add wildcard test
	AccessControlAllowOrigin = "*"
	setCORSOrigins()
	reqWildcard := &http.Request{Header: http.Header{"Origin": {"http://any.com"}}, Host: "mailpit.local"}
	if !corsOriginAccessControl(reqWildcard) {
		t.Error("Wildcard origin should be allowed")
	}

	// Reset to specific hosts
	AccessControlAllowOrigin = "example.com,foo.com,bar.com"
	setCORSOrigins()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{Header: http.Header{}, Host: tt.host}
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			allowed := corsOriginAccessControl(req)
			if allowed != tt.allow {
				t.Errorf("expected allowed=%v, got %v for origin=%q host=%q", tt.allow, allowed, tt.origin, tt.host)
			}
		})
	}
}
