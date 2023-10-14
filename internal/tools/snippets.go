package tools

import (
	"regexp"
	"strings"

	"github.com/axllent/mailpit/internal/tools/html2text"
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

		return data[0:limit] + "..."
	}

	if text != "" {
		// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
		text = strings.ReplaceAll(text, string('\uFEFF'), " ")
		text = strings.TrimSpace(spaceRe.ReplaceAllString(text, " "))
		if len(text) <= limit {
			return text
		}

		return text[0:limit] + "..."
	}

	return ""
}
