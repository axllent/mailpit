// Package linkcheck handles message links checking
package linkcheck

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

var linkRe = regexp.MustCompile(`(?im)\b(http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+)`)

// RunTests will run all tests on an HTML string
func RunTests(msg *storage.Message, followRedirects bool) (Response, error) {
	s := Response{}

	allLinks := extractHTMLLinks(msg)
	allLinks = strUnique(append(allLinks, extractTextLinks(msg)...))
	s.Links = getHTTPStatuses(allLinks, followRedirects)

	for _, l := range s.Links {
		if l.StatusCode >= 400 || l.StatusCode == 0 {
			s.Errors++
		}
	}

	return s, nil
}

func extractTextLinks(msg *storage.Message) []string {
	testLinkRe := regexp.MustCompile(`(?im)([^<]\b)((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;]+))`)
	// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
	// recognize potential spaces in between the URL
	// @see https://www.rfc-editor.org/rfc/rfc2396#appendix-E
	bracketLinkRe := regexp.MustCompile(`(?im)<((http|https):\/\/([\-\w@:%_\+'!.~#?,&\/\/=;][^>]+))>`)

	links := []string{}

	matches := testLinkRe.FindAllStringSubmatch(msg.Text, -1)
	for _, match := range matches {
		if len(match) > 0 {
			links = append(links, match[2])
		}
	}

	angleMatches := bracketLinkRe.FindAllStringSubmatch(msg.Text, -1)
	for _, match := range angleMatches {
		if len(match) > 0 {
			link := strings.ReplaceAll(match[1], "\n", "")
			links = append(links, link)
		}
	}

	return links
}

func extractHTMLLinks(msg *storage.Message) []string {
	links := []string{}

	reader := strings.NewReader(msg.HTML)

	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return links
	}

	aLinks := doc.Find("a[href]").Nodes
	for _, link := range aLinks {
		l, err := tools.GetHTMLAttributeVal(link, "href")
		if err == nil && linkRe.MatchString(l) {
			links = append(links, l)
		}
	}

	cssLinks := doc.Find("link[rel=\"stylesheet\"]").Nodes
	for _, link := range cssLinks {
		l, err := tools.GetHTMLAttributeVal(link, "href")
		if err == nil && linkRe.MatchString(l) {
			links = append(links, l)
		}
	}

	imgLinks := doc.Find("img[src]").Nodes
	for _, link := range imgLinks {
		l, err := tools.GetHTMLAttributeVal(link, "src")
		if err == nil && linkRe.MatchString(l) {
			links = append(links, l)
		}
	}

	return links
}

// ExtractLinks extracts all unique links from a message (without checking HTTP status)
func ExtractLinks(msg *storage.Message) []string {
	allLinks := extractHTMLLinks(msg)
	allLinks = strUnique(append(allLinks, extractTextLinks(msg)...))
	return allLinks
}

// strUnique return a slice of unique strings from a slice
func strUnique(strSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range strSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}

	return list
}
