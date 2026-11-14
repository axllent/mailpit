package html2text

import (
	"testing"
)

// Text which is only the name of a skipped element (eg: the link text "link") is text, and is kept.
// It used to be lost, as the name of every node was compared with the skipped elements, and the name
// of a text node is its text.
func TestTextWhichIsASkippedElementName(t *testing.T) {
	tests := []struct {
		name, html, plain, links string
	}{
		// the text is the same as the name of a skipped element
		{"exploit: title", `<b>title</b>`, "title", "title"},
		{"exploit: link", `<b>link</b>`, "link", "link"},
		{"exploit: script", `<p>script</p>`, "script", "script"},
		{"exploit: style", `<td>style</td>`, "style", "style"},
		{"exploit: head", `<div>head</div>`, "head", "head"},
		{"exploit: meta", `<span>meta</span>`, "meta", "meta"},
		{"exploit: noscript", `<i>noscript</i>`, "noscript", "noscript"},
		{"exploit: link text", `<a href="https://example.com">link</a>`, "link", "link https://example.com"},
		{"exploit: link text in a sentence", `<p>Click <a href="https://example.com">link</a> now</p>`, "Click link now", "Click link https://example.com now"},
		{"exploit: surrounded by space", `<p> title </p>`, "title", "title"},
		{"exploit: bare text", `link`, "link", "link"},
		{"exploit: next to other text", `<p>title</p><p>body</p>`, "title body", "title body"},

		// controls: text which is not the name, in the same shapes
		{"control: other text", `<b>word</b>`, "word", "word"},
		{"control: nearly the name", `<b>titl</b>`, "titl", "titl"},
		{"control: more than the name", `<b>the title</b>`, "the title", "the title"},
		{"control: a capital letter", `<b>Title</b>`, "Title", "Title"},
		{"control: a link", `<a href="https://example.com">word</a>`, "word", "word https://example.com"},

		// controls: the elements themselves are still skipped, with what is in them
		{"control: a title element", `<title>title</title><p>body</p>`, "body", "body"},
		{"control: a style element", `<style>link</style><p>body</p>`, "body", "body"},
		{"control: a script element", `<script>style</script><p>body</p>`, "body", "body"},
		{"control: a noscript element", `<noscript>meta</noscript><p>body</p>`, "body", "body"},
		{"control: a head", `<html><head><title>t</title><meta charset="utf-8"><link rel="x"></head><body>body</body></html>`, "body", "body"},
		{"control: a skipped element with its own name in it", `<style>style</style>body`, "body", "body"},
		{"control: a link element, not a link text", `<link rel="stylesheet" href="https://example.com/a.css">body`, "body", "body"},
		{"control: a link to a skipped link is not added", `<a href="https://example.com"><script>x</script></a>`, "", "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain, err := Strip(tt.html, false)
			if err != nil || plain != tt.plain {
				t.Errorf("Strip(%q, false) = %q, %v, want %q", tt.html, plain, err, tt.plain)
			}

			links, err := Strip(tt.html, true)
			if err != nil || links != tt.links {
				t.Errorf("Strip(%q, true) = %q, %v, want %q", tt.html, links, err, tt.links)
			}

			trimmed, full, err := StripTrimmedAndFull(tt.html)
			if err != nil || trimmed != tt.plain || full != tt.links {
				t.Errorf("StripTrimmedAndFull(%q) = %q, %q, %v, want %q, %q", tt.html, trimmed, full, err, tt.plain, tt.links)
			}
		})
	}
}
