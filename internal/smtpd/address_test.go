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
