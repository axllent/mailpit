// Package html2text is a simple library to convert HTML to plain text
package html2text

import (
	"bytes"
	"log"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var (
	re      = regexp.MustCompile(`\s+`)
	spaceRe = regexp.MustCompile(`(?mi)<\/(div|p|td|th|h[1-6]|ul|ol|li|address|article|aside|blockquote|dl|dt|footer|header|hr|main|nav|pre|table|thead|tfoot|video)><`)
	brRe    = regexp.MustCompile(`(?mi)<(br /|br)>`)
	imgRe   = regexp.MustCompile(`(?mi)<(img)`)
	skip    = make(map[string]bool)
)

func init() {
	skip["script"] = true
	skip["title"] = true
	skip["head"] = true
	skip["link"] = true
	skip["meta"] = true
	skip["style"] = true
	skip["noscript"] = true
}

// Strip will convert a HTML string to plain text
func Strip(h string, includeLinks bool) string {
	h = spaceRe.ReplaceAllString(h, "</$1> <")
	h = brRe.ReplaceAllString(h, " ")
	h = imgRe.ReplaceAllString(h, " <$1")
	var buffer bytes.Buffer
	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		log.Fatal(err)
	}

	extract(doc, &buffer, includeLinks)
	return clean(buffer.String())
}

func extract(node *html.Node, buff *bytes.Buffer, includeLinks bool) {
	if node.Type == html.TextNode {
		data := node.Data
		if data != "" {
			buff.WriteString(data)
		}
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if _, skip := skip[c.Data]; !skip {
			if includeLinks && c.Data == "a" {
				for _, a := range c.Attr {
					if a.Key == "href" && strings.HasPrefix(strings.ToLower(a.Val), "http") {
						buff.WriteString(" " + a.Val + " ")
					}
				}
			}
			extract(c, buff, includeLinks)
		}
	}
}

func clean(text string) string {
	// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
	text = strings.ReplaceAll(text, string('\uFEFF'), " ")

	// remove non-printable characters
	text = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return []rune(" ")[0]
	}, text)

	text = re.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
