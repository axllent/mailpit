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

		// the line breaks which are not CRLF
		{"exploit: LF", "From: a\nTo: b\n\nbody\n", "From: a\r\nTo: b\r\n\r\nbody\n", true},
		{"exploit: CR in the middle of a value", "Subject: one\rBcc: x\r\n\r\nbody", "Subject: one\r\nBcc: x\r\n\r\nbody", true},
		{"exploit: CR CR LF", "From: a\r\r\nTo: b\r\r\n\r\r\nbody", "From: a\r\nTo: b\r\n\r\nbody", true},
		{"exploit: more than two CR", "From: a\r\r\r\nTo: b\r\n\r\nbody", "From: a\r\nTo: b\r\n\r\nbody", true},
		{"exploit: CR only", "From: a\rSubject: b\r\rbody\r", "From: a\r\nSubject: b\r\n\r\nbody\r", true},
		{"exploit: mixed", "From: a\r\nSubject: b\nTo: c\rX: d\r\r\n\r\nbody", "From: a\r\nSubject: b\r\nTo: c\r\nX: d\r\n\r\nbody", true},
		{"exploit: folded line with LF", "Bcc: a,\n b\nTo: x\n\nbody", "Bcc: a,\r\n b\r\nTo: x\r\n\r\nbody", true},
		{"exploit: a bare CR is the empty line", "From: a\r\nTo: b\r\n\rbody", "From: a\r\nTo: b\r\n\r\nbody", true},
		{"exploit: LF first", "\nbody\r", "\r\nbody\r", true},
		{"exploit: only a CR", "\r", "\r\n", true},
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

// refNormalise is another way of doing the same: the line breaks are found with an expression, and
// the first empty line ends the headers.
var refBreakRe = regexp.MustCompile(`\r+\n|\n|\r`)

func refNormalise(in string) (string, bool) {
	var out strings.Builder
	changed := false
	pos := 0

	for _, loc := range refBreakRe.FindAllStringIndex(in, -1) {
		line := in[pos:loc[0]]
		br := in[loc[0]:loc[1]]
		out.WriteString(line)
		out.WriteString("\r\n")
		if br != "\r\n" {
			changed = true
		}
		pos = loc[1]

		if line == "" {
			// the empty line which ends the headers
			break
		}
	}

	out.WriteString(in[pos:])

	return out.String(), changed
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
		want, wantChanged := refNormalise(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
	}
}

func FuzzNormaliseHeaderLineBreaks(f *testing.F) {
	for _, s := range []string{"From: a\r\n\r\nbody", "Subject: one\rBcc: x\r\n\r\nb", "a\r\r\nb\n\nc\r", "", "\r", "\n"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, changed := NormaliseHeaderLineBreaks([]byte(in))
		want, wantChanged := refNormalise(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
		if again, changedAgain := NormaliseHeaderLineBreaks(got); changedAgain || !bytes.Equal(again, got) {
			t.Fatalf("%q is not stable", in)
		}
	})
}

