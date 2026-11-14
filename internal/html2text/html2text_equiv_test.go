package html2text

import (
	"bytes"
	"math/rand"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/net/html"
)

// the original regexp based implementation, used as a reference
var (
	refRe      = regexp.MustCompile(`\s+`)
	refSpaceRe = regexp.MustCompile(`(?mi)<\/(div|p|td|th|h[1-6]|ul|ol|li|address|article|aside|blockquote|dl|dt|footer|header|hr|main|nav|pre|table|thead|tfoot|video)><`)
	refBrRe    = regexp.MustCompile(`(?mi)<(br /|br)>`)
	refImgRe   = regexp.MustCompile(`(?mi)<(img)`)
)

func refStrip(h string, includeLinks bool) (string, error) {
	h = refSpaceRe.ReplaceAllString(h, "</$1> <")
	h = refBrRe.ReplaceAllString(h, " ")
	h = refImgRe.ReplaceAllString(h, " <$1")
	var buffer bytes.Buffer
	doc, err := html.Parse(strings.NewReader(h))
	if err != nil {
		return "", err
	}
	refExtract(doc, &buffer, includeLinks)
	return refClean(buffer.String()), nil
}

func refExtract(node *html.Node, buff *bytes.Buffer, includeLinks bool) {
	if node.Type == html.TextNode && node.Data != "" {
		buff.WriteString(node.Data)
	}
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		if _, shouldSkip := skip[c.Data]; !shouldSkip {
			refExtract(c, buff, includeLinks)
			if includeLinks && c.Data == "a" {
				for _, a := range c.Attr {
					if a.Key == "href" && strings.HasPrefix(strings.ToLower(a.Val), "http") {
						buff.WriteString(" " + a.Val + " ")
					}
				}
			}
		}
	}
}

func refClean(text string) string {
	text = strings.ReplaceAll(text, string('\uFEFF'), " ")
	text = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return []rune(" ")[0]
	}, text)
	text = refRe.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

func checkEquiv(t *testing.T, h string) {
	t.Helper()
	if got, want := spaceBlockTags(h), refSpaceRe.ReplaceAllString(h, "</$1> <"); got != want {
		t.Fatalf("spaceBlockTags(%q) = %q, want %q", h, got, want)
	}
	if got, want := replaceBr(h), refBrRe.ReplaceAllString(h, " "); got != want {
		t.Fatalf("replaceBr(%q) = %q, want %q", h, got, want)
	}
	if got, want := spaceImg(h), refImgRe.ReplaceAllString(h, " <$1"); got != want {
		t.Fatalf("spaceImg(%q) = %q, want %q", h, got, want)
	}
	if got, want := clean(h), refClean(h); got != want {
		t.Fatalf("clean(%q) = %q, want %q", h, got, want)
	}
	gotPlain, gotLinks, err := StripTrimmedAndFull(h)
	wantPlain, _ := refStrip(strings.TrimSpace(h), false)
	wantLinks, _ := refStrip(h, true)
	if err != nil || gotPlain != wantPlain || gotLinks != wantLinks {
		t.Fatalf("StripTrimmedAndFull(%q) = %q, %q, want %q, %q", h, gotPlain, gotLinks, wantPlain, wantLinks)
	}
	for _, links := range []bool{false, true} {
		got, err1 := Strip(h, links)
		want, err2 := refStrip(h, links)
		if got != want || (err1 == nil) != (err2 == nil) {
			t.Fatalf("Strip(%q, %v) = %q, want %q", h, links, got, want)
		}
	}
}

var equivPieces = []string{
	"</p><", "</P><", "</pre><", "</p>", "<p>", "</div></p><", "</li><li>", "</Td><td>", "</h1><", "</h7><", "</aſide><", "</bloc\u212Aquote><",
	"</", "><", "<", ">", "<br>", "<BR>", "<br />", "<bR />", "<br/>", "<br  />", "<br", "<img", "<IMG src=x>", "<iMg", "<im", "</p><img",
	"<a href=\"https://x.y\">link</a>", "<a href='HTTP://A'>", "<script>x</script>", "<style>.a{}</style>", "<title>t</title>",
	"text", " ", "  ", "\t", "\n", "\r\n", "\u00a0", "\uFEFF", "\u200b", "\x00", "\xff", "é", "ж", "日本", "&nbsp;", "&amp;",
	"link", "title", "</a>", "</A> ", "<!-- c ", "<textarea>",
}

func TestStripEquivalentToRegexp(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 200000; i++ {
		var b strings.Builder
		for n := r.Intn(12); n >= 0; n-- {
			b.WriteString(equivPieces[r.Intn(len(equivPieces))])
		}
		checkEquiv(t, b.String())
	}
	checkEquiv(t, htmlTestData)
}

func FuzzStripEquivalentToRegexp(f *testing.F) {
	for _, p := range equivPieces {
		f.Add(p)
	}
	f.Add(htmlTestData)
	f.Fuzz(func(t *testing.T, h string) { checkEquiv(t, h) })
}

func BenchmarkRefStripBoth(b *testing.B) {
	for b.Loop() {
		_, _ = refStrip(htmlTestData, false)
		_, _ = refStrip(htmlTestData, true)
	}
}

func BenchmarkStripBoth(b *testing.B) {
	for b.Loop() {
		_, _, _ = StripBoth(htmlTestData)
	}
}
