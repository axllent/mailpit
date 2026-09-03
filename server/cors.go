package server

import (
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
)

var (
	// AccessControlAllowOrigin CORS policy - set with flags/env
	AccessControlAllowOrigin string

	// AllowedHosts is an optional comma-separated allowlist of Host header
	// values, set via flags/env. When non-empty, requests carrying a Host
	// header outside the list are rejected before any other check runs.
	// This is a defense-in-depth measure against DNS-rebinding attacks: an
	// attacker who rebinds a name they control to Mailpit's listen address
	// can otherwise satisfy the same-origin comparison in
	// corsOriginAccessControl, because both the Origin and Host headers
	// then come from the same attacker-controlled context.
	AllowedHosts string

	// CorsAllowOrigins are optional allowed origins by hostname, set via setCORSOrigins().
	corsAllowOrigins = make(map[string]bool)

	// allowedHostsExact holds entries from AllowedHosts that include a port.
	// A Host header must match host:port exactly (case-folded) to be admitted.
	allowedHostsExact = make(map[string]bool)

	// allowedHostsBare holds entries from AllowedHosts that omit a port, plus
	// the loopback names/addresses which are always trusted (an attacker
	// cannot DNS-rebind onto loopback). Any Host header whose hostname
	// portion matches (case-folded) is admitted regardless of port.
	allowedHostsBare = make(map[string]bool)
)

// equalASCIIFold reports whether s and t, interpreted as UTF-8 strings, are equal
// under Unicode case folding, ignoring any difference in length.
func asciiFoldString(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		b[i] = toLowerASCIIFold(s[i])
	}
	return string(b)
}

// toLowerASCIIFold returns the Unicode case-folded equivalent of the ASCII character c.
// It is equivalent to the Unicode 13.0.0 function foldCase(c, CaseFoldingMapping).
func toLowerASCIIFold(c byte) byte {
	if 'A' <= c && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// hostAccessControl checks whether the request's Host header is admitted by
// the configured allowlist. When AllowedHosts is unset the check is a no-op
// (any Host is admitted, preserving existing behaviour). When configured, a
// Host outside the allowlist is rejected before Origin is even inspected: the
// same-origin branch in corsOriginAccessControl compares two client-supplied
// values (Host and Origin), so an attacker who can DNS-rebind a name they
// control could otherwise satisfy it. Gating on Host anchors the decision on
// a value the operator has independently declared trustworthy.
//
// Match semantics: entries with a port match host:port exactly (case-folded);
// entries without a port match any port. Loopback names/addresses are always
// admitted because they cannot be reached from a rebinding attacker (no one
// controls DNS for "localhost" / 127.0.0.1 / ::1 from outside the machine).
// Requests whose Host header is a raw IP literal are also admitted: DNS
// rebinding produces a Host containing the attacker's DNS name, never a raw
// IP, and a cross-origin fetch that targets an IP directly is already stopped
// by the CORS same-origin check on the Origin header.
func hostAccessControl(r *http.Request) bool {
	if len(allowedHostsExact) == 0 && len(allowedHostsBare) == 0 {
		return true
	}

	hostFold := asciiFoldString(r.Host)
	if allowedHostsExact[hostFold] {
		return true
	}

	bare, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		bare = r.Host
	}
	if allowedHostsBare[asciiFoldString(bare)] {
		return true
	}
	// Raw IP literals in the Host header cannot originate from a DNS-rebinding
	// attack, so they are admitted without needing an allowlist entry. Strip
	// IPv6 brackets before parsing.
	if ip := net.ParseIP(strings.Trim(bare, "[]")); ip != nil {
		return true
	}

	logger.Log().Warnf("[host] blocking request with disallowed Host header: %s", r.Host)
	return false
}

