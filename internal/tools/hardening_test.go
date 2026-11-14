package tools

import (
	"io/fs"
	"math/rand"
	"net"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"
)

// An address which is not 4 or 16 bytes long was reported as not internal, as every check says no to it.
func TestIsInternalIPInvalid(t *testing.T) {
	for _, ip := range []net.IP{nil, {}, {1}, {8, 8, 8}, {8, 8, 8, 8, 8}, make(net.IP, 15), make(net.IP, 17), net.ParseIP("not an address"), net.ParseIP("")} {
		if !IsInternalIP(ip) {
			t.Errorf("IsInternalIP(%v, %d bytes) = false", ip, len(ip))
		}
	}

	// controls: valid external addresses, in both forms
	for _, s := range []string{"8.8.8.8", "1.1.1.1", "93.184.216.34", "2606:4700:4700::1111", "2001:4860:4860::8888"} {
		ip := net.ParseIP(s)
		if IsInternalIP(ip) {
			t.Errorf("IsInternalIP(%s) = true", s)
		}
		if v4 := ip.To4(); v4 != nil && IsInternalIP(v4) {
			t.Errorf("IsInternalIP(%s, 4 bytes) = true", s)
		}
	}

	// and internal ones still are
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fd00:ec2::254"} {
		if !IsInternalIP(net.ParseIP(s)) {
			t.Errorf("IsInternalIP(%s) = false", s)
		}
	}
}

func TestUnixSocketAddress(t *testing.T) {
	tests := []struct {
		in   string
		path string
		mode fs.FileMode
		ok   bool
	}{
		{"unix:/tmp/mailpit.sock:0666", "/tmp/mailpit.sock", 0o666, true},
		{"unix:/tmp/mailpit.sock:660", "/tmp/mailpit.sock", 0o660, true},
		{"unix:relative.sock:0600", "relative.sock", 0o600, true},
		{"unix:/tmp//a/../mailpit.sock:0644", "/tmp/mailpit.sock", 0o644, true},
		{"unix:/with:colon/s.sock:0600", "/with:colon/s.sock", 0o600, true},

		{"unix:/tmp/mailpit.sock", "", 0, false},
		{"unix:/tmp/mailpit.sock:06666", "", 0, false},
		{"unix:/tmp/mailpit.sock:66", "", 0, false},
		{"unix:/tmp/mailpit.sock:0999", "", 0, false},
		{"unix:/tmp/mailpit.sock:abcd", "", 0, false},
		{"/tmp/mailpit.sock:0666", "", 0, false},
		{"0.0.0.0:8025", "", 0, false},
		{"", "", 0, false},
	}

	for _, tt := range tests {
		path, mode, ok := UnixSocket(tt.in)
		if path != tt.path || mode != tt.mode || ok != tt.ok {
			t.Errorf("UnixSocket(%q) = %q, %v, %v, want %q, %v, %v", tt.in, path, mode, ok, tt.path, tt.mode, tt.ok)
		}
	}
}

// oldTextSnippet is how the snippet of a text was made: the whole text was changed, then cut.
var oldSpaceRe = regexp.MustCompile(`\s+`)

func oldTextSnippet(text string) string {
	limit := 200
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	text = strings.ReplaceAll(text, string('\uFEFF'), " ")
	text = strings.TrimSpace(oldSpaceRe.ReplaceAllString(text, " "))
	if len(text) <= limit {
		return text
	}

	return truncate(text, limit) + "..."
}

func TestTextSnippetMatchesWholeTextMethod(t *testing.T) {
	fixed := []string{
		"", " ", "\n\t", "\uFEFF", " \uFEFF ", "a", " a ", "a  b", "a\n\n\tb", "a b", "a   b", " a ", "a\uFEFF b", "a\uFEFF b",
		" \uFEFFa", "a  b", "日本語 テスト", "a\xffb", "\xff", "a\xe3\x81", "a \xff ", "a\x00b", "a\vb", "a\fb",
		strings.Repeat("a", 200), strings.Repeat("a", 201), strings.Repeat("a", 199) + " b", strings.Repeat("a", 200) + " ", strings.Repeat("a", 200) + "  b",
		strings.Repeat("a ", 100), strings.Repeat("a ", 101), strings.Repeat("あ", 66), strings.Repeat("あ", 67), strings.Repeat("a", 199) + "あ",
		strings.Repeat("a", 200) + " ", strings.Repeat("a", 200) + " b", strings.Repeat("a", 198) + " b",
	}
	for _, text := range fixed {
		if got, want := CreateSnippet(text, ""), oldTextSnippet(text); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}

	rng := rand.New(rand.NewSource(7))
	pieces := []string{" ", "  ", "\n", "\t", "\r\n", "\f", "\uFEFF", " ", " ", "a", "bc", "word", "é", "日", "\xff", "\xe3", "\x00", strings.Repeat("x", 60), strings.Repeat("あ", 30)}
	for range 100000 {
		var b strings.Builder
		for range rng.Intn(24) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		text := b.String()
		if got, want := CreateSnippet(text, ""), oldTextSnippet(text); got != want {
			t.Fatalf("%q: got %q, want %q", text, got, want)
		}
	}
}

