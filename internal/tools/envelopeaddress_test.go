package tools

import (
	"net/mail"
	"strings"
	"testing"
)

// The address is used as it is in the command: no name or comment, and quotes on a local part which needs them (the
// address of the parser has none, so what was a b@example.com for "a b"@example.com can not be used).
func TestEnvelopeAddress(t *testing.T) {
	for in, want := range map[string]string{
		// control: nothing to change
		"a@b.test":           "a@b.test",
		"a.b@example.test":   "a.b@example.test",
		"a+tag@example.test": "a+tag@example.test",
		"<a@b.test>":         "a@b.test",
		"a@[192.0.2.1]":      "a@[192.0.2.1]",
		"ü@example.test":     "ü@example.test", // not ASCII, which is allowed
		"a@bücher.example":   "a@bücher.example",
		// names and comments are not part of it
		`Name <a@b.test>`:           "a@b.test",
		`"Name, with comma" <a@b>`:  "a@b",
		`a@b.test (a comment)`:      "a@b.test",
		`"evil@x.test" <ok@a.test>`: "ok@a.test",
		// the quotes are put back
		`"a b"@example.test`:        `"a b"@example.test`,
		`"x@y.test"@example.test`:   `"x@y.test"@example.test`,
		`"a\"b"@example.test`:       `"a\"b"@example.test`,
		`"a\\b"@example.test`:       `"a\\b"@example.test`,
		`"a..b"@example.test`:       `"a..b"@example.test`,
		`".a"@example.test`:         `".a"@example.test`,
		`"a."@example.test`:         `"a."@example.test`,
		`"a(b)"@example.test`:       `"a(b)"@example.test`,
		`"a,b"@example.test`:        `"a,b"@example.test`,
		`Name <"a b"@example.test>`: `"a b"@example.test`,
	} {
		a, err := mail.ParseAddress(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}

		got := EnvelopeAddress(a)
		if got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}

		// what is made is read as the address which it was made from
		if back, err := mail.ParseAddress("<" + got + ">"); err != nil || back.Address != a.Address {
			t.Errorf("%q: %q is read as %v (%v), not %q", in, got, back, err, a.Address)
		}
	}

	if got := EnvelopeAddress(nil); got != "" {
		t.Errorf("nil: %q", got)
	}
	if got := EnvelopeAddress(&mail.Address{}); got != "" {
		t.Errorf("an empty address: %q", got)
	}
	if got := EnvelopeAddress(&mail.Address{Address: "postmaster"}); got != "postmaster" {
		t.Errorf("no domain: %q", got)
	}
}

// For any address which is read, what is made has no line break and is read as the same address.
func FuzzEnvelopeAddress(f *testing.F) {
	for _, s := range []string{`a@b.test`, `"a b"@c.test`, `"x@y"@z`, "Name <a@b>", `"a\"b"@c`, `ü@bücher.example`, `"a..b"@c`, `a@[1.2.3.4]`, `"\\"@c`, "<>"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		a, err := mail.ParseAddress(s)
		if err != nil {
			return
		}

		got := EnvelopeAddress(a)
		if strings.ContainsAny(got, "\r\n<>") && !strings.ContainsAny(a.Address, "<>") {
			t.Fatalf("%q: %q has a line break or a bracket", s, got)
		}

		back, err := mail.ParseAddress("<" + got + ">")
		if err != nil || back.Address != a.Address {
			t.Fatalf("%q: %q is read as %v (%v), not %q", s, got, back, err, a.Address)
		}
	})
}
