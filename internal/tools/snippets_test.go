package tools

import (
	"strings"
	"testing"
)

// HTML which has no text (only images, for example) gave an empty snippet even when there was a
// text version. It now falls back to the text.
func TestCreateSnippetFallsBackToText(t *testing.T) {
	const text = "Hello from the text part"

	longText := strings.Repeat("word ", 80) // over the limit of 200
	// the first 200 bytes, as it is for any text which is too long
	truncated := strings.Repeat("word ", 40)

	tests := []struct {
		name, text, html, want string
	}{
		// the HTML has no text
		{"exploit: only an image", text, `<html><body><img src="x.png"></body></html>`, text},
		{"exploit: an image with alt text", text, `<img src="x.png" alt="Company logo">`, text},
		{"exploit: a linked image", text, `<a href="https://example.com"><img src="x.png"></a>`, text},
		{"exploit: only a non-breaking space", text, `<p>&nbsp;</p>`, text},
		{"exploit: only breaks and spaces", text, `<p> </p><br><br>`, text},
		{"exploit: only a comment", text, `<!-- nothing -->`, text},
		{"exploit: only a style and a script", text, `<html><head><style>p{color:red}</style><script>var a=1</script></head><body></body></html>`, text},
		{"exploit: an empty table", text, `<table><tr><td></td></tr></table>`, text},
		{"exploit: a long text is shortened", longText, `<img src="x.png">`, truncated + "..."},
		{"exploit: the text is tidied", "  Hello \n\n  there\t\uFEFFfriend ", `<img src="x.png">`, "Hello there friend"},

		// controls: the same shapes where nothing should change
		{"control: the HTML has text, which is used", text, `<p>Hello from html</p>`, "Hello from html"},
		{"control: the HTML has text and an image", text, `<p>Hello from html</p><img src="x.png">`, "Hello from html"},
		{"control: HTML only, with text", "", `<p>Hello from html</p>`, "Hello from html"},
		{"control: HTML only, an image", "", `<img src="x.png">`, ""},
		{"control: HTML only, whitespace", "", "  \n ", ""},
		{"control: text only", text, "", text},
		{"control: neither", "", "", ""},
		{"control: whitespace in both", " \n", " ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateSnippet(tt.text, tt.html); got != tt.want {
				t.Errorf("CreateSnippet(%q, %q) = %q, want %q", tt.text, tt.html, got, tt.want)
			}
		})
	}
}