func FuzzTextSnippet(f *testing.F) {
	for _, s := range []string{"a  b", " \uFEFF a", strings.Repeat("あ", 70), "a\xffb", "a  b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if got, want := CreateSnippet(text, ""), oldTextSnippet(text); got != want {
			t.Fatalf("%q: got %q, want %q", text, got, want)
		}
	})
}

// A message can be 50 MiB, and a snippet is 200 bytes. The text used to be changed all the way
// through, which took about 5 seconds.
func TestTextSnippetOnlyReadsWhatItNeeds(t *testing.T) {
	for _, shape := range []struct{ name, text string }{
		{"words", strings.Repeat("hello world ", (50<<20)/12)},
		{"one long word", strings.Repeat("a", 50<<20)},
		{"mostly spaces", "start" + strings.Repeat(" ", 50<<20) + "end"},
		{"unicode spaces", "start " + strings.Repeat(" ", 20<<20) + "end"},
	} {
		start := time.Now()
		got := CreateSnippet(shape.text, "")
		if d := time.Since(start); d > 500*time.Millisecond {
			t.Errorf("%s: a %d MiB text took %v", shape.name, len(shape.text)>>20, d)
		}
		if got == "" || len(got) > 203 {
			t.Errorf("%s: snippet is %d bytes", shape.name, len(got))
		}
	}

	// and does not keep it all: spaces which are not collapsed would otherwise be copied
	// until something follows them
	nbsp := "start\u00a0" + strings.Repeat("\u00a0", 20<<20) + "end"
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_ = CreateSnippet(nbsp, "")
	runtime.ReadMemStats(&after)
	if used := after.TotalAlloc - before.TotalAlloc; used > 4<<20 {
		t.Errorf("a snippet of a %d MiB text of no-break spaces allocated %d MiB", len(nbsp)>>20, used>>20)
	}

	// the cases which have to read all of it are the same as ever
	if got := CreateSnippet("start"+strings.Repeat(" ", 1<<20)+"end", ""); got != "start end" {
		t.Errorf("got %q", got)
	}
}

// A document with neither "<base" nor "<meta" has no such element, so it is not parsed. The
// property is checked on documents made of the pieces which could make one: for every one which is
// passed over, parsing it (with scripting on and off) must find nothing to remove.
func TestRedirectPrefilterIsSound(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	pieces := []string{
		"<", ">", "/", " ", "\n", "=", `"`, "'", "-->", "<!--", "<![CDATA[", "]]>", "<svg>", "</svg>", "<math>", "<noscript>", "</noscript>", "<template>", "<table>", "<select>", "<head>",
		"<base", "<BASE", "<meta", "<Meta", "ba", "se", "me", "ta", "<ba", "<me", "<b", "base", "meta", "&lt;base", "&#60;base", "<\x00base", "<ba\x00se", "<m\x00eta", "< base", "<\nbase",
		"href=x", "http-equiv=refresh", "content=0", "<a>", "</a>", "<p>", "text", "<style>", "</style>", "<script>", "</script>", "<title>", "</title>", "<plaintext>", "<textarea>", "<frameset>",
		"<image>", "<isindex>", "<br>", "</br>", "<ba<se>", "<<base", "<!base", "<?base", "</base>", "</meta>",
	}

	passed := 0
	for range 150000 {
		var b strings.Builder
		for range rng.Intn(10) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		h := b.String()

		if mayContainRedirectElement(h) {
			continue
		}
		passed++

		for _, scripting := range []bool{true, false} {
			doc, err := html.ParseWithOptions(strings.NewReader(h), html.ParseOptionEnableScripting(scripting))
			if err != nil {
				continue
			}
			if removeRedirectNodes(doc) {
				t.Fatalf("%q was passed over, but has a redirect element (scripting %v)", h, scripting)
			}
		}

		// and processing it would have changed nothing but dropping a tag which is cut off at the
		// end of the document, which a browser drops too
		if out, err := stripHTMLRedirectTags(h); err != nil || !strings.HasPrefix(h, out) {
			t.Fatalf("%q is changed by the tokeniser to %q (%v)", h, out, err)
		}
	}

	if passed < 1000 {
		t.Fatalf("only %d documents were passed over, the test is not testing much", passed)
	}
}

