package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/axllent/mailpit/config"
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
			expected: []string{"example.com", "foo.com:8080"},
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
		// example.com:1234 must NOT be admitted by an allowlist entry for example.com (different port)
		{"allowed origin", "http://example.com:1234", "mailpit.local", false},
		{"allowed origin", "http://example.com:1234", "example.com", false},
		{"allowed origin", "http://example.com:1234", "example.com:1234", true},
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

// TestCORSMiddlewarePathCheck verifies that middleWareFunc keys its CORS gate on
// r.URL.Path (the percent-decoded value Go's ServeMux routes on) rather than
// r.RequestURI (the raw wire bytes). The two diverge when the client sends a
// percent-encoded path such as /%61pi/events: ServeMux decodes %61 to 'a' and
// routes to /api/events, but r.RequestURI remains /%61pi/events. Using
// r.RequestURI for the prefix check allowed an attacker to bypass the origin
// gate while still reaching the WebSocket handler.
//
// The test runs the full case matrix for both the default webroot ("/") and a
// non-default webroot ("/mailpit/") to confirm the fix holds regardless of
// deployment path configuration.
func TestCORSMiddlewarePathCheck(t *testing.T) {
	// Save and restore global state.
	origAllowOrigin := AccessControlAllowOrigin
	origWebroot := config.Webroot
	origRe := htmlPreviewRouteRe
	defer func() {
		AccessControlAllowOrigin = origAllowOrigin
		config.Webroot = origWebroot
		htmlPreviewRouteRe = origRe
		setCORSOrigins()
	}()

	AccessControlAllowOrigin = "allowed.example"
	setCORSOrigins()

	// makeReq simulates what Go's HTTP server sets on the request after parsing
	// a raw wire target. RequestURI is the unmodified wire value; URL.Path is
	// the percent-decoded path that ServeMux uses for routing.
	makeReq := func(rawURI, decodedPath, origin string) *http.Request {
		req := &http.Request{
			Method:     "GET",
			RequestURI: rawURI,
			URL:        &url.URL{Path: decodedPath},
			Header:     http.Header{},
			Host:       "localhost:8025",
			Body:       http.NoBody,
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return req.WithContext(context.Background())
	}

	type testCase struct {
		name        string
		rawURI      string // r.RequestURI — raw wire bytes
		decodedPath string // r.URL.Path  — what ServeMux routes on
		origin      string
		wantStatus  int
	}

	// casesFor returns the test matrix for a given webroot (e.g. "/" or "/mailpit/").
	// encodedWR is the same prefix with its first letter percent-encoded
	// (e.g. "/%6dailpit/" for "/mailpit/"), used to exercise the case where the
	// webroot segment itself is encoded.
	casesFor := func(wr, encodedWR string) []testCase {
		return []testCase{
			{
				// Control: canonical path, blocked origin.
				name:        "canonical path blocks cross-origin",
				rawURI:      wr + "api/events",
				decodedPath: wr + "api/events",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Regression: api segment encoded as %61pi. rawURI does not match
				// the prefix check but decodedPath does, so with the fix the CORS
				// check fires and blocks the request. With the old r.RequestURI
				// code this returned 200.
				name:        "encoded api segment (%61pi) blocks cross-origin (regression)",
				rawURI:      wr + "%61pi/events",
				decodedPath: wr + "api/events",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Encoded webroot segment: ServeMux decodes and routes normally;
				// CORS gate must still fire.
				name:        "encoded webroot segment blocks cross-origin",
				rawURI:      encodedWR + "api/events",
				decodedPath: wr + "api/events",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Configured origin is allowed even via the encoded path.
				name:        "encoded api segment allows configured origin",
				rawURI:      wr + "%61pi/events",
				decodedPath: wr + "api/events",
				origin:      "http://allowed.example",
				wantStatus:  http.StatusOK,
			},
			{
				// No Origin header: not a cross-origin request, always allowed.
				name:        "no origin header passes",
				rawURI:      wr + "api/events",
				decodedPath: wr + "api/events",
				origin:      "",
				wantStatus:  http.StatusOK,
			},
			{
				// Same host as the server: always allowed.
				name:        "same-host origin passes",
				rawURI:      wr + "api/events",
				decodedPath: wr + "api/events",
				origin:      "http://localhost:8025",
				wantStatus:  http.StatusOK,
			},
			{
				// HTML preview route: htmlPreviewRouteRe matches this path so the
				// CORS gate must fire, using the decoded URL.Path and the webroot-
				// anchored regex built from config.Webroot.
				name:        "html preview route blocks cross-origin",
				rawURI:      wr + "view/abc123.html",
				decodedPath: wr + "view/abc123.html",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Encoded view segment: analogous to the %61pi API bypass but for
				// the preview route. %76 = 'v'; decoded path is /view/abc123.html
				// which matches htmlPreviewRouteRe, so CORS must fire.
				name:        "encoded view segment in preview route blocks cross-origin",
				rawURI:      wr + "%76iew/abc123.html",
				decodedPath: wr + "view/abc123.html",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Encoded .html extension: %6c = 'l'; decoded path is
				// /view/abc123.html which matches htmlPreviewRouteRe.
				name:        "encoded html extension in preview route blocks cross-origin",
				rawURI:      wr + "view/abc123.htm%6c",
				decodedPath: wr + "view/abc123.html",
				origin:      "http://evil.example",
				wantStatus:  http.StatusForbidden,
			},
			{
				// Non-preview /view/ path: does not match htmlPreviewRouteRe
				// (no .html suffix), so the CORS gate does not apply.
				name:        "non-preview view path not subject to CORS gate",
				rawURI:      wr + "view/index",
				decodedPath: wr + "view/index",
				origin:      "http://evil.example",
				wantStatus:  http.StatusOK,
			},
		}
	}

	roots := []struct {
		name      string
		webroot   string
		encodedWR string // same prefix with one letter of the first segment percent-encoded
	}{
		{name: "default webroot", webroot: "/", encodedWR: "/"},
		// %70 = 'p'; /mail%70it/ decodes to /mailpit/
		{name: "custom webroot", webroot: "/mailpit/", encodedWR: "/mail%70it/"},
	}

	for _, wr := range roots {
		t.Run(wr.name, func(t *testing.T) {
			config.Webroot = wr.webroot
			htmlPreviewRouteRe = nil // force recompile for new webroot

			for _, tt := range casesFor(wr.webroot, wr.encodedWR) {
				t.Run(tt.name, func(t *testing.T) {
					w := httptest.NewRecorder()
					req := makeReq(tt.rawURI, tt.decodedPath, tt.origin)

					handler := middleWareFunc(func(w http.ResponseWriter, _ *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
					handler(w, req)

					if w.Code != tt.wantStatus {
						t.Errorf("expected HTTP %d, got %d (rawURI=%q origin=%q)",
							tt.wantStatus, w.Code, tt.rawURI, tt.origin)
					}
				})
			}
		})
	}
}
