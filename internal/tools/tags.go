package tools

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// Invalid tag characters regex
	tagsInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9\-\ \_]`)

	// Regex to catch multiple spaces
	multiSpaceRe = regexp.MustCompile(`(\s+)`)

	// TagsTitleCase enforces TitleCase on all tags
	TagsTitleCase bool
)

// CleanTag returns a clean tag, removing whitespace and invalid characters
func CleanTag(s string) string {
	s = strings.TrimSpace(
		multiSpaceRe.ReplaceAllString(
			tagsInvalidChars.ReplaceAllString(s, " "),
			" ",
		),
	)

	if TagsTitleCase {
		return cases.Title(language.Und, cases.NoLower).String(s)
	}

	return s
}
