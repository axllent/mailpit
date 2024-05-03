package tools

import (
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	// Invalid tag characters regex
	tagsInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9\-\ \_\.]`)

	// Regex to catch multiple spaces
	multiSpaceRe = regexp.MustCompile(`(\s+)`)

	// TagsTitleCase enforces TitleCase on all tags
	TagsTitleCase bool
)

// CleanTag returns a clean tag, trimming whitespace and replacing invalid characters
func CleanTag(s string) string {
	return strings.TrimSpace(
		multiSpaceRe.ReplaceAllString(
			tagsInvalidChars.ReplaceAllString(s, " "),
			" ",
		),
	)
}

// SetTagCasing returns the slice of tags, title-casing if set
func SetTagCasing(s []string) []string {
	if !TagsTitleCase {
		return s
	}

	titleTags := []string{}

	c := cases.Title(language.Und, cases.NoLower)

	for _, t := range s {
		titleTags = append(titleTags, c.String(t))
	}

	return titleTags
}
