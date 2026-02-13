package server

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/axllent/mailpit/internal/logger"
)

var (
	// AccessControlAllowOrigin CORS policy - set with flags/env
	AccessControlAllowOrigin string

	// CorsAllowOrigins are optional allowed origins by hostname, set via setCORSOrigins().
	corsAllowOrigins = make(map[string]bool)
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
		// allow same origin || is "*" is defined as an origin
		if asciiFoldString(u.Host) == asciiFoldString(r.Host) || allAllowed {
			return true
		}

		originHostFold := asciiFoldString(u.Hostname())
		if corsAllowOrigins[originHostFold] {
			return true
		}

		logger.Log().Warnf("[cors] blocking request from unauthorized origin: %s", u.Hostname())

		return false
	}

	return true
}

// SetCORSOrigins sets the allowed CORS origins from a comma-separated string.
// It does not consider port or protocol, only the hostname.
func setCORSOrigins() {
	corsAllowOrigins = make(map[string]bool)

	hosts := extractOrigins(AccessControlAllowOrigin)
	for _, host := range hosts {
		corsAllowOrigins[asciiFoldString(host)] = true
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

			origins = append(origins, u.Hostname())
		}
	}

	sort.Strings(origins)

	return origins
}