// CorsOriginAccessControl checks if the request origin is allowed based on the configured CORS origins.
func corsOriginAccessControl(r *http.Request) bool {
	origin := r.Header["Origin"]

	if len(origin) != 0 {
		u, err := url.Parse(origin[0])
		if err != nil {
			logger.Log().Errorf("[cors] origin parse error: %v", err)
			return false
		}

		_, allAllowed := corsAllowOrigins["*"]
		// allow same origin, or if "*" is defined as an origin
		if asciiFoldString(u.Host) == asciiFoldString(r.Host) || allAllowed {
			return true
		}

		// match on full host:port so that example.com:8080 is not admitted
		// by an allowlist entry for example.com (standard port 80/443).
		originHostFold := asciiFoldString(u.Host)
		if corsAllowOrigins[originHostFold] {
			return true
		}

		logger.Log().Warnf("[cors] blocking request from unauthorized origin: %s", u.Host)

		return false
	}

	return true
}

// setAllowedHosts parses the AllowedHosts string into the maps consumed by
// hostAccessControl. Loopback names/addresses are always seeded into the
// bare-hostname set so that local development flows remain unaffected when
// an operator opts into the strict check.
func setAllowedHosts() {
	allowedHostsExact = make(map[string]bool)
	allowedHostsBare = make(map[string]bool)

	entries := strings.FieldsFunc(strings.TrimSpace(AllowedHosts), func(r rune) bool {
		return r == ',' || r == ' '
	})

	if len(entries) == 0 {
		return
	}

	for _, loopback := range []string{"localhost", "127.0.0.1", "::1"} {
		allowedHostsBare[asciiFoldString(loopback)] = true
	}

	for _, entry := range entries {
		h := strings.TrimSpace(entry)
		if h == "" {
			continue
		}

		h = strings.TrimPrefix(h, "http://")
		h = strings.TrimPrefix(h, "https://")

		if _, _, err := net.SplitHostPort(h); err == nil {
			allowedHostsExact[asciiFoldString(h)] = true
		} else {
			allowedHostsBare[asciiFoldString(h)] = true
		}
	}

	keys := make([]string, 0, len(allowedHostsExact)+len(allowedHostsBare))
	for k := range allowedHostsExact {
		keys = append(keys, k)
	}
	for k := range allowedHostsBare {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	logger.Log().Infof("[host] allowed Host headers: %v", strings.Join(keys, ", "))
}

// SetCORSOrigins sets the allowed CORS origins from a comma-separated string.
// Origins are matched on the full host:port, so example.com and example.com:8080
// are treated as distinct origins.
func setCORSOrigins() {
	corsAllowOrigins = make(map[string]bool)

	hosts := extractOrigins(AccessControlAllowOrigin)
	for _, host := range hosts {
		corsAllowOrigins[asciiFoldString(host)] = true
	}

	if len(corsAllowOrigins) == 0 {
		return
	}

	if _, wildCard := corsAllowOrigins["*"]; wildCard {
		// reset to just wildcard
		corsAllowOrigins = make(map[string]bool)
		corsAllowOrigins["*"] = true
		logger.Log().Info("[cors] all origins are allowed due to wildcard \"*\"")
	} else {
		keys := make([]string, 0)
		for k := range corsAllowOrigins {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		logger.Log().Infof("[cors] allowed API origins: %v", strings.Join(keys, ", "))
	}
}

// extractOrigins extracts and returns a sorted list of origins from a comma-separated string.
func extractOrigins(str string) []string {
	origins := make([]string, 0)
	s := strings.TrimSpace(str)
	if s == "" {
		return origins
	}

	hosts := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' '
	})

	for _, host := range hosts {
		h := strings.TrimSpace(host)
		if h != "" {
			if h == "*" {
				return []string{"*"}
			}

			if !strings.HasPrefix(h, "http://") && !strings.HasPrefix(h, "https://") {
				h = "http://" + h
			}

			u, err := url.Parse(h)
			if err != nil || u.Hostname() == "" || strings.Contains(h, "*") {
				logger.Log().Warnf("[cors] invalid CORS origin \"%s\", ignoring", h)
				continue
			}

			// Store host:port so port differences are respected.
			// u.Host equals u.Hostname() when no port is present.
			origins = append(origins, u.Host)
		}
	}

	sort.Strings(origins)

	return origins
}
