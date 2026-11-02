package smtpd

import (
	"net/mail"
	"strings"
	"testing"
)

func TestScanAddressesInHeaderCharsets(t *testing.T) {
	names := map[string]string{
		"utf-8":        "=?UTF-8?B?5bGx55Sw5aSq6YOO?=",
		"iso-2022-jp":  "=?ISO-2022-JP?B?GyRCOzNFREJATzobKEI=?=",
		"shift_jis":    "=?Shift_JIS?B?jlKTY5G+mFk=?=",
		"euc-jp":       "=?EUC-JP?B?u7PFxMLAz7o=?=",
		"windows-1252": "=?windows-1252?Q?Andr=E9?=",
	}

	for charset, name := range names {
		t.Run(charset, func(t *testing.T) {
			raw := "To: " + name + " <to@example.com>\r\n" +
				"Cc: " + name + " <cc@example.com>\r\n" +
				"\r\nbody\r\n"
			msg, err := mail.ReadMessage(strings.NewReader(raw))
			if err != nil {
				t.Fatal(err)
			}

			emails, hasBcc := scanAddressesInHeader(msg.Header)
			if hasBcc {
				t.Error("expected no Bcc header")
			}
			for _, addr := range []string{"to@example.com", "cc@example.com"} {
				if !emails[addr] {
					t.Errorf("%s not found in headers", addr)
				}
			}
		})
	}
}

func TestAddBccAddresses(t *testing.T) {
	tests := map[string]struct {
		raw  string
		want string
	}{
		// exploit: an existing Bcc header must be kept and the missing addresses appended
		"existing Bcc": {
			raw:  "To: to@example.com\r\nBcc: existing@example.com\r\n\r\nbody\r\n",
			want: "existing@example.com, hidden1@example.com, hidden2@example.com",
		},
		// control: no existing Bcc header
		"no Bcc": {
			raw:  "To: to@example.com\r\n\r\nbody\r\n",
			want: "hidden1@example.com, hidden2@example.com",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			msg, err := mail.ReadMessage(strings.NewReader(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			_, hasBcc := scanAddressesInHeader(msg.Header)

			out, err := addBccAddresses([]byte(tc.raw), msg.Header, hasBcc, []string{"hidden1@example.com", "hidden2@example.com"})
			if err != nil {
				t.Fatal(err)
			}

			parsed, err := mail.ReadMessage(strings.NewReader(string(out)))
			if err != nil {
				t.Fatal(err)
			}
			if got := parsed.Header.Get("Bcc"); got != tc.want {
				t.Errorf("Bcc = %q, want %q", got, tc.want)
			}
		})
	}
}
