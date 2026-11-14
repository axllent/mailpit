// Package linkcheck handles message links checking
package linkcheck

import (
	"context"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

var (
	linkRe = regexp.MustCompile(`(?im)\b(http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+)`)

	// a link may start the text, otherwise it must not follow a `<` (handled by bracketLinkRe below)
	textLinkRe = regexp.MustCompile(`(?im)(\A|[^<]\b)((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+))`)

	// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
	// recognize potential spaces in between the URL
	// @see https://www.rfc-editor.org/rfc/rfc2396#appendix-E
	bracketLinkRe = regexp.MustCompile(`(?im)<((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;][^>]+))>`)

	// maxScanBytes is how much of the text and HTML body is scanned for links. Anything after
	// this is ignored, bounding the CPU and memory used on very large messages.
	// It is a variable so that tests can lower it.
	maxScanBytes = 5 * 1024 * 1024
)

const (
	// maxUniqueLinks caps how many unique links will be tested per message.
	maxUniqueLinks = 100

	// maxLinkLength is the longest link (in bytes) that will be requested. Longer links
	// are reported as unchecked, as web servers reject request lines far shorter than this.
	maxLinkLength = 8192
)

// limitScan returns at most the first maxScanBytes of s.
func limitScan(s, part string) string {
	if len(s) <= maxScanBytes {
		return s
	}

	logger.Log().Warnf("[link-check] only scanning the first %d bytes of the message %s for links", maxScanBytes, part)

	return s[:maxScanBytes]
}

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

	// Keep at most one byte over the limit, which is enough for getHTTPStatuses to
	// recognise an over-long link without holding on to (or fetching) all of it.
	// The copy releases the memory of the (possibly huge) message it was sliced from.
	if len(link) > maxLinkLength {
		link = strings.Clone(link[:maxLinkLength+1])
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

	text := limitScan(msg.Text, "text")

	// Cap the regex match count to bound work on very large bodies; the
	// 3x multiplier leaves headroom for duplicates the collector will drop.
	matchLimit := maxUniqueLinks * 3

	matches := textLinkRe.FindAllStringSubmatch(text, matchLimit)
	for _, match := range matches {
		if len(match) > 0 {
			if !c.add(match[2]) {
				return
			}
		}
	}

	angleMatches := bracketLinkRe.FindAllStringSubmatch(text, matchLimit)
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

	reader := strings.NewReader(limitScan(msg.HTML, "HTML"))

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

// displayLink returns link shortened for display if it is longer than maxLinkLength.
func displayLink(link string) string {
	if len(link) <= maxLinkLength {
		return link
	}

	// do not leave half of a multi-byte character at the cut (at most 3 bytes)
	cut := link[:maxLinkLength]
	for range utf8.UTFMax - 1 {
		if r, size := utf8.DecodeLastRuneInString(cut); r != utf8.RuneError || size != 1 {
			break
		}
		cut = cut[:len(cut)-1]
	}

	return cut + "..."
}
