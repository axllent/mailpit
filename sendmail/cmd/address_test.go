package cmd

import (
	"bufio"
	"net"
	"net/mail"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/smtpd"
)

// socketPath returns the path for a Unix socket which is short enough to be used. t.TempDir() is not: it has the name
// of the test in it (and of the subtest), which with the temporary directory of macOS (about 50 bytes) is more than the
// 104 which a socket path can be there (108 on Linux), and listening fails with "bind: invalid argument".
func socketPath(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "mp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	path := filepath.Join(dir, "s.sock")
	if len(path) > 100 {
		t.Skipf("the temporary directory is too long for the path of a Unix socket (%d bytes): %s", len(path), path)
	}

	return path
}

// recorder is an SMTP server which accepts a message, and records the MAIL and RCPT commands it was sent.
type recorder struct {
	mu       sync.Mutex
	commands []string
	data     []byte // the message, as it was sent (before its end)
	after    string // the command which came after the end of the message
}

func (r *recorder) get() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.commands...)
}

func newRecorder(t *testing.T, network, addr string) (string, *recorder) {
	t.Helper()

	ln, err := net.Listen(network, addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	rec := &recorder{}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()
				_ = c.SetDeadline(time.Now().Add(5 * time.Second))
				w := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
				r := bufio.NewReader(c)
				w("220 fake")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					line := strings.TrimSpace(l)
					switch cmd := strings.ToUpper(line); {
					case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
						rec.mu.Lock()
						rec.commands = append(rec.commands, line)
						rec.mu.Unlock()
						w("250 ok")
					case cmd == "DATA":
						w("354 go")
						var data []byte
						for {
							d, err := r.ReadString('\n')
							if err != nil || d == ".\r\n" {
								break
							}
							data = append(data, d...)
						}
						rec.mu.Lock()
						rec.data = data
						rec.mu.Unlock()
						w("250 queued")
						// the command after it (one which is not QUIT would be what was in the message)
						if next, err := r.ReadString('\n'); err == nil {
							rec.mu.Lock()
							rec.after = strings.TrimSpace(next)
							rec.mu.Unlock()
							w("221 bye")
						}
						return
					case cmd == "QUIT":
						w("221 bye")
						return
					default:
						w("250 ok")
					}
				}
			}()
		}
	}()

	return ln.Addr().String(), rec
}

func headers(t *testing.T, h string) mail.Header {
	t.Helper()

	msg, err := mail.ReadMessage(strings.NewReader(h + "\r\n\r\nbody\r\n"))
	if err != nil {
		t.Fatal(err)
	}

	return msg.Header
}

