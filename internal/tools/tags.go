package tools

import (
	"regexp"
	"strings"
)

var (
	// Invalid tag characters regex
	tagsInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9\-\ \_]`)

	// Regex to catch multiple spaces
	multiSpaceRe = regexp.MustCompile(`(\s+)`)
)

// CleanTag returns a clean tag, removing whitespace and invalid characters
func CleanTag(s string) string {
	s = strings.TrimSpace(
		multiSpaceRe.ReplaceAllString(
			tagsInvalidChars.ReplaceAllString(s, " "),
			" ",
		),
	)
	return s
}
