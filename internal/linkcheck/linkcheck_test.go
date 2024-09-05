package linkcheck

import (
	"reflect"
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
		"http://example.com", "https://example.com", "HTTPS://EXAMPLE.COM", "http://localhost", "https://localhost", "https://127.0.0.1", "http://link with spaces", "http://example.com/?blaah=yes&test=true",
		"http://remote-host/style.css",  // css
		"https://example.com/image.jpg", // images
	}

	testTextLinks = `This is a line with http://example.com https://example.com
		HTTPS://EXAMPLE.COM
		[http://localhost]
		www.google.com < ignored
		|||http://example.com/?some=query-string|||
	`

	expectedTextLinks = []string{
		"http://example.com", "https://example.com", "HTTPS://EXAMPLE.COM", "http://localhost", "http://example.com/?some=query-string",
	}
)

func TestLinkDetection(t *testing.T) {

	t.Log("Testing HTML link detection")

	m := storage.Message{}

	m.Text = testTextLinks
	m.HTML = testHTML

	textLinks := extractTextLinks(&m)

	if !reflect.DeepEqual(textLinks, expectedTextLinks) {
		t.Fatalf("Failed to detect text links correctly")
	}

	htmlLinks := extractHTMLLinks(&m)

	if !reflect.DeepEqual(htmlLinks, expectedHTMLLinks) {
		t.Fatalf("Failed to detect HTML links correctly")
	}
}
