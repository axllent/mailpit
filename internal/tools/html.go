package tools

import (
	"fmt"

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
