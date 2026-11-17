// Package mimeparser provides the shared enmime parser used by Mailpit.
//
// Go's mime.ParseMediaType silently drops RFC 2231 extended parameters (such as
// filename*=iso-8859-1”Cl%E9a.pdf) unless their charset is utf-8 or us-ascii. This package
// restores those parameters (preferring them over any plain or partial value) so attachments with such file names are not lost.
package mimeparser

import (
	"bytes"
	"maps"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jhillyerd/enmime/v2"
	"github.com/jhillyerd/enmime/v2/mediatype"
	"golang.org/x/text/encoding/ianaindex"
)

// New returns an enmime parser with Mailpit's options and RFC 2231 charset support.
// Additional options are applied after Mailpit's defaults.
func New(opts ...enmime.Option) *enmime.Parser {
	return enmime.NewParser(append([]enmime.Option{
		enmime.DisableCharacterDetection(true),
		enmime.MaxMIMEParts(500),
		enmime.SetCustomParseMediaType(ParseMediaType),
	}, opts...)...)
}

// ParseMediaType wraps enmime's tolerant media type parser, adding back any RFC 2231
// parameters in a charset other than utf-8 or us-ascii that the standard library dropped.
func ParseMediaType(ctype string) (string, map[string]string, []string, error) {
	mtype, params, invalid, err := mediatype.Parse(ctype)
	if err != nil || !strings.Contains(ctype, "*") {
		return mtype, params, invalid, err
	}

	// mediatype.Parse returns a nil map (and nil error) when the media type is unparseable,
	// eg: a non-token character before the first ';'
	if params == nil {
		params = make(map[string]string)
	}

	// overrides any partial value the standard library kept from later continuation sections
	maps.Copy(params, decodeExtendedParams(ctype))

	return mtype, params, invalid, nil
}

// segment is a single RFC 2231 parameter section, eg: name*0*=iso-8859-1”abc
type segment struct {
	index   int
	encoded bool
	value   string
}

// decodeExtendedParams returns the decoded values of all RFC 2231 extended parameters in a
// header value. Parameters that cannot be decoded are omitted.
func decodeExtendedParams(ctype string) map[string]string {
	groups := make(map[string][]segment)

	for _, part := range splitParams(ctype) {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)

		base, index, encoded, ok := parseKey(key)
		if !ok {
			continue
		}
		if strings.HasPrefix(val, `"`) && strings.HasSuffix(val, `"`) && len(val) >= 2 {
			val = val[1 : len(val)-1]
		}
		groups[base] = append(groups[base], segment{index, encoded, val})
	}

	results := make(map[string]string)

	for base, segs := range groups {
		sort.SliceStable(segs, func(i, j int) bool { return segs[i].index < segs[j].index })

		// only the first section carries the charset and language
		first := segs[0]
		if !first.encoded {
			continue
		}
		charset, rest, ok := splitCharset(first.value)
		if !ok {
			continue
		}
		if charset == "" || strings.EqualFold(charset, "utf-8") || strings.EqualFold(charset, "us-ascii") {
			continue // handled by the standard library
		}

		var raw bytes.Buffer
		valid := true
		for i, s := range segs {
			v := s.value
			if i == 0 {
				v = rest
			}
			if s.encoded {
				dec, err := url.PathUnescape(v)
				if err != nil {
					valid = false
					break
				}
				v = dec
			}
			raw.WriteString(v)
		}
		if valid {
			results[base] = decodeCharset(charset, raw.Bytes())
		}
	}

	return results
}

// parseKey parses a parameter key into its base name, section index and whether it is encoded.
func parseKey(key string) (base string, index int, encoded, ok bool) {
	if !strings.Contains(key, "*") {
		return "", 0, false, false
	}
	base, suffix, _ := strings.Cut(key, "*")
	if base == "" {
		return "", 0, false, false
	}
	// suffix is one of: "" (name*), "N" (name*N), "N*" (name*N*)
	if suffix == "" {
		return base, 0, true, true
	}
	if strings.HasSuffix(suffix, "*") {
		encoded = true
		suffix = strings.TrimSuffix(suffix, "*")
	}
	index, err := strconv.Atoi(suffix)
	if err != nil || index < 0 {
		return "", 0, false, false
	}
	return base, index, encoded, true
}

// splitCharset splits "charset'language'value" into the charset and the value.
func splitCharset(s string) (charset, value string, ok bool) {
	charset, rest, ok := strings.Cut(s, "'")
	if !ok {
		return "", "", false
	}
	_, value, ok = strings.Cut(rest, "'")
	return charset, value, ok
}

// decodeCharset converts raw bytes to UTF-8, falling back to stripping invalid bytes when
// the charset is unknown.
func decodeCharset(charset string, raw []byte) string {
	enc, err := ianaindex.MIME.Encoding(charset)
	if err == nil && enc != nil {
		if out, err := enc.NewDecoder().Bytes(raw); err == nil {
			return string(out)
		}
	}
	if utf8.Valid(raw) {
		return string(raw)
	}
	return strings.ToValidUTF8(string(raw), "")
}

// splitParams splits a header value on semicolons that are not inside quotes.
func splitParams(s string) []string {
	var (
		parts []string
		start int
		quote bool
	)
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			if quote {
				i++
			}
		case '"':
			quote = !quote
		case ';':
			if !quote {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, s[start:])
}
