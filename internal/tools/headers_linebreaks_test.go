package tools

import (
	"bytes"
	"math/rand"
	"net/mail"
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestNormaliseHeaderLineBreaks(t *testing.T) {
	tests := []struct {
		name, in, want string
		changed        bool
	}{
		// controls: nothing to change
		{"control: CRLF", "From: a\r\nTo: b\r\n\r\nbody\r\n", "From: a\r\nTo: b\r\n\r\nbody\r\n", false},
		{"control: folded lines", "Bcc: a,\r\n b,\r\n\tc\r\nTo: x\r\n\r\nbody", "Bcc: a,\r\n b,\r\n\tc\r\nTo: x\r\n\r\nbody", false},
		{"control: a line of only whitespace is not the end of the headers", "A: x\r\n \r\nB: y\r\n\r\nbody", "A: x\r\n \r\nB: y\r\n\r\nbody", false},
		{"control: no body", "From: a\r\nTo: b\r\n", "From: a\r\nTo: b\r\n", false},
		{"control: no line break at all", "From: a", "From: a", false},
		{"control: empty", "", "", false},
		{"control: an empty line first", "\r\nbody\n", "\r\nbody\n", false},
		{"control: a CR on its own is text, not a line break", "Subject: one\rBcc: x\r\n\r\nbody", "Subject: one\rBcc: x\r\n\r\nbody", false},

		// the line breaks which are not CRLF
		{"exploit: LF", "From: a\nTo: b\n\nbody\n", "From: a\r\nTo: b\r\n\r\nbody\n", true},
		{"exploit: CR CR LF", "From: a\r\r\nTo: b\r\r\n\r\r\nbody", "From: a\r\nTo: b\r\n\r\nbody", true},
		{"exploit: more than two CR", "From: a\r\r\r\nTo: b\r\n\r\nbody", "From: a\r\nTo: b\r\n\r\nbody", true},
		{"control: CR only is all one header", "From: a\rSubject: b\r\rbody\r", "From: a\rSubject: b\r\rbody\r", false},
		{"exploit: mixed", "From: a\r\nSubject: b\nTo: c\rX: d\r\r\n\r\nbody", "From: a\r\nSubject: b\r\nTo: c\rX: d\r\n\r\nbody", true},
		{"exploit: folded line with LF", "Bcc: a,\n b\nTo: x\n\nbody", "Bcc: a,\r\n b\r\nTo: x\r\n\r\nbody", true},
		{"control: a bare CR is not the empty line", "From: a\r\nTo: b\r\n\rbody", "From: a\r\nTo: b\r\n\rbody", false},
		{"exploit: LF first", "\nbody\r", "\r\nbody\r", true},
		{"control: only a CR", "\r", "\r", false},
		{"exploit: only a LF", "\n", "\r\n", true},
		{"exploit: the last header has a bare LF and no body", "From: a\r\nTo: b\n", "From: a\r\nTo: b\r\n", true},

		// the body is exactly as it was, whatever line breaks it has
		{"body: bare CR, bare LF and CR CR LF", "From: a\r\n\r\nx\ry\nz\r\r\nw\r", "From: a\r\n\r\nx\ry\nz\r\r\nw\r", false},
		{"body: after headers with LF", "From: a\n\nx\ry\nz\r\r\nw", "From: a\r\n\r\nx\ry\nz\r\r\nw", true},
		{"body: headers in the body are not changed", "From: a\n\nFrom: b\rTo: c\n", "From: a\r\n\r\nFrom: b\rTo: c\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(tt.in)
			got, changed := NormaliseHeaderLineBreaks(in)

			if string(got) != tt.want || changed != tt.changed {
				t.Errorf("got %q, %v, want %q, %v", got, changed, tt.want, tt.changed)
			}
			if string(in) != tt.in {
				t.Errorf("the input was changed: %q", in)
			}

			// a second time changes nothing
			again, changedAgain := NormaliseHeaderLineBreaks(got)
			if changedAgain || !bytes.Equal(again, got) {
				t.Errorf("not stable: %q, %v", again, changedAgain)
			}

			// when nothing is changed the message is returned as it is, not copied
			if !changed && len(in) > 0 && &got[0] != &in[0] {
				t.Error("a message with nothing to change was copied")
			}
		})
	}
}

