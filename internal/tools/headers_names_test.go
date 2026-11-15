package tools

import (
	"bytes"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

func TestNormaliseHeaderNames(t *testing.T) {
	tests := []struct {
		name, in, want string
		changed        bool
	}{
		// controls: nothing to change
		{"control: normal headers", "From: a\r\nTo: b\r\n\r\nbody", "From: a\r\nTo: b\r\n\r\nbody", false},
		{"control: a colon in the value", "Subject: a : b\r\n\r\n", "Subject: a : b\r\n\r\n", false},
		{"control: a folded line which looks like a header", "Subject: a\r\n From : b\r\n\r\n", "Subject: a\r\n From : b\r\n\r\n", false},
		{"control: empty", "", "", false},
		{"control: no headers", "\r\nFrom : in the body\r\n", "\r\nFrom : in the body\r\n", false},

		// the whitespace between the name and the colon
		{"exploit: a space", "From : a\r\n\r\n", "From: a\r\n\r\n", true},
		{"exploit: a tab", "Bcc\t: x\r\n\r\n", "Bcc: x\r\n\r\n", true},
		{"exploit: several", "Subject \t  : x\r\n\r\n", "Subject: x\r\n\r\n", true},
		{"exploit: no value", "Bcc :\r\n\r\n", "Bcc:\r\n\r\n", true},
		{"exploit: a colon in the value too", "Subject : a : b\r\n\r\n", "Subject: a : b\r\n\r\n", true},
		{"exploit: the space after the colon is kept", "From :   a\r\n\r\n", "From:   a\r\n\r\n", true},
		{"exploit: with folded lines after it", "Subject : a\r\n b\r\n\tc\r\n\r\n", "Subject: a\r\n b\r\n\tc\r\n\r\n", true},
		{"exploit: several headers", "From : a\r\nTo: b\r\nBcc : c\r\n\r\nbody", "From: a\r\nTo: b\r\nBcc: c\r\n\r\nbody", true},
		{"exploit: the last line, with no line break", "From: a\r\nBcc : c", "From: a\r\nBcc: c", true},
		{"exploit: not at the end of the headers", "A : 1\r\n\r\nB : 2\r\n", "A: 1\r\n\r\nB : 2\r\n", true},

		// what is not a header name is left as it is
		{"line which is not a header: spaces in the name", "Not a header : x\r\n\r\n", "Not a header : x\r\n\r\n", false},
		{"line which is not a header: no name", " : x\r\n\r\n", " : x\r\n\r\n", false},
		{"line which is not a header: colon first", ": x\r\n\r\n", ": x\r\n\r\n", false},
		{"line which is not a header: no colon", "no colon here \r\n\r\n", "no colon here \r\n\r\n", false},
		{"line which is not a header: a mbox line", "From user@example.com Sat Jan  1 00:00:00 2000\r\nTo: b\r\n\r\n", "From user@example.com Sat Jan  1 00:00:00 2000\r\nTo: b\r\n\r\n", false},
		{"line which is not a header: not ASCII", "Fröm : a\r\n\r\n", "Fröm : a\r\n\r\n", false},
		{"line which is not a header: a control character", "Fr\x01om : a\r\n\r\n", "Fr\x01om : a\r\n\r\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := []byte(tt.in)
			got, changed := normaliseHeaderNames(in)

			if string(got) != tt.want || changed != tt.changed {
				t.Errorf("got %q, %v, want %q, %v", got, changed, tt.want, tt.changed)
			}
			if string(in) != tt.in {
				t.Errorf("the input was changed: %q", in)
			}
			if again, changedAgain := normaliseHeaderNames(got); changedAgain || !bytes.Equal(again, got) {
				t.Errorf("not stable: %q, %v", again, changedAgain)
			}
			if !changed && len(in) > 0 && &got[0] != &in[0] {
				t.Error("a message with nothing to change was copied")
			}
		})
	}
}

// refHeaderNames is another way of doing the same: the lines of the header block, with an expression.
var refNameRe = regexp.MustCompile(`^([\x21-\x39\x3b-\x7e]+)[ \t]+:`)

func refHeaderNames(in string) (string, bool) {
	lines := strings.SplitAfter(in, "\r\n")
	changed := false
	for i, line := range lines {
		if strings.TrimSuffix(line, "\r\n") == "" {
			break
		}
		if line[0] == ' ' || line[0] == '\t' {
			continue
		}
		if m := refNameRe.FindStringSubmatchIndex(line); m != nil {
			lines[i] = line[m[2]:m[3]] + line[m[1]-1:]
			changed = true
		}
	}

	return strings.Join(lines, ""), changed
}

