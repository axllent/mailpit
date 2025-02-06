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

// SetHTMLAttributeVal sets an attribute on a node.
func SetHTMLAttributeVal(n *html.Node, key, val string) {
	for i := range n.Attr {
		a := &n.Attr[i]
		if a.Key == key {
			a.Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{
		Key: key,
		Val: val,
	})
}

// WalkHTML traverses the entire HTML tree and calls fn on each node.
func WalkHTML(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}

	fn(n)

	// Each node has a pointer to its first child and next sibling. To traverse
	// all children of a node, we need to start from its first child and then
	// traverse the next sibling until nil.
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		WalkHTML(c, fn)
	}
}
