package tools

import (
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
)

// GetHTMLAttributeVal returns the value of an HTML Attribute, else an error.
// Returns a blank value if the attribute is set but empty.
func GetHTMLAttributeVal(e *html.Node, key string) (string, error) {
	for _, a := range e.Attr {
		if a.Key == key {
			return a.Val, nil
		}
	}

	return "", fmt.Errorf("%s not found", key)
}

// StripHTML returns text from an HTML string
func stripHTML(h string) string {
	p := bluemonday.StrictPolicy()
	// // ensure joining html elements are spaced apart, eg table cells etc
	h = strings.ReplaceAll(h, "><", "> <")
	// return p.Sanitize(h)
	return html.UnescapeString(p.Sanitize(h))
}
