package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/html2text"
	"github.com/axllent/mailpit/internal/mimeparser"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/jhillyerd/enmime/v2"
)

// refSearchTextAndSnippet is the previous implementation: a full parse (including
// enmime's HTML to text down-conversion) and separate HTML strips for the search
// text and snippet
func refSearchTextAndSnippet(t testing.TB, raw []byte) (string, string) {
	t.Helper()
	env, err := mimeparser.New().ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	h, _ := html2text.Strip(env.HTML, true)
	body := env.Text
	if h != "" {
		body = h
	}
	searchText := cleanString(env.GetHeader("From") + " " + env.GetHeader("Subject") + " " +
		env.GetHeader("To") + " " + env.GetHeader("Cc") + " " + env.GetHeader("Bcc") + " " +
		env.GetHeader("Reply-To") + " " + env.GetHeader("Return-Path") + " " + body + " " + attachmentNames(env.Attachments))

	return searchText, tools.CreateSnippet(env.Text, env.HTML)
}

func checkSearchTextAndSnippet(t testing.TB, raw []byte) {
	t.Helper()
	env, err := storeParser.ReadEnvelope(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}

	gotSearch, gotSnippet := searchTextAndSnippet(env)
	wantSearch, wantSnippet := refSearchTextAndSnippet(t, raw)
	if gotSearch != wantSearch {
		t.Fatalf("search text mismatch for %q:\n got: %q\nwant: %q", raw, gotSearch, wantSearch)
	}
	if gotSnippet != wantSnippet {
		t.Fatalf("snippet mismatch for %q:\n got: %q\nwant: %q", raw, gotSnippet, wantSnippet)
	}
}

func htmlOnlyMessage(html string) []byte {
	return []byte("From: a@example.com\r\nTo: b@example.com\r\nSubject: Test\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n\r\n" + html)
}

var equivHTMLBodies = []string{
	"<p>Hello <b>world</b></p>",
	"<p>Hello</p>\r\n",
	"\r\n  <div>leading space</div>",
	"link</a>\r\n",
	"<b>title</b>\n",
	"<img src=\"logo.png\" alt=\"Logo\">",
	"<a href=\"https://example.com\"><img src=\"x.png\"></a>",
	"<style>p{}</style>\n",
	"   \r\n",
	"<p>broken tail</ \n",
	"<p>broken attr <a href=\"https://x\n",
	"<table><tr><td>a</td><td>b</td></tr></table><br/><br />x<BR>y",
	"<html><head><title>T</title></head><body><p>Body</p></body></html>\n\n",
}

func TestSearchTextAndSnippetEquivalent(t *testing.T) {
	files, err := filepath.Glob("testdata/*.eml")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		checkSearchTextAndSnippet(t, raw)
	}

	for _, h := range equivHTMLBodies {
		checkSearchTextAndSnippet(t, htmlOnlyMessage(h))
	}

	// multipart with both a text and HTML part
	checkSearchTextAndSnippet(t, []byte("From: a@example.com\r\nSubject: Alt\r\nMIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=b\r\n\r\n"+
		"--b\r\nContent-Type: text/plain\r\n\r\nplain text\r\n"+
		"--b\r\nContent-Type: text/html\r\n\r\n<img src=x>\r\n--b--\r\n"))
}

func FuzzSearchTextAndSnippetEquivalent(f *testing.F) {
	for _, h := range equivHTMLBodies {
		f.Add(h)
	}
	f.Fuzz(func(t *testing.T, h string) {
		checkSearchTextAndSnippet(t, htmlOnlyMessage(h))
	})
}

func attachmentNames(parts []*enmime.Part) string {
	s := ""
	for _, a := range parts {
		s += a.FileName + " "
	}

	return s
}

// refCleanStringRe is the regular expression previously used by cleanString
var refCleanStringRe = regexp.MustCompile(`(\r?\n|\t|>|<|"|\,|;|\(|\))`)

func refCleanString(str string) string {
	str = strings.ReplaceAll(str, string('\uFEFF'), " ")
	str = refCleanStringRe.ReplaceAllString(str, " ")

	return foldCase(strings.Join(strings.Fields(strings.TrimSpace(str)), " "))
}

func FuzzCleanStringEquivalent(f *testing.F) {
	for _, s := range []string{"a\r\nb", "a\rb", "\r\r\n\n", "x<y>z", `"q",;()`, "\uFEFF\t ß İ", "\xff\xfe"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got, want := cleanString(s), refCleanString(s); got != want {
			t.Fatalf("cleanString(%q) = %q, want %q", s, got, want)
		}
	})
}
