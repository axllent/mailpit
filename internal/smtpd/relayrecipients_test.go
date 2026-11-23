package smtpd

import (
	"bufio"
	"net"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// rcptUpstream is an SMTP server which accepts one message and reports the RCPT TO arguments it was sent.
func rcptUpstream(t *testing.T) (string, int, <-chan []string) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ch := make(chan []string, 1)

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()

		write := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
		r := bufio.NewReader(c)
		rcpts := []string{}

		write("220 fake")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			switch cmd := strings.TrimSpace(line); {
			case strings.HasPrefix(strings.ToUpper(cmd), "RCPT TO:"):
				rcpts = append(rcpts, cmd[len("RCPT TO:"):])
				write("250 ok")
			case strings.EqualFold(cmd, "DATA"):
				write("354 go ahead")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
				}
				ch <- rcpts
				write("250 queued")
			case strings.EqualFold(cmd, "QUIT"):
				write("221 bye")
				return
			default:
				write("250 ok")
			}
		}
	}()

	return "127.0.0.1", ln.Addr().(*net.TCPAddr).Port, ch
}

// The block list and the matching expression are matched against the address as it is parsed, and that address is
// what is sent to the relay server, however the recipient was written.
func TestAutoRelayRecipientForms(t *testing.T) {
	const msg = "From: a@b.test\r\nSubject: s\r\n\r\nbody\r\n"

	const allowed = "other@allowed.example"

	tests := []struct {
		name     string
		blocked  string
		matching string // used instead of relay-all when set
		to       []string
		want     []string // RCPT TO arguments the relay server must get, nil for no relay at all
	}{
		// controls: the plain address is blocked, the other is sent
		{"plain blocked", `@blocked\.example$`, "", []string{"victim@blocked.example", allowed}, []string{"<" + allowed + ">"}},
		{"plain blocked by address", `^victim@blocked\.example$`, "", []string{"victim@blocked.example", allowed}, []string{"<" + allowed + ">"}},
		{"plain only blocked", `@blocked\.example$`, "", []string{"victim@blocked.example"}, nil},
		// the same address written another way is blocked just the same
		{"comment", `@blocked\.example$`, "", []string{"victim@blocked.example(x)", allowed}, []string{"<" + allowed + ">"}},
		{"group", `@blocked\.example$`, "", []string{"g:victim@blocked.example;", allowed}, []string{"<" + allowed + ">"}},
		{"quoted local part", `^victim@blocked\.example$`, "", []string{`"victim"@blocked.example`, allowed}, []string{"<" + allowed + ">"}},
		{"display name", `@blocked\.example$`, "", []string{"Some Name <victim@blocked.example>", allowed}, []string{"<" + allowed + ">"}},
		// what is not blocked is sent as the plain address
		{"comment sent plain", "", "", []string{"user@" + "allowed.example(x)"}, []string{"<user@allowed.example>"}},
		{"group sent plain", "", "", []string{"g:user@allowed.example;"}, []string{"<user@allowed.example>"}},
		{"display name sent plain", "", "", []string{"Some Name <user@allowed.example>"}, []string{"<user@allowed.example>"}},
		{"quoted local part sent quoted as needed", "", "", []string{`"a b"@allowed.example`}, []string{`<"a b"@allowed.example>`}},
		// a recipient which does not parse is not relayed
		{"invalid", "", "", []string{"not an address", allowed}, []string{"<" + allowed + ">"}},
		// matching is on the parsed address, not on text in a comment
		{"matching control", "", `@allowed\.example$`, []string{allowed, "x@other.example"}, []string{"<" + allowed + ">"}},
		{"matching comment text", "", `@allowed\.example`, []string{"victim@blocked.example(@allowed.example)"}, nil},
		{"matching with comment", "", `@allowed\.example$`, []string{"user@allowed.example(x)"}, []string{"<user@allowed.example>"}},
	}

	origRelay, origRelayAll, origMatching := config.SMTPRelayConfig, config.SMTPRelayAll, config.SMTPRelayMatchingRegexp
	t.Cleanup(func() {
		config.SMTPRelayConfig, config.SMTPRelayAll, config.SMTPRelayMatchingRegexp = origRelay, origRelayAll, origMatching
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			host, port, ch := rcptUpstream(t)

			config.SMTPRelayConfig = config.SMTPRelayConfigStruct{Host: host, Port: port}
			config.SMTPRelayAll = tc.matching == ""
			config.SMTPRelayMatchingRegexp = nil

			if tc.blocked != "" {
				config.SMTPRelayConfig.BlockedRecipientsRegexp = mustCompile(t, tc.blocked)
			}
			if tc.matching != "" {
				config.SMTPRelayMatchingRegexp = mustCompile(t, tc.matching)
			}

			data := []byte(msg)
			if err := autoRelayMessage("a@b.test", tc.to, &data); err != nil {
				t.Fatal(err)
			}

			if tc.want == nil {
				select {
				case got := <-ch:
					t.Fatalf("relayed to %v, want no relay", got)
				case <-time.After(200 * time.Millisecond):
				}
				return
			}

			select {
			case got := <-ch:
				if !reflect.DeepEqual(got, tc.want) {
					t.Errorf("RCPT TO = %v, want %v", got, tc.want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("relay server did not receive a message")
			}
		})
	}
}

func mustCompile(t *testing.T, expr string) *regexp.Regexp {
	t.Helper()

	re, err := regexp.Compile(expr)
	if err != nil {
		t.Fatal(err)
	}

	return re
}
