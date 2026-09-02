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

// TestHostAccessControl verifies the Host-header allowlist that gates the
// middleware ahead of the CORS check. This is the anchor that a DNS-rebinding
// attacker cannot satisfy: the same-origin branch of corsOriginAccessControl
// compares two client-supplied values (Host and Origin), so it admits a
// rebound request where Host and Origin both carry an attacker-controlled name;
// hostAccessControl rejects such a request outright when an allowlist is set.
func TestHostAccessControl(t *testing.T) {
	origAllowedHosts := AllowedHosts
	defer func() {
		AllowedHosts = origAllowedHosts
		setAllowedHosts()
	}()

	tests := []struct {
		name         string
		allowedHosts string
		host         string
		allow        bool
	}{
		// Unset: preserves prior behaviour, every Host is admitted so no
		// existing deployment breaks after the upgrade.
		{"unset admits any host", "", "attacker.example:8025", true},
		{"unset admits empty host", "", "", true},

		// Configured allowlist: exact host:port match.
		{"exact match host:port", "mailpit.local:8025", "mailpit.local:8025", true},
		{"exact match host:port wrong port", "mailpit.local:8025", "mailpit.local:9000", false},
		{"exact match host:port different host", "mailpit.local:8025", "evil.example:8025", false},

		// Configured allowlist entry without port: matches any port.
		{"bare host matches with port", "mailpit.local", "mailpit.local:8025", true},
		{"bare host matches without port", "mailpit.local", "mailpit.local", true},
		{"bare host rejects other host", "mailpit.local", "evil.example:8025", false},

		// Loopback is always admitted so local dev workflows still work when
		// the operator scopes the allowlist to a hostname.
		{"loopback admitted implicitly (localhost)", "mailpit.local", "localhost:8025", true},
		{"loopback admitted implicitly (127.0.0.1)", "mailpit.local", "127.0.0.1:8025", true},
		{"loopback admitted implicitly (IPv6)", "mailpit.local", "[::1]:8025", true},

		// Raw IP literals in Host are always admitted regardless of the
		// allowlist: DNS rebinding cannot produce an IP in the Host header
		// (rebinding needs a DNS name, and the browser puts the URL host into
		// the Host header, not the resolved address). Cross-origin fetches
		// that target an IP directly are stopped by the CORS check.
		{"LAN IPv4 admitted", "mailpit.local", "192.168.1.5:8025", true},
		{"LAN IPv4 without port admitted", "mailpit.local", "192.168.1.5", true},
		{"public IPv4 admitted", "mailpit.local", "203.0.113.5:8025", true},
		{"IPv6 literal admitted", "mailpit.local", "[2001:db8::1]:8025", true},

		// Case folding on both sides.
		{"case-insensitive host match", "mailpit.local:8025", "MAILPIT.LOCAL:8025", true},
		{"case-insensitive allowlist entry", "MAILPIT.LOCAL:8025", "mailpit.local:8025", true},

		// The rebinding shape: attacker DNS name in Host does not match a
		// hostname allowlist scoped to the operator's own deployment.
		{"rebinding shape rejected", "mailpit.local", "attacker.example:8025", false},

		// Multiple entries (comma-separated, mixed with/without port).
		{"multiple entries first", "mail.a,mail.b:8025", "mail.a:1234", true},
		{"multiple entries second", "mail.a,mail.b:8025", "mail.b:8025", true},
		{"multiple entries reject", "mail.a,mail.b:8025", "mail.c:8025", false},

		// Scheme in the entry is tolerated (users often paste URLs).
		{"scheme in entry tolerated", "http://mailpit.local:8025", "mailpit.local:8025", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AllowedHosts = tt.allowedHosts
			setAllowedHosts()
			req := &http.Request{Host: tt.host}
			got := hostAccessControl(req)
			if got != tt.allow {
				t.Errorf("hostAccessControl(host=%q, allowed=%q) = %v, want %v",
					tt.host, tt.allowedHosts, got, tt.allow)
			}
		})
	}
}

// TestMiddlewareRejectsRebindingHost verifies end-to-end that when
// --allowed-hosts is set, middleWareFunc rejects a request carrying an
// attacker-controlled Host header even if it satisfies the same-origin CORS
// branch (Host and Origin matching each other). Without --allowed-hosts,
// the same request continues to be admitted (no behaviour change for
// existing deployments).
func TestMiddlewareRejectsRebindingHost(t *testing.T) {
	origAllowOrigin := AccessControlAllowOrigin
	origAllowedHosts := AllowedHosts
	origWebroot := config.Webroot
	origRe := htmlPreviewRouteRe
	defer func() {
		AccessControlAllowOrigin = origAllowOrigin
		AllowedHosts = origAllowedHosts
		config.Webroot = origWebroot
		htmlPreviewRouteRe = origRe
		setCORSOrigins()
		setAllowedHosts()
	}()

	AccessControlAllowOrigin = ""
	config.Webroot = "/"
	setCORSOrigins()

	makeReq := func(host, origin string) *http.Request {
		req := &http.Request{
			Method:     "GET",
			RequestURI: "/api/v1/messages",
			URL:        &url.URL{Path: "/api/v1/messages"},
			Header:     http.Header{},
			Host:       host,
			Body:       http.NoBody,
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		return req.WithContext(context.Background())
	}

	run := func(name string, req *http.Request, wantStatus int) {
		t.Run(name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handler := middleWareFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			handler(w, req)
			if w.Code != wantStatus {
				t.Errorf("expected HTTP %d, got %d", wantStatus, w.Code)
			}
		})
	}

	// Baseline: no allowlist configured, the rebinding shape is admitted by
	// the same-origin branch of the CORS check. This is the deployment shape
	// existing users rely on and must not regress when they don't opt in.
	AllowedHosts = ""
	setAllowedHosts()
	run("no allowlist admits rebinding shape",
		makeReq("attacker.example:8025", "http://attacker.example:8025"),
		http.StatusOK)

	// With allowlist configured to the operator's own deployment, the same
	// rebinding shape is rejected at the Host gate.
	AllowedHosts = "mailpit.local:8025"
	setAllowedHosts()
	run("allowlist rejects rebinding shape",
		makeReq("attacker.example:8025", "http://attacker.example:8025"),
		http.StatusForbidden)

	// Legitimate access to the allowlisted host still works.
	run("allowlist admits configured host",
		makeReq("mailpit.local:8025", "http://mailpit.local:8025"),
		http.StatusOK)

	// Loopback access still works even when allowlist is scoped to a name.
	run("allowlist admits loopback",
		makeReq("localhost:8025", "http://localhost:8025"),
		http.StatusOK)
}