func FuzzRedirectPrefilter(f *testing.F) {
	for _, s := range []string{"<p>text</p>", "<meta http-equiv=refresh content=0>", "<svg><style><meta http-equiv=refresh></style></svg>", "<ba\x00se>", "<noscript><base href=x></noscript>"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, h string) {
		if mayContainRedirectElement(h) {
			return
		}
		for _, scripting := range []bool{true, false} {
			doc, err := html.ParseWithOptions(strings.NewReader(h), html.ParseOptionEnableScripting(scripting))
			if err == nil && removeRedirectNodes(doc) {
				t.Fatalf("%q has a redirect element but was passed over", h)
			}
		}
	})
}

func TestMayContainRedirectElement(t *testing.T) {
	for in, want := range map[string]bool{
		"": false, "<p>text</p>": false, "<b>ase</b>": false, "base": false, "meta": false, "&lt;base&gt;": false, "<ba": false, "<bas": false,
		"<base": true, "<BASE href=x>": true, "<Base>": true, "<meta charset=utf-8>": true, "<META http-equiv=refresh>": true, "text <meta": true, "<<meta": true,
		"<p><meta": true, "<!-- <base href=x> -->": true,
	} {
		if got := mayContainRedirectElement(in); got != want {
			t.Errorf("mayContainRedirectElement(%q) = %v, want %v", in, got, want)
		}
	}
}

// A document which has an element to remove is still removed, whatever the prefilter does.
func TestStripHTMLRedirectsStillRemovesWhatItShould(t *testing.T) {
	for _, in := range []string{
		`<html><head><META HTTP-EQUIV="Refresh" content="0;url=https://x.test"></head><body>b</body></html>`,
		`<BASE href="https://evil.test/">text`,
		`<noscript><base href=x></noscript>`,
		`<svg><style><meta http-equiv=refresh content=0></style></svg>`,
	} {
		out, err := StripHTMLRedirects(in)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToLower(out), "<base") || strings.Contains(strings.ToLower(out), "http-equiv") {
			t.Errorf("%q: %q", in, out)
		}
	}

	// control: a meta which is not a refresh stays, and a document with neither is as it was
	if out, _ := StripHTMLRedirects(`<meta charset="utf-8"><p>x</p>`); out != `<meta charset="utf-8"><p>x</p>` {
		t.Errorf("a charset meta was changed: %q", out)
	}
	if out, _ := StripHTMLRedirects("<p>no redirect here</p>\n"); out != "<p>no redirect here</p>\n" {
		t.Errorf("a document with nothing to remove was changed: %q", out)
	}
}

// A large document with no such element used to be tokenised and parsed twice (5.7 seconds and
// 1.3 GiB for 50 MiB), where the answer is that there is nothing to do.
func TestStripHTMLRedirectsLargeDocumentWithoutRedirect(t *testing.T) {
	chunk := `<div class="a"><p>text <a href="https://example.com/x">link</a></p></div>`
	big := strings.Repeat(chunk, (50<<20)/len(chunk))

	start := time.Now()
	out, err := StripHTMLRedirects(big)
	if err != nil || out != big {
		t.Fatalf("changed or failed: %v", err)
	}
	if d := time.Since(start); d > time.Second {
		t.Errorf("a 50 MiB document took %v", d)
	}
}

// Only a few documents which may have a redirect are processed at once. A document which has no
// such element does not wait.
func TestStripHTMLRedirectsIsLimited(t *testing.T) {
	orig := redirectSlots
	redirectSlots = make(chan struct{}, 2)
	t.Cleanup(func() { redirectSlots = orig })

	withMeta := `<meta charset="utf-8"><p>x</p>`

	// control: with a free slot it is done at once
	done := make(chan string, 1)
	go func() { out, _ := StripHTMLRedirects(withMeta); done <- out }()
	select {
	case out := <-done:
		if out != withMeta {
			t.Errorf("changed: %q", out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("did not finish with free slots")
	}

	// exploit: with every slot in use another waits
	redirectSlots <- struct{}{}
	redirectSlots <- struct{}{}

	go func() { out, _ := StripHTMLRedirects(withMeta); done <- out }()
	select {
	case <-done:
		t.Fatal("a document was processed while all the slots were in use")
	case <-time.After(300 * time.Millisecond):
	}

	// one with nothing which could be removed is not held up
	if out, err := StripHTMLRedirects("<p>nothing</p>"); err != nil || out != "<p>nothing</p>" {
		t.Fatalf("%q, %v", out, err)
	}

	<-redirectSlots
	select {
	case out := <-done:
		if out != withMeta {
			t.Errorf("changed: %q", out)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting document was not processed once a slot was free")
	}
	<-redirectSlots
}
