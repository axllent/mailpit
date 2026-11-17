package smtpd

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// received is what a capturing upstream server was sent for one message.
type received struct {
	mailFrom string // the MAIL FROM argument, eg: <a@b.test>
	data     string // the message as sent after DATA
}

// capturingUpstream is an SMTP server which accepts one message and reports what it was sent.
func capturingUpstream(t *testing.T) (string, int, <-chan received) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ch := make(chan received, 1)

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()

		write := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
		r := bufio.NewReader(c)
		var got received

		write("220 fake")
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			switch cmd := strings.TrimSpace(line); {
			case strings.HasPrefix(strings.ToUpper(cmd), "MAIL FROM:"):
				got.mailFrom = cmd[len("MAIL FROM:"):]
				write("250 ok")
			case strings.EqualFold(cmd, "DATA"):
				write("354 go ahead")
				var b strings.Builder
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					if l == ".\r\n" {
						break
					}
					b.WriteString(l)
				}
				got.data = b.String()
				ch <- got
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

// Relaying (released and auto-relayed messages) and forwarding must handle return-path
// and override-from the same way.
func TestReturnPath(t *testing.T) {
	const msg = "Return-Path: <original@b.test>\r\nFrom: a@b.test\r\nSubject: s\r\n\r\nbody\r\n"

	tests := []struct {
		name           string
		returnPath     string
		overrideFrom   string
		wantMailFrom   string
		wantReturnPath string
	}{
		// control: without return-path, the original sender and header are kept
		{"not set", "", "", "<a@b.test>", "Return-Path: <original@b.test>"},
		{"set", "bounce@b.test", "", "<bounce@b.test>", "Return-Path: <bounce@b.test>"},
		{"override-from only", "", "from@b.test", "<from@b.test>", "Return-Path: <original@b.test>"},
		// return-path is the envelope sender even when override-from is also set
		{"with override-from", "bounce@b.test", "from@b.test", "<bounce@b.test>", "Return-Path: <bounce@b.test>"},
	}

	senders := map[string]func(host string, port int, returnPath, overrideFrom string) error{
		"forward": func(host string, port int, returnPath, overrideFrom string) error {
			config.SMTPForwardConfig = config.SMTPForwardConfigStruct{
				Host: host, Port: port, To: "c@d.test",
				ReturnPath: returnPath, OverrideFrom: overrideFrom,
			}
			return forward("a@b.test", []byte(msg))
		},
		// as used when releasing a message via the web UI or API
		"relay": func(host string, port int, returnPath, overrideFrom string) error {
			config.SMTPRelayConfig = config.SMTPRelayConfigStruct{
				Host: host, Port: port, ReturnPath: returnPath, OverrideFrom: overrideFrom,
			}
			return Relay("a@b.test", []string{"c@d.test"}, []byte(msg))
		},
		"auto-relay": func(host string, port int, returnPath, overrideFrom string) error {
			config.SMTPRelayConfig = config.SMTPRelayConfigStruct{
				Host: host, Port: port, ReturnPath: returnPath, OverrideFrom: overrideFrom,
			}
			config.SMTPRelayAll = true
			data := []byte(msg)
			return autoRelayMessage("a@b.test", []string{"c@d.test"}, &data)
		},
	}

	origForward, origRelay, origRelayAll := config.SMTPForwardConfig, config.SMTPRelayConfig, config.SMTPRelayAll
	t.Cleanup(func() {
		config.SMTPForwardConfig, config.SMTPRelayConfig, config.SMTPRelayAll = origForward, origRelay, origRelayAll
	})

	for name, send := range senders {
		for _, tc := range tests {
			t.Run(name+" "+tc.name, func(t *testing.T) {
				host, port, ch := capturingUpstream(t)

				if err := send(host, port, tc.returnPath, tc.overrideFrom); err != nil {
					t.Fatalf("%s: %v", name, err)
				}

				checkReceived(t, ch, tc.wantMailFrom, tc.wantReturnPath)
			})
		}
	}
}

func checkReceived(t *testing.T, ch <-chan received, wantMailFrom, wantReturnPath string) {
	t.Helper()

	var got received
	select {
	case got = <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream did not receive a message")
	}

	if got.mailFrom != wantMailFrom {
		t.Errorf("MAIL FROM = %q, want %q", got.mailFrom, wantMailFrom)
	}
	if n := strings.Count(got.data, "Return-Path:"); n != 1 {
		t.Errorf("message has %d Return-Path headers, want 1:\n%s", n, got.data)
	}
	if !strings.Contains(got.data, wantReturnPath+"\r\n") {
		t.Errorf("message does not contain %q:\n%s", wantReturnPath, got.data)
	}
}
