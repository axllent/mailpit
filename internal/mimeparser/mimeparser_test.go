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
