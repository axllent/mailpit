package smtpd

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// withUpstreamTimeouts sets the time limits for relaying and forwarding for the length of the test.
func withUpstreamTimeouts(t *testing.T, dial, idle time.Duration) {
	t.Helper()

	origDial, origIdle := relayDialTimeout, relayIdleTimeout
	relayDialTimeout, relayIdleTimeout = dial, idle
	t.Cleanup(func() { relayDialTimeout, relayIdleTimeout = origDial, origIdle })
}

// upstream is an SMTP server which accepts a message. It waits for delay before each response (and does not answer
// at all if silent is set, not even with the greeting).
func upstream(t *testing.T, delay time.Duration, silent bool) (string, int) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				if silent {
					// never answers (whatever it is sent), until the client gives up
					_, _ = io.Copy(io.Discard, c)
					return
				}

				write := func(s string) {
					time.Sleep(delay)
					_, _ = c.Write([]byte(s + "\r\n"))
				}
				r := bufio.NewReader(c)
				write("220 fake")
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
					case cmd == "DATA":
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
						write("250 queued")
					case cmd == "QUIT":
						write("221 bye")
						return
					default:
						write("250 ok")
					}
				}
			}()
		}
	}()

	return "127.0.0.1", ln.Addr().(*net.TCPAddr).Port
}

// send relays and forwards (the two are separate code) a message to the host, and returns what each one returned.
func sendUpstream(t *testing.T, host string, port int, tlsConn bool) map[string]error {
	t.Helper()

	results := map[string]error{}

	origRelay, origForward := config.SMTPRelayConfig, config.SMTPForwardConfig
	t.Cleanup(func() { config.SMTPRelayConfig, config.SMTPForwardConfig = origRelay, origForward })

	config.SMTPRelayConfig = config.SMTPRelayConfigStruct{Host: host, Port: port, TLS: tlsConn, AllowInsecure: true}
	config.SMTPForwardConfig = config.SMTPForwardConfigStruct{Host: host, Port: port, TLS: tlsConn, AllowInsecure: true, To: "c@d.test"}

	msg := []byte("From: a@b.test\r\nSubject: s\r\n\r\nbody\r\n")

	for name, send := range map[string]func() error{
		"relay":   func() error { return Relay("a@b.test", []string{"c@d.test"}, msg) },
		"forward": func() error { return forward("a@b.test", msg) },
	} {
		done := make(chan error, 1)
		go func() { done <- send() }()

		select {
		case results[name] = <-done:
		case <-time.After(8 * time.Second):
			results[name] = errStillBlocked
		}
	}

	return results
}

type blockedError struct{}

func (blockedError) Error() string { return "still blocked" }

var errStillBlocked error = blockedError{}

// A server which accepts the connection and says nothing is given up on after the idle time. Relaying and forwarding
// used to wait for ever, which held the session of the sender (the message is relayed before it is answered).
func TestUpstreamSilentServerTimesOut(t *testing.T) {
	withUpstreamTimeouts(t, time.Second, 300*time.Millisecond)
	host, port := upstream(t, 0, true)

	start := time.Now()
	for name, err := range sendUpstream(t, host, port, false) {
		if err == nil || err == errStillBlocked || !strings.Contains(err.Error(), "timeout") {
			t.Errorf("%s: got %v, want a timeout error", name, err)
		}
	}
	if took := time.Since(start); took > 4*time.Second {
		t.Errorf("both took %v, the idle time is 300ms", took)
	}
}

// The handshake of a TLS connection is within the dial time as well.
func TestUpstreamSilentTLSServerTimesOut(t *testing.T) {
	withUpstreamTimeouts(t, 300*time.Millisecond, time.Minute)
	host, port := upstream(t, 0, true)

	start := time.Now()
	for name, err := range sendUpstream(t, host, port, true) {
		if err == nil || err == errStillBlocked || !strings.Contains(err.Error(), "TLS dial error") {
			t.Errorf("%s: got %v, want a TLS dial error", name, err)
		}
	}
	if took := time.Since(start); took > 4*time.Second {
		t.Errorf("both took %v, the dial time is 300ms", took)
	}
}

// Control: a server which answers is used as before, and one which is slow to answer but does (each response
// takes longer than half the idle time, and the whole takes more than the idle time) is waited for. The idle time is
// for a pause, not for how long the whole takes, so a large message to a slow server still goes.
func TestUpstreamSlowServerIsWaitedFor(t *testing.T) {
	withUpstreamTimeouts(t, time.Second, 500*time.Millisecond)

	for name, delay := range map[string]time.Duration{"fast": 0, "slow": 250 * time.Millisecond} {
		t.Run(name, func(t *testing.T) {
			host, port := upstream(t, delay, false)

			start := time.Now()
			for name, err := range sendUpstream(t, host, port, false) {
				if err != nil {
					t.Errorf("%s: %v", name, err)
				}
			}
			if delay > 0 && time.Since(start) < 500*time.Millisecond {
				t.Errorf("took %v, the test is not slow enough to show anything", time.Since(start))
			}
		})
	}
}
