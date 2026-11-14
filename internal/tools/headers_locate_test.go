package tools

import (
	"bytes"
	"net/mail"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func parseMsg(tb testing.TB, msg []byte) (mail.Header, string, bool) {
	tb.Helper()

	m, err := mail.ReadMessage(bytes.NewReader(msg))
	if err != nil {
		return nil, "", false
	}
	var body bytes.Buffer
	_, _ = body.ReadFrom(m.Body)

	return m.Header, body.String(), true
}

func crlf(s string) []byte { return []byte(strings.ReplaceAll(s, "\n", "\r\n")) }

// Each of these used to leave the header in place without an error: the text of the header was
// searched for in the message, and it was not found (or found somewhere else) when the message had
// LF line endings, the header was the last thing in it, or the same text was in a folded line before it.
var locateCases = []struct {
	name string
	msg  []byte
}{
	{"control: CRLF", crlf("Subject: hi\nFrom: evil@x.test\nTo: a@b.test\n\nbody\n")},
	{"exploit: LF line endings", []byte("Subject: hi\nFrom: evil@x.test\nTo: a@b.test\n\nbody\n")},
	{"exploit: same text in an earlier folded header", crlf("Subject: hi\nX-Foo: a\n From: evil@x.test\nFrom: evil@x.test\nTo: a@b.test\n\nbody\n")},
	{"exploit: same text in an earlier folded header, LF", []byte("Subject: hi\nX-Foo: a\n From: evil@x.test\nFrom: evil@x.test\n\nbody\n")},
	{"exploit: last line, no line break", []byte("Subject: hi\r\nFrom: evil@x.test")},
	{"exploit: only line, no line break", []byte("From: evil@x.test")},
	{"exploit: mixed line endings", []byte("Subject: hi\r\nFrom: evil@x.test\nTo: a@b.test\r\n\r\nbody\r\n")},
	{"header name in other case", crlf("Subject: hi\nFROM: evil@x.test\n\nbody\n")},
}

func TestOverrideFromHeaderLocatesTheHeader(t *testing.T) {
	for _, tt := range locateCases {
		t.Run(tt.name, func(t *testing.T) {
			h0, body0, _ := parseMsg(t, tt.msg)

			out, err := OverrideFromHeader(tt.msg, "operator@example.com")
			if err != nil {
				t.Fatal(err)
			}

			h1, body1, ok := parseMsg(t, out)
			if !ok {
				t.Fatalf("the result does not parse: %q", out)
			}
			if body1 != body0 {
				t.Errorf("body changed: %q", body1)
			}
			if v := h1["From"]; len(v) != 1 || !strings.Contains(v[0], "operator@example.com") {
				t.Errorf("From is %q", v)
			}
			if h1.Get("X-Original-From") != "evil@x.test" {
				t.Errorf("X-Original-From is %q", h1.Get("X-Original-From"))
			}
			if h1.Get("Subject") != h0.Get("Subject") || h1.Get("To") != h0.Get("To") {
				t.Errorf("other headers changed: %v", h1)
			}
		})
	}
}

func TestRemoveMessageHeadersLocatesTheHeader(t *testing.T) {
	for _, tt := range locateCases {
		t.Run(tt.name, func(t *testing.T) {
			h0, body0, _ := parseMsg(t, tt.msg)

			out, err := RemoveMessageHeaders(tt.msg, []string{"From"})
			if err != nil {
				t.Fatal(err)
			}
			if len(out) == 0 {
				// the only header was removed, and an empty message has nothing to parse
				return
			}

			h1, body1, ok := parseMsg(t, out)
			if !ok {
				t.Fatalf("the result does not parse: %q", out)
			}
			if hasHeader(h1, "From") {
				t.Errorf("From is still there: %q", out)
			}
			if body1 != body0 {
				t.Errorf("body changed: %q", body1)
			}
			// a folded line which looks like a From header is part of the header before it, and stays
			if h1.Get("X-Foo") != h0.Get("X-Foo") {
				t.Errorf("X-Foo changed from %q to %q", h0.Get("X-Foo"), h1.Get("X-Foo"))
			}
		})
	}
}

func TestSetMessageHeaderLocatesTheHeader(t *testing.T) {
	for _, tt := range locateCases {
		t.Run(tt.name, func(t *testing.T) {
			_, body0, _ := parseMsg(t, tt.msg)

			out, err := SetMessageHeader(tt.msg, "From", "set@example.com")
			if err != nil {
				t.Fatal(err)
			}

			h1, body1, ok := parseMsg(t, out)
			if !ok {
				t.Fatalf("the result does not parse: %q", out)
			}
			if v := h1["From"]; len(v) != 1 || v[0] != "set@example.com" {
				t.Errorf("From is %q", v)
			}
			if body1 != body0 {
				t.Errorf("body changed: %q", body1)
			}
		})
	}
}

// Bcc is removed wherever it is, which is what keeps the recipients private when a message is released.
func TestRemoveMessageHeadersRemovesEveryInstance(t *testing.T) {
	msg := crlf("From: a@b.test\nBcc: one@x.test\nTo: c@d.test\nbcc: two@x.test,\n three@x.test\nSubject: s\nBCC: four@x.test\n\nbody\n")

	out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		t.Fatal(err)
	}

	h, body, ok := parseMsg(t, out)
	if !ok || hasHeader(h, "Bcc") || strings.Contains(string(out), "x.test") {
		t.Fatalf("Bcc is still there: %q", out)
	}
	if h.Get("To") != "c@d.test" || h.Get("Subject") != "s" || h.Get("From") != "a@b.test" || body != "body\r\n" {
		t.Errorf("something else changed: %q", out)
	}

	// control: only the header asked for
	out, _ = RemoveMessageHeaders(msg, []string{"Subject"})
	if h, _, _ := parseMsg(t, out); hasHeader(h, "Subject") || len(h["Bcc"]) != 3 {
		t.Errorf("removing Subject changed Bcc: %q", out)
	}
}

