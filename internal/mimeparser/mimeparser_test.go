package mimeparser

import (
	"strings"
	"testing"
)

func message(disposition string) string {
	return "From: a@example.com\r\nTo: b@example.com\r\nSubject: x\r\nMIME-Version: 1.0\r\n" +
		"Content-Type: multipart/mixed; boundary=B\r\n\r\n--B\r\nContent-Type: text/plain\r\n\r\nhi\r\n" +
		"--B\r\nContent-Type: application/pdf\r\nContent-Disposition: " + disposition + "\r\n" +
		"Content-Transfer-Encoding: base64\r\n\r\nSGVsbG8=\r\n--B--\r\n"
}

func TestAttachmentFileNames(t *testing.T) {
	tests := map[string]struct {
		disposition string
		want        string
	}{
		"iso-8859-1 exploit case":  {`attachment; filename*=iso-8859-1''Cl%E9a.pdf`, "Cléa.pdf"},
		"utf-8 control":            {`attachment; filename*=utf-8''Cl%C3%A9a.pdf`, "Cléa.pdf"},
		"windows-1252":             {`attachment; filename*=windows-1252''%80uro.pdf`, "€uro.pdf"},
		"with language":            {`attachment; filename*=iso-8859-1'en'Cl%E9a.pdf`, "Cléa.pdf"},
		"continuations":            {`attachment; filename*0*=iso-8859-1''Cl%E9; filename*1*=a%2Epdf`, "Cléa.pdf"},
		"continuation plain part":  {`attachment; filename*0*=iso-8859-1''Cl%E9; filename*1=a.pdf`, "Cléa.pdf"},
		"unknown charset":          {`attachment; filename*=x-unknown''Clea%20b.pdf`, "Clea b.pdf"},
		"plain filename unchanged": {`attachment; filename="plain.pdf"`, "plain.pdf"},
		"extended beats plain":     {`attachment; filename="fallback.pdf"; filename*=iso-8859-1''Cl%E9a.pdf`, "Cléa.pdf"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env, err := New().ReadEnvelope(strings.NewReader(message(tc.disposition)))
			if err != nil {
				t.Fatal(err)
			}
			if len(env.Attachments) != 1 {
				t.Fatalf("expected 1 attachment, got %d", len(env.Attachments))
			}
			if got := env.Attachments[0].FileName; got != tc.want {
				t.Errorf("FileName = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestParseMediaTypeNilParams covers media types that mediatype.Parse cannot parse, for which
// it returns a nil params map and nil error. Writing decoded extended parameters into that map
// must not panic.
func TestParseMediaTypeNilParams(t *testing.T) {
	tests := map[string]struct {
		ctype     string
		wantType  string
		wantParam string // expected value of parameter "a"
	}{
		"nbsp media type exploit case":  {"\u00a0 ;a*=iso-8859-1''%E9", "", "é"},
		"invalid token exploit case":    {"@@ ;a*=iso-8859-1''%E9", "", "é"},
		"valid media type control":      {"text/plain ;a*=iso-8859-1''%E9", "text/plain", "é"},
		"empty media type control":      {";a*=iso-8859-1''%E9", "", "é"},
		"nbsp without extended control": {"\u00a0 ;a=b", "", ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			mtype, params, _, err := ParseMediaType(tc.ctype)
			if err != nil {
				t.Fatal(err)
			}
			if mtype != tc.wantType {
				t.Errorf("media type = %q, want %q", mtype, tc.wantType)
			}
			if got := params["a"]; got != tc.wantParam {
				t.Errorf("param a = %q, want %q", got, tc.wantParam)
			}
		})
	}
}

// TestReadEnvelopeInvalidMediaType parses a full message with an unparseable part media type
// carrying an RFC 2231 parameter. This previously panicked, crashing the SMTP server.
func TestReadEnvelopeInvalidMediaType(t *testing.T) {
	msg := func(ctype string) string {
		return "From: a@example.com\r\nTo: b@example.com\r\nSubject: x\r\nMIME-Version: 1.0\r\n" +
			"Content-Type: multipart/mixed; boundary=B\r\n\r\n--B\r\nContent-Type: " + ctype +
			"\r\n\r\nhi\r\n--B--\r\n"
	}

	t.Run("exploit case", func(t *testing.T) {
		env, err := New().ReadEnvelope(strings.NewReader(msg("\u00a0 ;a*=iso-8859-1''%E9")))
		if err != nil {
			t.Fatal(err)
		}
		// enmime treats a part with an empty content type as an "other" part
		if len(env.OtherParts) != 1 {
			t.Fatalf("expected 1 other part, got %d", len(env.OtherParts))
		}
		if got := string(env.OtherParts[0].Content); got != "hi" {
			t.Errorf("Content = %q, want %q", got, "hi")
		}
	})

	t.Run("control", func(t *testing.T) {
		env, err := New().ReadEnvelope(strings.NewReader(msg("text/plain ;a*=iso-8859-1''%E9")))
		if err != nil {
			t.Fatal(err)
		}
		if env.Text != "hi" {
			t.Errorf("Text = %q, want %q", env.Text, "hi")
		}
	})
}
