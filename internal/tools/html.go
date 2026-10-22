package tools

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

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

// StripHTMLRedirects returns the HTML with `<base>` tags and `<meta http-equiv="refresh">` tags removed,
// as neither is (fully) governed by a Content-Security-Policy.
//
// The tokenizer used by stripHTMLRedirectTags knows nothing about the tree-builder rules a browser applies
// (foreign content such as `<svg><style>`, `<noscript>` when scripting is disabled, `<![CDATA[`), so a tag
// it considers to be text may be a live element to a browser. The result is therefore verified by parsing
// it, with scripting both enabled and disabled. In the (unusual) case that a redirect element survives, the
// document is rewritten from the parsed tree with those elements replaced by comments. This does normalise
// the HTML, but only for documents which would otherwise have bypassed the filter.
//
// An error is returned if the HTML cannot be processed, so the caller can fail closed.
func StripHTMLRedirects(htmlStr string) (string, error) {
	out, err := stripHTMLRedirectTags(htmlStr)
	if err != nil {
		return "", err
	}

	for _, scripting := range []bool{true, false} {
		doc, err := html.ParseWithOptions(strings.NewReader(out), html.ParseOptionEnableScripting(scripting))
		if err != nil {
			return "", err
		}

		if !removeRedirectNodes(doc) {
			continue
		}

		// A redirect element survived the tokenizer pass. Scripting is disabled for this parse, as it is the
		// superset, so the contents of `<noscript>` become elements which can be removed, and rendered.
		doc, err = html.ParseWithOptions(strings.NewReader(out), html.ParseOptionEnableScripting(false))
		if err != nil {
			return "", err
		}
		removeRedirectNodes(doc)

		b := bytes.Buffer{}
		if err := html.Render(&b, doc); err != nil {
			return "", err
		}

		// the rendered output is verified again, as it is what will be parsed by the browser
		final := b.String()
		for _, sc := range []bool{true, false} {
			check, err := html.ParseWithOptions(strings.NewReader(final), html.ParseOptionEnableScripting(sc))
			if err != nil {
				return "", err
			}
			if removeRedirectNodes(check) {
				return "", errors.New("unable to remove HTML redirect elements")
			}
		}

		return final, nil
	}

	return out, nil
}

// removeRedirectNodes replaces any `<base>` and `<meta http-equiv="refresh">` elements in the tree with
// comments, returning whether any were found. The tree is walked iteratively as nesting depth is attacker controlled.
func removeRedirectNodes(root *html.Node) bool {
	found := false
	stack := []*html.Node{root}

	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if n.Type == html.ElementNode && n.Namespace == "" {
			marker := ""
			switch n.Data {
			case "base":
				marker = " base tag removed "
			case "meta":
				for _, a := range n.Attr {
					if a.Key == "http-equiv" && strings.EqualFold(strings.TrimSpace(a.Val), "refresh") {
						marker = " meta refresh removed "
					}
				}
			}
			if marker != "" && n.Parent != nil {
				found = true
				n.Parent.InsertBefore(&html.Node{Type: html.CommentNode, Data: marker}, n)
				n.Parent.RemoveChild(n)

				continue
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			stack = append(stack, c)
		}
	}

	return found
}

// stripHTMLRedirectTags returns the HTML with every `<base>` tag and every
// `<meta http-equiv="refresh">` tag replaced with a fixed HTML comment.
//
// Neither is governed by a Content-Security-Policy (`<base>` is only partially covered by base-uri),
// so they must be removed from untrusted HTML which is served directly to a browser.
//
// The HTML is tokenized rather than parsed, and the raw bytes of every other token are written
// out unmodified, so the rest of the document is preserved byte for byte. The original tag is
// deliberately never included in the replacement comment, as an attribute value containing `-->`
// would otherwise close the comment early.
//
// An error is returned if the HTML cannot be tokenized, so the caller can fail closed.
func stripHTMLRedirectTags(htmlStr string) (string, error) {
	z := html.NewTokenizer(strings.NewReader(htmlStr))
	out := bytes.Buffer{}
	out.Grow(len(htmlStr))

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if err := z.Err(); err != io.EOF {
				return "", err
			}

			return out.String(), nil
		}

		// Raw() must be copied before TagName() or TagAttr() are called, as they modify the underlying buffer.
		mark := out.Len()
		out.Write(z.Raw())

		if tt != html.StartTagToken && tt != html.SelfClosingTagToken {
			continue
		}

		name, hasAttr := z.TagName()

		switch string(name) {
		case "base":
			out.Truncate(mark)
			out.WriteString("<!-- base tag removed -->")
		case "meta":
			if hasAttr && isMetaRefresh(z) {
				out.Truncate(mark)
				out.WriteString("<!-- meta refresh removed -->")
			}
		}
	}
}

// isMetaRefresh returns whether the tokenizer's current tag has http-equiv="refresh"
func isMetaRefresh(z *html.Tokenizer) bool {
	for {
		key, val, more := z.TagAttr()
		if string(key) == "http-equiv" && strings.EqualFold(strings.TrimSpace(string(val)), "refresh") {
			return true
		}
		if !more {
			return false
		}
	}
}