// A header that is present but empty is a header, so it is replaced rather than left beside a new one.
func TestEmptyHeadersAreHeaders(t *testing.T) {
	msg := crlf("From: a@b.test\nBcc:\nReturn-Path:\n\nbody\n")

	out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
	if err != nil {
		t.Fatal(err)
	}
	if h, _, _ := parseMsg(t, out); hasHeader(h, "Bcc") {
		t.Errorf("an empty Bcc is still there: %q", out)
	}

	out, err = SetMessageHeader(msg, "Return-Path", "<x@y.test>")
	if err != nil {
		t.Fatal(err)
	}
	if h, _, _ := parseMsg(t, out); len(h["Return-Path"]) != 1 || h.Get("Return-Path") != "<x@y.test>" {
		t.Errorf("Return-Path is %q", h["Return-Path"])
	}

	msg = crlf("From:\nSubject: s\n\nbody\n")
	out, err = OverrideFromHeader(msg, "operator@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if h, _, _ := parseMsg(t, out); len(h["From"]) != 1 || h.Get("From") != "operator@example.com" || hasHeader(h, "X-Original-From") {
		t.Errorf("an empty From: %q", out)
	}
}

// A second From would otherwise be sent through the relay beside the operator's address.
func TestOverrideFromHeaderRemovesOtherFromHeaders(t *testing.T) {
	msg := crlf("From: first@x.test\nSubject: s\nFrom: second@x.test\n\nbody\n")

	out, err := OverrideFromHeader(msg, "operator@example.com")
	if err != nil {
		t.Fatal(err)
	}

	h, _, _ := parseMsg(t, out)
	if len(h["From"]) != 1 || !strings.Contains(h.Get("From"), "operator@example.com") || strings.Contains(string(out), "second@x.test") {
		t.Errorf("From headers: %q in %q", h["From"], out)
	}
	if h.Get("X-Original-From") != "first@x.test" {
		t.Errorf("X-Original-From is %q", h.Get("X-Original-From"))
	}
}

func TestFoldedHeaderIsReplacedWhole(t *testing.T) {
	msg := crlf("From: a@b.test\nBcc: one@x.test,\n\ttwo@x.test,\n  three@x.test\nSubject: s\n\nbody\n")

	out, err := SetMessageHeader(msg, "Bcc", "new@x.test")
	if err != nil {
		t.Fatal(err)
	}
	h, _, _ := parseMsg(t, out)
	if len(h["Bcc"]) != 1 || h.Get("Bcc") != "new@x.test" || strings.Contains(string(out), "three@x.test") || h.Get("Subject") != "s" {
		t.Errorf("%q", out)
	}
}

// The original value is kept when it is not an address, folded lines are joined, and it cannot add a header.
func TestOverrideFromHeaderKeepsUnparsableValue(t *testing.T) {
	msg := crlf("From: not an address,\n  at all\nSubject: s\n\nbody\n")

	out, err := OverrideFromHeader(msg, "operator@example.com")
	if err != nil {
		t.Fatal(err)
	}
	h, _, _ := parseMsg(t, out)
	if h.Get("From") != "operator@example.com" || h.Get("X-Original-From") != "not an address, at all" {
		t.Errorf("%q", out)
	}
}

