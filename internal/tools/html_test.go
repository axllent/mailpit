package tools

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestStripHTMLRedirects(t *testing.T) {
	const (
		refresh = "<!-- meta refresh removed -->"
		base    = "<!-- base tag removed -->"
	)

	// Each exploit is paired with a benign control of the same shape which must be left untouched.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"refresh", `<meta http-equiv="refresh" content="0;url=https://x.invalid">`, refresh},
		{"refresh mixed case", `<META HTTP-EQUIV="Refresh" content="0;url=https://x.invalid">`, refresh},
		{"refresh padded", `<meta http-equiv=" refresh " content="0;url=https://x.invalid">`, refresh},
		{"refresh entity", `<meta http-equiv="&#114;efresh" content="0;url=https://x.invalid">`, refresh},
		{"refresh slash separator", `<meta/http-equiv=refresh content="0;url=https://x.invalid">`, refresh},
		{"refresh self closing", `<meta http-equiv="refresh" content="0;url=https://x.invalid" />`, refresh},
		{"refresh attribute last", `<meta content="0;url=https://x.invalid" http-equiv="refresh">`, refresh},
		{"refresh comment close in value", `<meta content="--> <b>x</b>" http-equiv="refresh">`, refresh},
		{"base", `<base href="https://x.invalid/">`, base},
		{"base uppercase", `<BASE HREF="https://x.invalid/">`, base},
		{"base no href", `<base target="_blank">`, base},

		// controls
		{"control content-type", `<meta http-equiv="Content-Type" content="text/html; charset=utf-8">`, ""},
		{"control charset", `<meta charset="utf-8">`, ""},
		{"control viewport", `<meta name="viewport" content="width=device-width">`, ""},
		{"control refresh as content", `<meta name="description" content="refresh">`, ""},
		{"control meta in comment", `<!-- <meta http-equiv="refresh" content="0;url=x"> -->`, ""},
		{"control meta in conditional", `<!--[if mso]><meta http-equiv="refresh" content="0;url=x"><![endif]-->`, ""},
		{"control meta in style", `<style>/* <meta http-equiv="refresh"> */</style>`, ""},
		{"control text", `<p>a &lt;base href=x&gt; b</p>`, ""},
		{"control anchor", `<a href="https://example.com" target="_blank">x</a>`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StripHTMLRedirects(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			want := tt.want
			if want == "" {
				want = tt.in
			}
			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

func TestStripHTMLRedirectsPreservesDocument(t *testing.T) {
	// odd casing, whitespace, unquoted/duplicate attributes, unclosed tags, CRLF and invalid UTF-8
	doc := "<!DOCTYPE html>\r\n<HTML lang=en><head>\n<META charset=utf-8>\n<Title>T</Title>\n" +
		"<style>p{color:red}</style></head>\n<body bgcolor=\"#fff\"  class=a class=b>\n" +
		"<P>unclosed <b>bold <img src=x alt=\"a>b\"> \xff\xfe <a href='y'>z</a>\n" +
		"<table><tr><td>cell</table><!--[if !mso]><!--><span>x</span><!--<![endif]--></body></HTML>"

	got, err := StripHTMLRedirects(doc)
	if err != nil {
		t.Fatal(err)
	}
	if got != doc {
		t.Errorf("document was modified:\n got %q\nwant %q", got, doc)
	}

	// the same document with the tags is changed only at those tags
	withTags := strings.Replace(doc, "<head>", `<head><base href="//x"><meta http-equiv=refresh content="0;url=//x">`, 1)
	got, err = StripHTMLRedirects(withTags)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(doc, "<head>", "<head><!-- base tag removed --><!-- meta refresh removed -->", 1)
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// hasRedirectElement parses the HTML as a browser would, with scripting both enabled and disabled
func hasRedirectElement(t *testing.T, doc string) bool {
	t.Helper()

	for _, scripting := range []bool{true, false} {
		n, err := html.ParseWithOptions(strings.NewReader(doc), html.ParseOptionEnableScripting(scripting))
		if err != nil {
			t.Fatal(err)
		}
		if removeRedirectNodes(n) {
			return true
		}
	}

	return false
}

// These documents hide a redirect from the tokenizer, but a browser's tree builder treats it as a live element.
func TestStripHTMLRedirectsTokenizerMismatch(t *testing.T) {
	const meta = `<meta http-equiv="refresh" content="0;url=https://x.invalid">`

	bypasses := map[string]string{
		"svg style":           `<svg><style>` + meta + `</style></svg>`,
		"svg script":          `<svg><script>` + meta + `</script></svg>`,
		"svg title":           `<svg><title>` + meta + `</title></svg>`,
		"math xmp":            `<math><xmp>` + meta + `</xmp></math>`,
		"noscript":            `<head><noscript>` + meta + `</noscript></head>`,
		"noscript base":       `<noscript><base href="//x.invalid/"></noscript>`,
		"cdata hides comment": `<svg><![CDATA[ > <!-- ]]>` + meta + ` -->`,
	}

	for name, in := range bypasses {
		t.Run(name, func(t *testing.T) {
			if !hasRedirectElement(t, in) {
				t.Fatal("test document does not contain a live redirect element")
			}
			got, err := StripHTMLRedirects(in)
			if err != nil {
				t.Fatal(err)
			}
			if hasRedirectElement(t, got) {
				t.Errorf("redirect element survived: %q", got)
			}
		})
	}

	// controls: the same shapes without a redirect are preserved byte for byte
	controls := map[string]string{
		"svg style":    `<svg><style>.a{fill:red}</style></svg>`,
		"svg script":   `<svg><script>var a = 1;</script></svg>`,
		"noscript":     `<head><noscript><meta charset="utf-8"></noscript></head>`,
		"cdata":        `<svg><![CDATA[ > <!-- ]]><meta charset="utf-8"> -->`,
		"noscript img": `<noscript><img src="x"></noscript>`,
	}

	for name, in := range controls {
		t.Run("control "+name, func(t *testing.T) {
			got, err := StripHTMLRedirects(in)
			if err != nil {
				t.Fatal(err)
			}
			if got != in {
				t.Errorf("got %q, want %q", got, in)
			}
		})
	}
}
