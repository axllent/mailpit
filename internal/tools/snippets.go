package tools

import (
	"regexp"
	"strings"
)

// CreateSnippet returns a message snippet. It will use the HTML version (if it exists)
// and fall back to the text version.
func CreateSnippet(text, html string) string {
	text = strings.TrimSpace(text)
	html = strings.TrimSpace(html)
	characters := 200
	spaceRe := regexp.MustCompile(`\s+`)
	nlRe := regexp.MustCompile(`\r?\n`)

	if text == "" && html == "" {
		return ""
	}

	if html != "" {
		data := nlRe.ReplaceAllString(stripHTML(html), " ")
		data = strings.TrimSpace(spaceRe.ReplaceAllString(data, " "))

		if len(data) <= characters {
			return data
		}

		return data[0:characters] + "..."
	}

	if text != "" {
		text = spaceRe.ReplaceAllString(text, " ")
		if len(text) <= characters {
			return text
		}

		return text[0:characters] + "..."
	}

	return ""
}
