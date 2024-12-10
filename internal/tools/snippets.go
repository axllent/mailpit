package tools

import (
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/html2text"
)

// CreateSnippet returns a message snippet. It will use the HTML version (if it exists)
// otherwise the text version.
func CreateSnippet(text, html string) string {
	text = strings.TrimSpace(text)
	html = strings.TrimSpace(html)
	limit := 200
	spaceRe := regexp.MustCompile(`\s+`)

	if text == "" && html == "" {
		return ""
	}

	if html != "" {
		data := html2text.Strip(html, false)

		if len(data) <= limit {
			return data
		}

		return truncate(data, limit) + "..."
	}

	if text != "" {
		// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
		text = strings.ReplaceAll(text, string('\uFEFF'), " ")
		text = strings.TrimSpace(spaceRe.ReplaceAllString(text, " "))
		if len(text) <= limit {
			return text
		}

		return truncate(text, limit) + "..."
	}

	return ""
}

// Truncate a string allowing for multi-byte encoding.
// Shamelessly borrowed from Tailscale.
// See https://github.com/tailscale/tailscale/blob/main/util/truncate/truncate.go
func truncate(s string, n int) string {
	if n >= len(s) {
		return s
	}

	// Back up until we find the beginning of a UTF-8 encoding.
	for n > 0 && s[n-1]&0xc0 == 0x80 { // 0x10... is a continuation byte
		n--
	}

	// If we're at the beginning of a multi-byte encoding, back up one more to
	// skip it. It's possible the value was already complete, but it's simpler
	// if we only have to check in one direction.
	//
	// Otherwise, we have a single-byte code (0x00... or 0x01...).
	if n > 0 && s[n-1]&0xc0 == 0xc0 { // 0x11... starts a multibyte encoding
		n--
	}

	return s[:n]
}
