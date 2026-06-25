// Package linkcheck handles message links checking
package linkcheck

import (
	"context"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

var linkRe = regexp.MustCompile(`(?im)\b(http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+)`)

// maxUniqueLinks caps how many unique links will be tested per message.
const maxUniqueLinks = 100

// RunTests will run all tests on an HTML string
func RunTests(ctx context.Context, msg *storage.Message, followRedirects bool) (Response, error) {
	s := Response{}

	c := &linkCollector{seen: make(map[string]bool)}
	extractHTMLLinks(msg, c)
	extractTextLinks(msg, c)
	s.Links = getHTTPStatuses(ctx, c.links, followRedirects)

	for _, l := range s.Links {
		if l.StatusCode >= 400 || l.StatusCode == 0 {
			s.Errors++
		}
	}

	return s, nil
}

// linkCollector accumulates unique links up to maxUniqueLinks.
type linkCollector struct {
	seen  map[string]bool
	links []string
}

// full reports whether the collector has reached maxUniqueLinks.
func (c *linkCollector) full() bool {
	return len(c.links) >= maxUniqueLinks
}

// add appends link if new and within capacity, returning false when the
// collector is full and the caller should stop producing more links.
func (c *linkCollector) add(link string) bool {
	if c.full() {
		return false
	}
	if !c.seen[link] {
		c.seen[link] = true
		c.links = append(c.links, link)
	}
	return !c.full()
}

func extractTextLinks(msg *storage.Message, c *linkCollector) {
	if c.full() {
		return
	}

	testLinkRe := regexp.MustCompile(`(?im)([^<]\b)((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+))`)
	// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
	// recognize potential spaces in between the URL
	// @see https://www.rfc-editor.org/rfc/rfc2396#appendix-E
	bracketLinkRe := regexp.MustCompile(`(?im)<((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;][^>]+))>`)

	// Cap the regex match count to bound work on very large bodies; the
	// 3x multiplier leaves headroom for duplicates the collector will drop.
	matchLimit := maxUniqueLinks * 3

	matches := testLinkRe.FindAllStringSubmatch(msg.Text, matchLimit)
	for _, match := range matches {
		if len(match) > 0 {
			if !c.add(match[2]) {
				return
			}
		}
	}

	angleMatches := bracketLinkRe.FindAllStringSubmatch(msg.Text, matchLimit)
	for _, match := range angleMatches {
		if len(match) > 0 {
			link := strings.ReplaceAll(match[1], "\n", "")
			if !c.add(link) {
				return
			}
		}
	}
}

func extractHTMLLinks(msg *storage.Message, c *linkCollector) {
	if c.full() {
		return
	}

	reader := strings.NewReader(msg.HTML)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return
	}

	for _, sel := range []struct{ selector, attr string }{
		{"a[href]", "href"},
		{`link[rel="stylesheet"]`, "href"},
		{"img[src]", "src"},
	} {
		for _, node := range doc.Find(sel.selector).Nodes {
			l, err := tools.GetHTMLAttributeVal(node, sel.attr)
			if err != nil || !linkRe.MatchString(l) {
				continue
			}
			if !c.add(l) {
				return
			}
		}
	}
}