// A bare CR is data to Go's parser but a line break to some other software. These messages hide a
// header inside another one, so the header which is removed or replaced was not seen (and was
// still sent on). The line breaks are fixed first, so the header is where the other software sees it.
func TestRewritingHeadersWithBareCR(t *testing.T) {
	t.Run("Bcc is removed", func(t *testing.T) {
		for _, msg := range []string{
			"From: a@b.test\r\nSubject: one\rBcc: hidden@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n",
			"From: a@b.test\rSubject: one\rBcc: hidden@x.test\rTo: c@d.test\r\rbody\r",
			"From: a@b.test\nSubject: one\rBcc: hidden@x.test\nTo: c@d.test\n\nbody\n",
		} {
			out, err := RemoveMessageHeaders([]byte(msg), []string{"Bcc"})
			if err != nil {
				t.Fatal(err)
			}
			h, _, ok := parseMsg(t, out)
			if !ok || hasHeader(h, "Bcc") || strings.Contains(string(out), "hidden@x.test") {
				t.Errorf("Bcc is still there: %q", out)
			}
			if h.Get("Subject") != "one" || h.Get("To") != "c@d.test" {
				t.Errorf("other headers changed: %q", out)
			}
		}

		// control: the same without the bare CR
		out, _ := RemoveMessageHeaders(crlf("From: a@b.test\nSubject: one\nBcc: hidden@x.test\nTo: c@d.test\n\nbody\n"), []string{"Bcc"})
		if h, _, _ := parseMsg(t, out); hasHeader(h, "Bcc") || h.Get("Subject") != "one" {
			t.Errorf("control: %q", out)
		}
	})

	t.Run("From is the only one", func(t *testing.T) {
		msg := "Subject: one\rFrom: evil@x.test\r\nFrom: real@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n"

		out, err := OverrideFromHeader([]byte(msg), "operator@example.com")
		if err != nil {
			t.Fatal(err)
		}
		h, _, ok := parseMsg(t, out)
		if !ok || len(h["From"]) != 1 || !strings.Contains(h.Get("From"), "operator@example.com") {
			t.Errorf("From: %q in %q", h["From"], out)
		}
		if n := len(regexp.MustCompile(`(?m)^From:`).FindAll(out, -1)); n != 1 || regexp.MustCompile(`(?m)^From: (evil|real)@`).Match(out) {
			t.Errorf("%d From headers, another one is still there: %q", n, out)
		}
		if h.Get("Subject") != "one" {
			t.Errorf("Subject: %q", h.Get("Subject"))
		}
	})

	t.Run("a header is replaced where the other software sees it", func(t *testing.T) {
		msg := "From: a@b.test\r\nSubject: one\rReturn-Path: <evil@x.test>\r\n\r\nbody\r\n"

		out, err := SetMessageHeader([]byte(msg), "Return-Path", "<new@x.test>")
		if err != nil {
			t.Fatal(err)
		}
		h, _, _ := parseMsg(t, out)
		if len(h["Return-Path"]) != 1 || h.Get("Return-Path") != "<new@x.test>" || strings.Contains(string(out), "evil@x.test") {
			t.Errorf("Return-Path: %q in %q", h["Return-Path"], out)
		}
	})

	t.Run("the Subject prefix", func(t *testing.T) {
		out, err := PrefixSubject([]byte("From: a@b.test\r\nSubject: one\rX-Hidden: yes\r\n\r\nbody\r\n"), "[dev]")
		if err != nil {
			t.Fatal(err)
		}
		h, _, _ := parseMsg(t, out)
		if h.Get("Subject") != "[dev] one" || h.Get("X-Hidden") != "yes" {
			t.Errorf("Subject %q, X-Hidden %q in %q", h.Get("Subject"), h.Get("X-Hidden"), out)
		}
	})
}

// Whatever the line breaks are, every reader finds the same headers afterwards.
func TestNormalisedHeadersAreTheSameToEveryReader(t *testing.T) {
	msg := []byte("From: a@b.test\rSubject: one\nBcc: hidden@x.test\r\r\nTo: c@d.test\r\n\r\nbody\rmore\n")

	out, _ := NormaliseHeaderLineBreaks(msg)

	m, err := mail.ReadMessage(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"From", "Subject", "Bcc", "To"} {
		v := m.Header.Get(k)
		if v == "" || strings.ContainsAny(v, "\r\n") {
			t.Errorf("%s header is %q", k, v)
		}
	}
	// and the part of the message which is not headers is as it was
	if !bytes.HasSuffix(out, []byte("\r\n\r\nbody\rmore\n")) {
		t.Errorf("the body changed: %q", out)
	}
}

// refNormaliseAll is another way of doing the same: every line break is found with an expression.
func refNormaliseAll(in string) (string, bool) {
	changed := false
	out := refBreakRe.ReplaceAllStringFunc(in, func(br string) string {
		if br != "\r\n" {
			changed = true
		}

		return "\r\n"
	})

	return out, changed
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
		{"exploit: a CR in the body", "From: a\r\n\r\nbody\rmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: a LF in the body", "From: a\r\n\r\nbody\nmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: CR CR LF in the body", "From: a\r\n\r\nbody\r\r\nmore", "From: a\r\n\r\nbody\r\nmore", true},
		{"exploit: a CR at the end", "From: a\r\n\r\nbody\r", "From: a\r\n\r\nbody\r\n", true},
		{"exploit: a dot after a CR", "x\r.\r\ny", "x\r\n.\r\ny", true},
		{"exploit: a dot after a LF", "x\n.\ny", "x\r\n.\r\ny", true},
		{"exploit: every kind", "a\rb\nc\r\r\nd\r\ne", "a\r\nb\r\nc\r\nd\r\ne", true},
		{"exploit: headers and body", "A: x\rB: y\n\n\rbody\n", "A: x\r\nB: y\r\n\r\n\r\nbody\r\n", true},
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
	for _, shape := range []struct{ name, line string }{
		{"compliant", "a line of text which is fairly typical\r\n"},
		{"bare LF", "a line of text which is fairly typical\n"},
		{"bare CR", "a line of text which is fairly typical\r"},
	} {
		msg := []byte(strings.Repeat(shape.line, (50<<20)/len(shape.line)))

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
