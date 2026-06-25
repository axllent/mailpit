package linkcheck

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/storage"
)

var (
	testHTML = `
	<html>
	<head>
		<link rel=stylesheet href="http://remote-host/style.css"></link>
		<script async src="https://www.googletagmanager.com/gtag/js?id=ignored"></script>
	</head>
	<body>
		<div>
			<p><a href="http://example.com">HTTP link</a></p>
			<p><a href="https://example.com">HTTPS link</a></p>
			<p><a href="HTTPS://EXAMPLE.COM">HTTPS link</a></p>
			<p><a href="http://localhost">Localhost link</a> (ignored)</p>
			<p><a href="https://localhost">Localhost link</a> (ignored)</p>
			<p><a href='https://127.0.0.1'>Single quotes link</a> (ignored)</p>
			<p><img src=https://example.com/image.jpg></p>
			<p href="http://invalid-link.com">This should be ignored</p>
			<p><a href="http://link with spaces">Link with spaces</a></p>
			<p><a href="http://example.com/?blaah=yes&amp;test=true">URL-encoded characters</a></p>
		</div>
	</body>
	</html>`

	expectedHTMLLinks = []string{
		"http://example.com",
		"https://example.com",
		"HTTPS://EXAMPLE.COM",
		"http://localhost",
		"https://localhost",
		"https://127.0.0.1",
		"http://link with spaces",
		"http://example.com/?blaah=yes&test=true",
		"http://remote-host/style.css",  // css
		"https://example.com/image.jpg", // images
	}

	testTextLinks = `This is a line with http://example.com https://example.com
		HTTPS://EXAMPLE.COM
		[http://localhost]
		www.google.com < ignored
		|||http://example.com/?some=query-string|||
		// RFC2396 appendix E states angle brackets are recommended for text/plain emails to
		// recognize potential spaces in between the URL
		<https://example.com/ link with spaces>
	`

	expectedTextLinks = []string{
		"http://example.com",
		"https://example.com",
		"HTTPS://EXAMPLE.COM",
		"http://localhost",
		"http://example.com/?some=query-string",
		"https://example.com/ link with spaces",
	}
)

func TestLinkDetection(t *testing.T) {

	t.Log("Testing HTML link detection")

	m := storage.Message{}

	m.Text = testTextLinks
	m.HTML = testHTML

	textC := &linkCollector{seen: make(map[string]bool)}
	extractTextLinks(&m, textC)

	if !reflect.DeepEqual(textC.links, expectedTextLinks) {
		t.Fatalf("Failed to detect text links correctly")
	}

	htmlC := &linkCollector{seen: make(map[string]bool)}
	extractHTMLLinks(&m, htmlC)

	if !reflect.DeepEqual(htmlC.links, expectedHTMLLinks) {
		t.Fatalf("Failed to detect HTML links correctly")
	}
}

func TestLinkLimit(t *testing.T) {
	var html strings.Builder
	html.WriteString("<html><body>")
	for i := range maxUniqueLinks + 50 {
		fmt.Fprintf(&html, `<a href="http://example.com/%d">link</a>`, i)
	}
	html.WriteString("</body></html>")

	var text strings.Builder
	for i := range 100 {
		fmt.Fprintf(&text, " http://text-example.com/%d ", i)
	}

	m := storage.Message{HTML: html.String(), Text: text.String()}

	c := &linkCollector{seen: make(map[string]bool)}
	extractHTMLLinks(&m, c)
	extractTextLinks(&m, c)

	if len(c.links) != maxUniqueLinks {
		t.Fatalf("expected %d links, got %d", maxUniqueLinks, len(c.links))
	}

	for _, l := range c.links {
		if strings.HasPrefix(l, "http://text-example.com/") {
			t.Fatalf("text extractor should not have run once HTML filled the collector, got %q", l)
		}
	}
}