// refBreakRe finds the line breaks of a body, and refHeaderBreakRe those of the headers, where a <CR>
// on its own is not one.
var refBreakRe = regexp.MustCompile(`\r+\n|\n|\r`)
var refHeaderBreakRe = regexp.MustCompile(`\r*\n`)

// refSplit is another way of finding the end of the headers: the first empty line.
func refSplit(in string) (head, rest string) {
	pos := 0
	for _, loc := range refHeaderBreakRe.FindAllStringIndex(in, -1) {
		if loc[0] == pos {
			return in[:loc[1]], in[loc[1]:]
		}
		pos = loc[1]
	}

	return in, ""
}

// refBare returns whether there is a <CR> which is not followed by a <LF>.
func refBare(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && (i+1 >= len(s) || s[i+1] != '\n') {
			return true
		}
	}

	return false
}

// refNormalise returns the message with the line breaks of its headers as <CR><LF>, whether that changed
// anything, and whether the headers have a <CR> on its own.
func refNormalise(in string) (string, bool, bool) {
	head, rest := refSplit(in)
	changed := false
	head = refHeaderBreakRe.ReplaceAllStringFunc(head, func(br string) string {
		if br != "\r\n" {
			changed = true
		}

		return "\r\n"
	})

	return head + rest, changed, refBare(head)
}

func TestNormaliseHeaderLineBreaksMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	pieces := []string{"\r\n", "\r\n", "\n", "\r", "\r\r\n", "\r\r\r\n", " ", "\t", "a", "b", "From: a", "Bcc: x", ": ", "\r\n ", "\n\n", "\r\r", "\r\n\r\n", "é", "\x00"}

	for range 100000 {
		var b strings.Builder
		for range rng.Intn(14) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		in := b.String()

		got, changed := NormaliseHeaderLineBreaks([]byte(in))
		want, wantChanged, wantBare := refNormalise(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
		if HeadersHaveBareCR([]byte(in)) != wantBare {
			t.Fatalf("%q: a bare CR in the headers should be %v", in, wantBare)
		}
	}
}

func FuzzNormaliseHeaderLineBreaks(f *testing.F) {
	for _, s := range []string{"From: a\r\n\r\nbody", "Subject: one\rBcc: x\r\n\r\nb", "a\r\r\nb\n\nc\r", "", "\r", "\n"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, changed := NormaliseHeaderLineBreaks([]byte(in))
		want, wantChanged, wantBare := refNormalise(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
		if HeadersHaveBareCR([]byte(in)) != wantBare {
			t.Fatalf("%q: a bare CR in the headers should be %v", in, wantBare)
		}
		if again, changedAgain := NormaliseHeaderLineBreaks(got); changedAgain || !bytes.Equal(again, got) {
			t.Fatalf("%q is not stable", in)
		}
	})
}