func TestNormaliseHeaderNamesMatchesReference(t *testing.T) {
	rng := rand.New(rand.NewSource(5))
	pieces := []string{"From", "Bcc", "Sub ject", "X-A", " ", "  ", "\t", ":", " :", "\t:", ": ", "a", "é", "\x01", "\r\n", "\r\n", "\r\n ", "\r\n\t", "\r\n\r\n", "x : y", "Name : v\r\n"}

	for range 100000 {
		var b strings.Builder
		for range rng.Intn(14) {
			b.WriteString(pieces[rng.Intn(len(pieces))])
		}
		in := b.String()
		// the function is for a header block with <CR><LF> line breaks, and a line may only end with one
		if strings.Contains(strings.ReplaceAll(in, "\r\n", ""), "\r") || strings.Contains(strings.ReplaceAll(in, "\r\n", ""), "\n") {
			continue
		}

		got, changed := normaliseHeaderNames([]byte(in))
		want, wantChanged := refHeaderNames(in)
		if string(got) != want || changed != wantChanged {
			t.Fatalf("%q: got %q, %v, want %q, %v", in, got, changed, want, wantChanged)
		}
	}
}

func FuzzNormaliseHeaders(f *testing.F) {
	for _, s := range []string{"From : a\r\n\r\nbody", "Bcc\t: x\nTo : y\n\nz", "Subject: a\rBcc : b\r\n\r\n", "a : b\r\n c : d\r\n", ""} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, in string) {
		got, changed := NormaliseHeaders([]byte(in))

		if again, changedAgain := NormaliseHeaders(got); changedAgain || !bytes.Equal(again, got) {
			t.Fatalf("%q is not stable: %q", in, again)
		}
		if !changed && string(got) != in {
			t.Fatalf("%q changed to %q without saying so", in, got)
		}
		// no header has whitespace before its colon in the headers which are left
		for _, line := range strings.Split(string(got), "\r\n") {
			if line == "" {
				// the end of the headers, which can be the first line
				break
			}
			if line[0] != ' ' && line[0] != '\t' && refNameRe.MatchString(line) {
				t.Fatalf("%q has a header with whitespace before the colon after %q", in, line)
			}
		}
	})
}

// With the whitespace before the colon, Go's parser had the header as "Bcc " and not "Bcc", so what is removed or
// replaced by name was not found. Postfix reads them as the header, and drops "Bcc :" as it does "Bcc:".
func TestRewritingHeadersWithWhitespaceBeforeTheColon(t *testing.T) {
	t.Run("From is the only one", func(t *testing.T) {
		msg := "From: real@x.test\r\nFrom : evil@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n"

		out, err := OverrideFromHeader([]byte(msg), "operator@example.com")
		if err != nil {
			t.Fatal(err)
		}
		h, _, ok := parseMsg(t, out)
		if !ok || len(h["From"]) != 1 || !strings.Contains(h.Get("From"), "operator@example.com") {
			t.Errorf("From: %q in %q", h["From"], out)
		}
		if strings.Contains(string(out), "evil@x.test") || len(h["From "]) != 0 {
			t.Errorf("the other From is still there: %q", out)
		}
	})

	t.Run("Bcc is removed", func(t *testing.T) {
		msg := "From: a@x.test\r\nBcc : hidden@x.test\r\nBcc\t: hidden2@x.test\r\nBcc: hidden3@x.test\r\n\r\nbody\r\n"

		out, err := RemoveMessageHeaders([]byte(msg), []string{"Bcc"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "hidden") {
			t.Errorf("a Bcc is still there: %q", out)
		}
	})

	t.Run("a header is replaced once", func(t *testing.T) {
		msg := "From: a@x.test\r\nReturn-Path : <old@x.test>\r\n\r\nbody\r\n"

		out, err := SetMessageHeader([]byte(msg), "Return-Path", "<new@x.test>")
		if err != nil {
			t.Fatal(err)
		}
		h, _, _ := parseMsg(t, out)
		if len(h["Return-Path"]) != 1 || h.Get("Return-Path") != "<new@x.test>" || strings.Contains(string(out), "old@x.test") {
			t.Errorf("%q in %q", h["Return-Path"], out)
		}
	})

	// A name and its colon on two lines is read as a header by neither Postfix nor Go, so it is as it was.
	t.Run("a name and its colon on two lines is not changed", func(t *testing.T) {
		msg := "From: a@x.test\r\nBcc\r\n : kept@x.test\r\n\r\nbody\r\n"

		out, err := RemoveMessageHeaders([]byte(msg), []string{"Bcc"})
		if err != nil || string(out) != msg {
			t.Errorf("%q, %v", out, err)
		}
	})
}