// Finding the header is a single pass, whatever the number of instances or the size of the message.
func TestHeaderRewritingIsLinear(t *testing.T) {
	// a million Bcc headers (7 MB), which a search for each one would take a very long time to remove
	many := []byte("From: a@b.test\r\n" + strings.Repeat("Bcc: a@x.test\r\n", 1_000_000) + "\r\nbody\r\n")

	start := time.Now()
	out, err := RemoveMessageHeaders(many, []string{"Bcc"})
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("removing a million headers took %v", d)
	}
	if h, _, _ := parseMsg(t, out); hasHeader(h, "Bcc") {
		t.Error("Bcc is still there")
	}

	// the largest message (50 MiB) with the header after a lot of other ones
	big := []byte("X-Pad: " + strings.Repeat("a", 25<<20) + "\r\nFrom: evil@x.test\r\n\r\n" + strings.Repeat("b", 25<<20))
	start = time.Now()
	out, err = OverrideFromHeader(big, "operator@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("a 50 MiB message took %v", d)
	}
	if h, _, _ := parseMsg(t, out); !strings.Contains(h.Get("From"), "operator@example.com") {
		t.Error("From was not replaced")
	}
}

// For any message that parses: the header asked for is replaced or removed, the other headers and
// the body are as they were, and the result parses.
func FuzzHeaderRewriting(f *testing.F) {
	f.Add("From: evil@x.test\nBcc: a@x.test\nTo: c@d.test\n\nbody\n")
	f.Add("Subject: hi\nX-Foo: a\n From: evil@x.test\nFrom: evil@x.test\n\nbody\n")
	f.Add("From: a@b.test\nBcc:\nBcc: x@y.test,\n z@y.test\n\nbody")
	f.Add("From: a@b.test")
	f.Add("From: <>\r\nfrom: b@c.test\r\n\r\n")

	f.Fuzz(func(t *testing.T, s string) {
		msg := []byte(s)
		h0, body0, ok := parseMsg(t, msg)
		if !ok {
			return
		}

		same := func(name string, h1 mail.Header, skip ...string) {
			t.Helper()
			for k, v := range h0 {
				if textproto.CanonicalMIMEHeaderKey(k) == name || containsFold(skip, k) {
					continue
				}
				if strings.Join(h1[k], "\x00") != strings.Join(v, "\x00") {
					t.Fatalf("header %s changed:\n%q", k, msg)
				}
			}
			for k := range h1 {
				if _, ok := h0[k]; !ok && k != name && !containsFold(skip, k) {
					t.Fatalf("header %s added:\n%q", k, msg)
				}
			}
		}

		// removing
		if out, err := RemoveMessageHeaders(msg, []string{"Bcc"}); err == nil {
			h1, body1, ok := parseMsg(t, out)
			if !ok && onlyHeaders(h0, "Bcc") {
				// the only header was removed, and a message with no headers is not read
				return
			}
			if !ok || body1 != body0 || hasHeader(h1, "Bcc") {
				t.Fatalf("remove: ok=%v bcc=%v body changed=%v\n%q\n%q", ok, hasHeader(h1, "Bcc"), body1 != body0, msg, out)
			}
			same("Bcc", h1)
		}

		// setting
		if out, err := SetMessageHeader(msg, "Return-Path", "<set@example.com>"); err == nil {
			h1, body1, ok := parseMsg(t, out)
			if !ok || body1 != body0 || len(h1["Return-Path"]) == 0 || h1["Return-Path"][0] != "<set@example.com>" {
				t.Fatalf("set: ok=%v value=%q\n%q\n%q", ok, h1["Return-Path"], msg, out)
			}
			if len(h1["Return-Path"]) != max(1, len(h0["Return-Path"])) {
				t.Fatalf("set: %d instances, were %d\n%q\n%q", len(h1["Return-Path"]), len(h0["Return-Path"]), msg, out)
			}
			same("Return-Path", h1)
		}

		// overriding the sender
		if out, err := OverrideFromHeader(msg, "operator@example.com"); err == nil {
			h1, body1, ok := parseMsg(t, out)
			if !ok || body1 != body0 || len(h1["From"]) != 1 {
				t.Fatalf("override: ok=%v From=%q\n%q\n%q", ok, h1["From"], msg, out)
			}
			if a, err := mail.ParseAddress(h1["From"][0]); err != nil || a.Address != "operator@example.com" {
				t.Fatalf("override: From is %q\n%q\n%q", h1["From"][0], msg, out)
			}
			same("From", h1, "X-Original-From")
		}
	})
}

// onlyHeaders returns whether h has no headers other than the one named.
func onlyHeaders(h mail.Header, name string) bool {
	for k := range h {
		if textproto.CanonicalMIMEHeaderKey(k) != name {
			return false
		}
	}

	return true
}

func containsFold(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}

	return false
}

