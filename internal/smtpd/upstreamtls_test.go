package smtpd

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/snakeoil"
)

// upstreamName is not a localhost name, as net/smtp sends PLAIN credentials to a localhost name over any connection.
const upstreamName = "mail.example.test"

// tlsUpstream is an SMTP server which offers AUTH PLAIN. With implicitTLS the connection is TLS from the start,
// otherwise STARTTLS is offered. If silentAfterGreeting is set it says nothing after the greeting. It returns the
// address, and a counter of the AUTH commands it accepted.
func tlsUpstream(t *testing.T, implicitTLS, silentAfterGreeting bool) (string, *atomic.Int32) {
	t.Helper()

	cert, err := tls.LoadX509KeyPair(snakeoil.Public("sans:"+upstreamName), snakeoil.Private("sans:"+upstreamName))
	if err != nil {
		t.Fatal(err)
	}
	tlsConf := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	auths := &atomic.Int32{}

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()

				encrypted := implicitTLS
				if implicitTLS {
					c = tls.Server(c, tlsConf)
				}

				r := bufio.NewReader(c)
				write := func(s string) { _, _ = fmt.Fprintf(c, "%s\r\n", s) }

				write("220 " + upstreamName)

				if silentAfterGreeting {
					for {
						if _, err := r.ReadString('\n'); err != nil {
							return
						}
					}
				}

				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}

					switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
					case strings.HasPrefix(cmd, "EHLO"):
						if encrypted {
							write("250-" + upstreamName)
							write("250 AUTH PLAIN")
						} else {
							write("250-" + upstreamName)
							write("250-AUTH PLAIN")
							write("250 STARTTLS")
						}
					case cmd == "STARTTLS":
						write("220 go ahead")
						c = tls.Server(c, tlsConf)
						r = bufio.NewReader(c)
						encrypted = true
					case strings.HasPrefix(cmd, "AUTH PLAIN"):
						auths.Add(1)
						write("235 ok")
					case cmd == "QUIT":
						write("221 bye")
						return
					default:
						write("250 ok")
					}
				}
			}(c)
		}
	}()

	return ln.Addr().String(), auths
}

// clientTLSConf is what relaying and forwarding use, with allow-insecure (the certificate is self-signed).
func clientTLSConf() *tls.Config {
	return &tls.Config{ServerName: upstreamName, InsecureSkipVerify: true} // #nosec
}

// PLAIN authentication over a connection which is TLS from the start sends the credentials. The SMTP client used to be
// given the connection wrapped for its time limits, so it did not know the connection was encrypted and refused to
// send them ("unencrypted connection") to any server which is not localhost.
func TestUpstreamImplicitTLSPlainAuth(t *testing.T) {
	addr, auths := tlsUpstream(t, true, false)

	client, err := dialSMTP(addr, upstreamName, true, clientTLSConf())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	if _, ok := client.TLSConnectionState(); !ok {
		t.Error("the SMTP client does not know the connection is TLS")
	}

	if err := client.Hello("client"); err != nil {
		t.Fatal(err)
	}

	if err := client.Auth(smtp.PlainAuth("", "user", "pass", upstreamName)); err != nil {
		t.Fatalf("AUTH PLAIN over implicit TLS: %v", err)
	}

	if n := auths.Load(); n != 1 {
		t.Errorf("the server accepted %d AUTH commands, want 1", n)
	}
}

// Control: PLAIN authentication after STARTTLS sends the credentials (the SMTP client marks the connection as TLS
// itself), and over a connection which is not encrypted they are still refused, so the test above can tell the two
// apart.
func TestUpstreamPlainAuthNeedsTLS(t *testing.T) {
	t.Run("starttls", func(t *testing.T) {
		addr, auths := tlsUpstream(t, false, false)

		client, err := dialSMTP(addr, upstreamName, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = client.Close() }()

		if err := client.Hello("client"); err != nil {
			t.Fatal(err)
		}
		if err := client.StartTLS(clientTLSConf()); err != nil {
			t.Fatal(err)
		}
		if err := client.Auth(smtp.PlainAuth("", "user", "pass", upstreamName)); err != nil {
			t.Fatalf("AUTH PLAIN after STARTTLS: %v", err)
		}
		if n := auths.Load(); n != 1 {
			t.Errorf("the server accepted %d AUTH commands, want 1", n)
		}
	})

	t.Run("unencrypted", func(t *testing.T) {
		addr, auths := tlsUpstream(t, false, false)

		client, err := dialSMTP(addr, upstreamName, false, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = client.Close() }()

		if err := client.Hello("client"); err != nil {
			t.Fatal(err)
		}
		if err := client.Auth(smtp.PlainAuth("", "user", "pass", upstreamName)); err == nil {
			t.Error("AUTH PLAIN over an unencrypted connection was sent")
		}
		if n := auths.Load(); n != 0 {
			t.Errorf("the server accepted %d AUTH commands, want 0", n)
		}
	})
}

// The idle time limit still applies beneath TLS: a server which completes the handshake and greets, then says nothing,
// is given up on after the idle time.
func TestUpstreamSilentAfterTLSHandshakeTimesOut(t *testing.T) {
	withUpstreamTimeouts(t, time.Second, 300*time.Millisecond)
	addr, _ := tlsUpstream(t, true, true)

	client, err := dialSMTP(addr, upstreamName, true, clientTLSConf())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	done := make(chan error, 1)
	start := time.Now()
	go func() { done <- client.Hello("client") }()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "timeout") {
			t.Errorf("got %v, want a timeout error", err)
		}
		if took := time.Since(start); took > 3*time.Second {
			t.Errorf("took %v, the idle time is 300ms", took)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("still blocked, the idle time is 300ms")
	}
}