// A bare CR is text to Go's parser and to Postfix, so "Subject: one<CR>Bcc: x" is one header (a Subject)
// to them, and Mailpit leaves it so that it reads the headers as they do. It is not a header which can
// be removed or replaced. What is sent has the bare CR replaced with a space, so it is no header to
// anything which takes a bare CR as a line break either.
func TestRewritingHeadersWithBareCR(t *testing.T) {
	// the headers of what would be sent on
	sent := func(t *testing.T, msg []byte) (mail.Header, string) {
		t.Helper()
		out, _ := NormaliseForSending(msg)
		h, _, ok := parseMsg(t, out)
		if !ok {
			t.Fatalf("not a message: %q", out)
		}

		return h, string(out)
	}

	t.Run("Bcc is not hidden in the Subject", func(t *testing.T) {
		msg := "From: a@b.test\r\nSubject: one\rBcc: hidden@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n"

		out, err := RemoveMessageHeaders([]byte(msg), []string{"Bcc"})
		if err != nil {
			t.Fatal(err)
		}
		if h, _, _ := parseMsg(t, out); hasHeader(h, "Bcc") {
			t.Errorf("Go reads a Bcc header: %q", out)
		}

		h, text := sent(t, out)
		if hasHeader(h, "Bcc") || regexp.MustCompile(`(?m)^Bcc:`).MatchString(text) {
			t.Errorf("a Bcc header is sent: %q", text)
		}
		if h.Get("Subject") != "one Bcc: hidden@x.test" || h.Get("To") != "c@d.test" {
			t.Errorf("Subject %q, To %q", h.Get("Subject"), h.Get("To"))
		}

		// control: the same with a line break is a Bcc header, and it is removed
		out, err = RemoveMessageHeaders([]byte(strings.Replace(msg, "one\rBcc", "one\r\nBcc", 1)), []string{"Bcc"})
		if err != nil {
			t.Fatal(err)
		}
		h, text = sent(t, out)
		if hasHeader(h, "Bcc") || strings.Contains(text, "hidden@x.test") || h.Get("Subject") != "one" {
			t.Errorf("control: %q", text)
		}
	})

	t.Run("From is the only one", func(t *testing.T) {
		msg := "Subject: one\rFrom: evil@x.test\r\nFrom: real@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n"

		out, err := OverrideFromHeader([]byte(msg), "operator@example.com")
		if err != nil {
			t.Fatal(err)
		}
		h, text := sent(t, out)
		if len(h["From"]) != 1 || !strings.Contains(h.Get("From"), "operator@example.com") {
			t.Errorf("From: %q in %q", h["From"], text)
		}
		if n := len(regexp.MustCompile(`(?m)^From:`).FindAllString(text, -1)); n != 1 {
			t.Errorf("%d lines start with From:, %q", n, text)
		}
		if h.Get("Subject") != "one From: evil@x.test" {
			t.Errorf("Subject: %q", h.Get("Subject"))
		}
	})

	t.Run("a header is replaced", func(t *testing.T) {
		msg := "From: a@b.test\r\nSubject: one\rReturn-Path: <evil@x.test>\r\n\r\nbody\r\n"

		out, err := SetMessageHeader([]byte(msg), "Return-Path", "<new@x.test>")
		if err != nil {
			t.Fatal(err)
		}
		h, text := sent(t, out)
		if len(h["Return-Path"]) != 1 || h.Get("Return-Path") != "<new@x.test>" {
			t.Errorf("Return-Path: %q in %q", h["Return-Path"], text)
		}
		if n := len(regexp.MustCompile(`(?m)^Return-Path:`).FindAllString(text, -1)); n != 1 {
			t.Errorf("%d lines start with Return-Path:, %q", n, text)
		}
	})

	t.Run("the Subject prefix", func(t *testing.T) {
		out, err := PrefixSubject([]byte("From: a@b.test\r\nSubject: one\rX-Hidden: yes\r\n\r\nbody\r\n"), "[dev]")
		if err != nil {
			t.Fatal(err)
		}
		h, text := sent(t, out)
		// the CR is encoded in the Subject, and is not a header
		if hasHeader(h, "X-Hidden") || refBare(text) || !strings.Contains(h.Get("Subject"), "=0D") {
			t.Errorf("Subject %q in %q", h.Get("Subject"), text)
		}

		// control: with a line break it is a header
		out, _ = PrefixSubject([]byte("From: a@b.test\r\nSubject: one\r\nX-Hidden: yes\r\n\r\nbody\r\n"), "[dev]")
		if h, _ := sent(t, out); h.Get("Subject") != "[dev] one" || h.Get("X-Hidden") != "yes" {
			t.Errorf("control: Subject %q, X-Hidden %q", h.Get("Subject"), h.Get("X-Hidden"))
		}
	})
}

