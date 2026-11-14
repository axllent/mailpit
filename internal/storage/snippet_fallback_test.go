package storage

import (
	"bytes"
	"strings"
	"testing"
)

func alternativeMessage(text, html string) []byte {
	return []byte("From: a@example.com\r\nTo: b@example.com\r\nSubject: Alt\r\nMIME-Version: 1.0\r\n" +
		"Content-Type: multipart/alternative; boundary=b\r\n\r\n" +
		"--b\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + text + "\r\n" +
		"--b\r\nContent-Type: text/html; charset=utf-8\r\n\r\n" + html + "\r\n--b--\r\n")
}

func plainMessage(text string) []byte {
	return []byte("From: a@example.com\r\nTo: b@example.com\r\nSubject: Plain\r\n\r\n" + text + "\r\n")
}

// A message whose HTML has no text (only images, for example) used to have an empty snippet even
// when it had a text part. The message is read as it is when it is stored.
func TestSnippetFallsBackToTextPart(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("word ", 80))

	tests := []struct {
		name string
		raw  []byte
		want string
	}{
		{"exploit: an image", alternativeMessage("Hello from the text part", `<img src="x.png">`), "Hello from the text part"},
		{"exploit: a linked image", alternativeMessage("Hello from the text part", `<a href="https://example.com"><img src="x.png"></a>`), "Hello from the text part"},
		{"exploit: a style only", alternativeMessage("Hello from the text part", `<style>p{}</style>`), "Hello from the text part"},
		{"exploit: a non-breaking space", alternativeMessage("Hello from the text part", `<p>&nbsp;</p>`), "Hello from the text part"},
		{"exploit: a long text part is shortened", alternativeMessage(long, `<img src="x.png">`), strings.Repeat("word ", 40) + "..."},

		{"control: the HTML has text", alternativeMessage("Hello from the text part", `<p>Hello from html</p>`), "Hello from html"},
		{"control: text only", plainMessage("Hello from the text part"), "Hello from the text part"},
		// there is no text part, so there is nothing to fall back to
		{"control: HTML only, an image", htmlOnlyMessage(`<img src="x.png">`), ""},
		{"control: HTML only, a linked image", htmlOnlyMessage(`<a href="https://example.com"><img src="x.png"></a>`), ""},
		{"control: HTML only, text", htmlOnlyMessage(`<p>Hello from html</p>`), "Hello from html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, err := storeParser.ReadEnvelope(bytes.NewReader(tt.raw))
			if err != nil {
				t.Fatal(err)
			}

			if _, got := searchTextAndSnippet(env); got != tt.want {
				t.Errorf("snippet %q, want %q", got, tt.want)
			}

			// the same snippet is made by the function used elsewhere
			checkSearchTextAndSnippet(t, tt.raw)
		})
	}
}