// The recipients of the headers are as they are used in a command, which has the quotes of a local part which needs
// them (the address of the parser has none, so "a b"@example.test was a b@example.test, which a server refuses, and
// "x@y.test"@example.test was x@y.test@example.test, which it can read two ways).
func TestHeaderRecipients(t *testing.T) {
	got := HeaderRecipients(headers(t, `To: "a b"@example.test, Plain <p@example.test>, "x@y.test"@example.test
Cc: c@example.test
Bcc: Hidden <"h h"@example.test>`))

	want := []string{`"a b"@example.test`, "p@example.test", `"x@y.test"@example.test`, "c@example.test", `"h h"@example.test`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}

	// control: the usual addresses are the same as they were, and a header which is not valid (or not there) has none
	got = HeaderRecipients(headers(t, "To: a@example.test, B <b@example.test>\r\nCc: not an address"))
	if want := []string{"a@example.test", "b@example.test"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := HeaderRecipients(headers(t, "Subject: none")); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

// What is sent is the sender and the recipients as they are used in a command, by TCP and over a Unix socket. The
// sender was refused ("invalid from address") as it was parsed again after its quotes had gone.
func TestSendUsesQuotedAddresses(t *testing.T) {
	socket := socketPath(t)

	for name, newServer := range map[string]func() (string, *recorder){
		"tcp":  func() (string, *recorder) { return newRecorder(t, "tcp", "127.0.0.1:0") },
		"unix": func() (string, *recorder) { a, r := newRecorder(t, "unix", socket); return "unix:" + a, r },
	} {
		t.Run(name, func(t *testing.T) {
			addr, rec := newServer()

			from, err := mail.ParseAddress(`"odd user"@example.test`)
			if err != nil {
				t.Fatal(err)
			}
			_ = from

			rcpts := HeaderRecipients(headers(t, `To: "a b"@example.test, "x@y.test"@example.test, plain@example.test`))
			if err := Send(addr, `"odd user"@example.test`, rcpts, []byte("Subject: x\r\n\r\nbody\r\n")); err != nil {
				t.Fatal(err)
			}

			want := []string{
				`MAIL FROM:<"odd user"@example.test>`,
				`RCPT TO:<"a b"@example.test>`,
				`RCPT TO:<"x@y.test"@example.test>`,
				`RCPT TO:<plain@example.test>`,
			}
			if got := rec.get(); !reflect.DeepEqual(got, want) {
				t.Fatalf("got %q, want %q", got, want)
			}
		})
	}

	// control: the sender as an address and a name, or only an address, is what it was
	for in, want := range map[string]string{
		"sender@example.test":          "MAIL FROM:<sender@example.test>",
		"Name <sender@example.test>":   "MAIL FROM:<sender@example.test>",
		"<sender@example.test>":        "MAIL FROM:<sender@example.test>",
		`"Odd, Name" <o@example.test>`: "MAIL FROM:<o@example.test>",
	} {
		addr, rec := newRecorder(t, "tcp", "127.0.0.1:0")
		if err := Send(addr, in, []string{"to@example.test"}, []byte("Subject: x\r\n\r\nbody\r\n")); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got := rec.get(); len(got) == 0 || got[0] != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}

	// a sender which is not an address is still refused, as are line breaks
	for _, bad := range []string{"not an address", "a@b.test\r\nRCPT TO:<x@y.test>", ""} {
		if err := Send("127.0.0.1:1", bad, []string{"to@example.test"}, nil); err == nil {
			t.Errorf("a sender of %q was accepted", bad)
		}
	}
}

// The server which receives it (Mailpit's own) takes the addresses which were made: with the quotes a recipient or
// a sender is accepted, and without them (as the parser gives it) it was refused.
func TestMailpitAcceptsQuotedAddresses(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	type received struct {
		from string
		to   []string
	}
	got := make(chan received, 1)

	srv := &smtpd.Server{
		DisableReverseDNS: true,
		Hostname:          "test",
		AppName:           "test",
		MsgIDHandler: func(_ net.Addr, from string, to []string, _ []byte, _ *string) (string, error) {
			got <- received{from, to}
			return "", nil
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close(); _ = ln.Close() })

	rcpts := HeaderRecipients(headers(t, `To: "a b"@example.test, "x@y.test"@example.test`))
	if err := Send(ln.Addr().String(), `"odd user"@example.test`, rcpts, []byte("Subject: x\r\n\r\nbody\r\n")); err != nil {
		t.Fatalf("Mailpit refused it: %v", err)
	}

	select {
	case r := <-got:
		if r.from != `"odd user"@example.test` || !reflect.DeepEqual(r.to, []string{`"a b"@example.test`, `"x@y.test"@example.test`}) {
			t.Fatalf("Mailpit got from %q to %q", r.from, r.to)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("nothing was received")
	}

	// and what it was before is what it refused
	err = Send(ln.Addr().String(), "odd user@example.test", []string{"a b@example.test"}, []byte("Subject: x\r\n\r\nbody\r\n"))
	if err == nil {
		t.Fatal("an address which has lost its quotes was accepted")
	}
}

// The sender which is given with -f (or in the environment) is used as it is in a command, with the quotes of its local
// part, and one which is not an address is refused.
func TestEnvelopeSender(t *testing.T) {
	for in, want := range map[string]string{
		"sender@example.test":        "sender@example.test",
		"Name <sender@example.test>": "sender@example.test",
		`"odd user"@example.test`:    `"odd user"@example.test`,
		`Odd <"o o"@example.test>`:   `"o o"@example.test`,
	} {
		if got, err := envelopeSender(in); err != nil || got != want {
			t.Errorf("%q: %q, %v, want %q", in, got, err, want)
		}
	}

	for _, bad := range []string{"", "not an address", "odd user@example.test"} {
		if got, err := envelopeSender(bad); err == nil {
			t.Errorf("%q was accepted as %q", bad, got)
		}
	}
}