// The headers which are stored (a bare CR is kept) are read the same by everything, and what is sent
// has the bare CR replaced, so there is no bare CR left for anything to read as a line break.
func TestNormalisedHeadersAreTheSameToEveryReader(t *testing.T) {
	msg := []byte("From: a@b.test\rSubject: one\nBcc: hidden@x.test\r\r\nTo: c@d.test\r\n\r\nbody\rmore\n")

	if !HeadersHaveBareCR(msg) {
		t.Error("the bare CR is not found")
	}

	stored, _ := NormaliseHeaderLineBreaks(msg)
	if !HeadersHaveBareCR(stored) {
		t.Error("the bare CR was removed from what is stored")
	}
	m, err := mail.ReadMessage(bytes.NewReader(stored))
	if err != nil {
		t.Fatal(err)
	}
	if v := m.Header.Get("From"); v != "a@b.test\rSubject: one" {
		t.Errorf("From is %q", v)
	}
	if !bytes.HasSuffix(stored, []byte("\r\n\r\nbody\rmore\n")) {
		t.Errorf("the body changed: %q", stored)
	}

	out, _ := NormaliseForSending(msg)
	m, err = mail.ReadMessage(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{"From": "a@b.test Subject: one", "Bcc": "hidden@x.test", "To": "c@d.test"} {
		if v := m.Header.Get(k); v != want {
			t.Errorf("%s header is %q, want %q", k, v, want)
		}
	}
	if HeadersHaveBareCR(out) || refBare(string(out)) {
		t.Errorf("a bare CR is left: %q", out)
	}
	if !bytes.HasSuffix(out, []byte("\r\n\r\nbody\r\nmore\r\n")) {
		t.Errorf("the body: %q", out)
	}
}

// refNormaliseAll is another way of doing the same for a message which is sent: the headers are
// as for refNormalise with a <CR> on its own made a space, and every line break of the body is found with
// an expression.
func refNormaliseAll(in string) (string, bool) {
	head, rest := refSplit(in)
	changed := false

	head = refHeaderBreakRe.ReplaceAllStringFunc(head, func(br string) string {
		if br != "\r\n" {
			changed = true
		}

		return "\r\n"
	})
	if refBare(head) {
		changed = true
		var b strings.Builder
		for i := 0; i < len(head); i++ {
			if head[i] == '\r' && (i+1 >= len(head) || head[i+1] != '\n') {
				b.WriteByte(' ')
			} else {
				b.WriteByte(head[i])
			}
		}
		head = b.String()
	}

	rest = refBreakRe.ReplaceAllStringFunc(rest, func(br string) string {
		if br != "\r\n" {
			changed = true
		}

		return "\r\n"
	})

	return head + rest, changed
}

func TestNormaliseLineBreaks(t *testing.T) {
	tests := []struct {
		name, in, want string
		changed        bool
	}{
		{"control: CRLF", "From: a\r\n\r\nbody\r\nmore\r\n", "From: a\r\n\r\nbody\r\nmore\r\n", false},
		{"control: no line breaks", "text", "text", false},
		{"control: empty", "", "", false},

		// the body as well as the headers
		{"control: a CR in the headers is text, and is sent as a space", "From: a\rSubject: b\r\n\r\nbody", "From: a Subject: b\r\n\r\nbody", true},
		{"exploit: CRs in the headers", "A: x\r\rB: y\r\n\r\nbody", "A: x  B: y\r\n\r\nbody", true},
		{"exploit: a CR in the body", "From: a\r\n\r\nbody\rmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: a LF in the body", "From: a\r\n\r\nbody\nmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: CR CR LF in the body", "From: a\r\n\r\nbody\r\r\nmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: a CR at the end", "From: a\r\n\r\nbody\r", "From: a\r\n\r\nbody\r\n", true},
		{"exploit: a dot after a CR", "A: x\r\n\r\nx\r.\r\ny", "A: x\r\n\r\nx\r\n.\r\ny", true},
		{"exploit: a dot after a LF", "A: x\n\nx\n.\ny", "A: x\r\n\r\nx\r\n.\r\ny", true},
		{"exploit: a dot after a CR in the headers", "A: x\r.\r\n\r\ny", "A: x .\r\n\r\ny", true},
		{"exploit: every kind", "A: x\r\n\r\na\rb\nc\r\r\nd\r\ne", "A: x\r\n\r\na\r\nb\r\nc\r\nd\r\ne", true},
		{"exploit: headers and body", "A: x\rB: y\n\n\rbody\n", "A: x B: y\r\n\r\n\r\nbody\r\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(tt.in)
			got, changed := NormaliseLineBreaks(in)

			if string(got) != tt.want || changed != tt.changed {
				t.Errorf("got %q, %v, want %q, %v", got, changed, tt.want, tt.changed)
			}
			if string(in) != tt.in {
				t.Errorf("the input was changed: %q", in)
			}
			if again, changedAgain := NormaliseLineBreaks(got); changedAgain || !bytes.Equal(again, got) {
				t.Errorf("not stable: %q, %v", again, changedAgain)
			}
			if !changed && len(in) > 0 && &got[0] != &in[0] {
				t.Error("a message with nothing to change was copied")
			}
		})
	}
}

func TestNormaliseLineBreaksMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	pieces := []string{"\r\n", "\r\n", "\n", "\r", "\r\r\n", "\r\r\r\n", " ", "a", "bc", ".", "\r\n.\r\n", "\r.\r\n", "\n.\n", "é", "\x00", "From: a", "\r\n\r\n"}

	for range 100000 {
		var b strings.Builder
		for range rng.Intn(14) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		in := b.String()

		got, changed := NormaliseLineBreaks([]byte(in))
		want, wantChanged := refNormaliseAll(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
	}
}

func FuzzNormaliseLineBreaks(f *testing.F) {
	for _, s := range []string{"From: a\r\n\r\nbody\r.\r\nx", "a\rb\nc\r\r\nd", "", "\r", "\n.\n"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, changed := NormaliseLineBreaks([]byte(in))
		want, wantChanged := refNormaliseAll(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}

		// nothing but <CR><LF> is left
		for i := 0; i < len(got); i++ {
			if (got[i] == '\r' && (i+1 >= len(got) || got[i+1] != '\n')) || (got[i] == '\n' && (i == 0 || got[i-1] != '\r')) {
				t.Fatalf("%q has a line break which is not CRLF: %q", in, got)
			}
		}
	})
}

// The largest message is 50 MiB, and it is checked for every message which is sent on.
func TestNormaliseLineBreaksLargeMessage(t *testing.T) {
	for _, shape := range []struct{ name, prefix, line string }{
		{"compliant", "", "a line of text which is fairly typical\r\n"},
		{"bare LF", "", "a line of text which is fairly typical\n"},
		{"bare CR in the headers", "", "a line of text which is fairly typical\r"},
		{"bare CR in the body", "\r\n", "a line of text which is fairly typical\r"},
	} {
		msg := []byte(shape.prefix + strings.Repeat(shape.line, (50<<20)/len(shape.line)))

		start := time.Now()
		got, _ := NormaliseLineBreaks(msg)
		// about a tenth of a second, and several times that with the race detector
		if d := time.Since(start); d > 5*time.Second {
			t.Errorf("%s: 50 MiB took %v", shape.name, d)
		}
		if len(got) < len(msg) {
			t.Errorf("%s: the message is shorter", shape.name)
		}
	}
}
