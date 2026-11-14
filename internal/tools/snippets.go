package tools

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/axllent/mailpit/internal/html2text"
)

// CreateSnippet returns a message snippet. It will use the HTML version (if it has any text)
// otherwise the text version.
func CreateSnippet(text, html string) string {
	text = strings.TrimSpace(text)
	html = strings.TrimSpace(html)
	limit := 200

	if text == "" && html == "" {
		return ""
	}

	if html != "" {
		data, err := html2text.Strip(html, false)
		// HTML without any text (only images, for example) falls back to the text version
		if err == nil && strings.TrimSpace(data) != "" {
			return SnippetFromStripped(data)
		}
	}

	if text != "" {
		return snippetFromText(text, limit)
	}

	return ""
}

// snippetFromText returns the first limit bytes of the text, with each run of whitespace as one
// space, and "..." if there is more. Only as much of the text as is needed is read, as a message
// can be very large and a snippet is a few hundred bytes.
func snippetFromText(text string, limit int) string {
	var (
		n          strings.Builder
		start      = -1 // where the first character which is not a space is in n
		end        = 0  // where the last one ends
		spaceAhead bool // there was whitespace (or a \uFEFF, see below) which is a single space when something follows
	)

	// The text is the same as when it was made by taking out a \uFEFF (see
	// https://github.com/golang/go/issues/42274#issuecomment-1017258184), replacing every run of
	// \s (an ASCII space) with a single space, and trimming. Only the unicode spaces which are not \s
	// (such as a no-break space) are kept, apart from at the start and end.
	for i := 0; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		raw := text[i : i+size]
		i += size

		if r == '\uFEFF' || r == ' ' || r == '\t' || r == '\n' || r == '\f' || r == '\r' {
			spaceAhead = true
			continue
		}

		if spaceAhead {
			n.WriteByte(' ')
			spaceAhead = false
		}

		// (invalid UTF-8 is a single byte with the rune error, and is kept as it is)
		if !unicode.IsSpace(r) {
			if start < 0 {
				start = n.Len()
			}
			n.WriteString(raw)
			end = n.Len()
		} else {
			n.WriteString(raw)
		}

		// enough, as long as what is wanted is not only spaces
		if start >= 0 && end-start > limit {
			return truncate(n.String()[start:end], limit) + "..."
		}

		// Spaces which are not \s (a no-break space, for example) are kept, so they can be more than
		// the limit before anything follows them. If something does the text is longer than the
		// limit, and if not they are trimmed. Which it is is all that has to be found out.
		if start >= 0 && n.Len()-start > limit {
			if strings.IndexFunc(text[i:], func(r rune) bool { return r != '\uFEFF' && !unicode.IsSpace(r) }) >= 0 {
				return truncate(n.String()[start:], limit) + "..."
			}

			return n.String()[start:end]
		}
	}

	if start < 0 {
		return ""
	}

	return n.String()[start:end]
}

// SnippetFromStripped returns a message snippet from HTML already converted to plain text
func SnippetFromStripped(data string) string {
	limit := 200
	if len(data) <= limit {
		return data
	}

	return truncate(data, limit) + "..."
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