// Only the headers are read, so the same text in the body is left alone.
func TestHeaderRewritingLeavesTheBodyAlone(t *testing.T) {
	for _, nl := range []string{"\r\n", "\n"} {
		body := "From: in-the-body@x.test" + nl + "Bcc: body@x.test" + nl
		msg := []byte("From: evil@x.test" + nl + "Bcc: secret@x.test" + nl + nl + body)

		out, err := RemoveMessageHeaders(msg, []string{"From", "Bcc"})
		if err != nil {
			t.Fatal(err)
		}
		if got := string(out); got != nl+body {
			t.Errorf("%q line endings: %q", nl, got)
		}

		out, err = OverrideFromHeader(msg, "operator@example.com")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(string(out), nl+nl+body) {
			t.Errorf("%q line endings, override: the body changed: %q", nl, out)
		}
	}
}

// A header folded over several lines is found whole, whatever the line endings are, and whether
// the fold lines are indented with a space or a tab.
func TestFoldedTargetHeadersAnyLineEnding(t *testing.T) {
	cases := []struct{ name, msg string }{
		{"CRLF", "From: a@b.test\r\nBcc: one@x.test,\r\n two@x.test,\r\n\tthree@x.test\r\nSubject: s\r\n\r\nbody\r\n"},
		{"LF", "From: a@b.test\nBcc: one@x.test,\n two@x.test,\n\tthree@x.test\nSubject: s\n\nbody\n"},
		{"mixed", "From: a@b.test\r\nBcc: one@x.test,\n two@x.test,\r\n\tthree@x.test\nSubject: s\r\n\r\nbody\r\n"},
		{"trailing spaces", "From: a@b.test\r\nBcc: one@x.test,   \r\n two@x.test  \r\nSubject: s\r\n\r\nbody\r\n"},
		{"a fold line with only whitespace", "From: a@b.test\r\nBcc: one@x.test\r\n \r\nSubject: s\r\n\r\nbody\r\n"},
		{"blank line with a CR, LF headers", "From: a@b.test\nBcc: one@x.test\n\r\nbody\n"},
		{"ends the headers, no line break", "From: a@b.test\nBcc: one@x.test,\n two@x.test"},
		{"first header", "Bcc: one@x.test,\n two@x.test\nSubject: s\n\nbody\n"},
		{"two folded instances", "Bcc: one@x.test,\n two@x.test\nSubject: s\nBcc: three@x.test,\n four@x.test\n\nbody\n"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			msg := []byte(tt.msg)
			h0, body0, ok := parseMsg(t, msg)
			if !ok {
				t.Fatal("the test message does not parse")
			}

			// removing
			out, err := RemoveMessageHeaders(msg, []string{"Bcc"})
			if err != nil {
				t.Fatal(err)
			}
			h1, body1, ok := parseMsg(t, out)
			if !ok || hasHeader(h1, "Bcc") || strings.Contains(string(out), "x.test") || body1 != body0 {
				t.Errorf("remove: parses=%v, Bcc present=%v, body same=%v: %q", ok, hasHeader(h1, "Bcc"), body1 == body0, out)
			}
			if h1.Get("Subject") != h0.Get("Subject") {
				t.Errorf("remove: Subject changed: %q", out)
			}

			// replacing: only the first instance, and none of its fold lines are left
			out, err = SetMessageHeader(msg, "Bcc", "new@x.test")
			if err != nil {
				t.Fatal(err)
			}
			h1, body1, ok = parseMsg(t, out)
			if !ok || len(h1["Bcc"]) != len(h0["Bcc"]) || h1["Bcc"][0] != "new@x.test" || body1 != body0 {
				t.Errorf("set: parses=%v, Bcc=%q, body same=%v: %q", ok, h1["Bcc"], body1 == body0, out)
			}
			if strings.Contains(string(out), "one@x.test") || strings.Contains(string(out), "two@x.test") {
				t.Errorf("set: the first header's fold lines are still there: %q", out)
			}
		})
	}

	// a folded From is replaced whole, and kept in X-Original-From
	for _, nl := range []string{"\r\n", "\n", "\r\n-\n"} {
		first, second := nl, nl
		if nl == "\r\n-\n" {
			first, second = "\r\n", "\n"
		}
		msg := []byte("From: Evil Name" + first + " <evil@x.test>" + second + "Subject: s" + first + first + "body" + first)

		out, err := OverrideFromHeader(msg, "op@example.com")
		if err != nil {
			t.Fatal(err)
		}
		h, _, ok := parseMsg(t, out)
		if !ok || len(h["From"]) != 1 || !strings.Contains(h.Get("From"), "op@example.com") || h.Get("X-Original-From") != "evil@x.test" || h.Get("Subject") != "s" {
			t.Errorf("override %q: %q", nl, out)
		}
	}
}
