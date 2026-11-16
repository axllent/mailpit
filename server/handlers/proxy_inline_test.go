package handlers

import "testing"

func TestRewriteCSSPreservesInlineDataURLs(t *testing.T) {
	const css = `a { background-image: url("data:image/png;base64,AAAA"); }
@font-face { src: url('DATA:font/woff2;base64,AAAA'); }`
	got := string(rewriteCSS("msg", "https://example.test/main.css", []byte(css)))
	if got != css {
		t.Fatalf("rewritten CSS = %q, want %q", got, css)
	}
}
