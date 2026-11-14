package smtpd

import (
	"bufio"
	"bytes"
	"net"
	"net/mail"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
)

func TestMain(m *testing.M) {
	logger.NoLogging = true

	dir, err := os.MkdirTemp("", "smtpd-test")
	if err != nil {
		panic(err)
	}
	config.Database = dir + "/mailpit.db"
	if err := storage.InitDB(); err != nil {
		panic(err)
	}

	code := m.Run()

	storage.Close()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func strictRFC(t *testing.T, strict bool) {
	t.Helper()

	orig := config.SMTPStrictRFCHeaders
	config.SMTPStrictRFCHeaders = strict
	t.Cleanup(func() { config.SMTPStrictRFCHeaders = orig })
}

func save(t *testing.T, data string) (string, error) {
	t.Helper()

	return SaveToDatabase(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1}, "a@b.test", []string{"c@d.test"}, []byte(data), nil)
}

func stored(t *testing.T, id string) string {
	t.Helper()

	raw, err := storage.GetMessageRaw(id)
	if err != nil {
		t.Fatal(err)
	}

	return string(raw)
}

// headerBlock returns the part of a message before the first empty line.
func headerBlock(raw string) string {
	head, _, _ := strings.Cut(raw, "\r\n\r\n")

	return head
}

func hasOddBreak(s string) bool {
	for i := 0; i < len(s); i++ {
		if (s[i] == '\r' && (i+1 >= len(s) || s[i+1] != '\n')) || (s[i] == '\n' && (i == 0 || s[i-1] != '\r')) {
			return true
		}
	}

	return false
}

// crHidingBcc has a Bcc header which is hidden inside the Subject by a CR.
const crHidingBcc = "From: a@b.test\r\nSubject: one\rBcc: hidden@x.test\r\nTo: c@d.test\r\n\r\nbody\r\n"

var lineBreakMessages = []struct{ name, data string }{
	{"CR CR LF", "From: a@b.test\r\nSubject: one\r\r\nTo: c@d.test\r\n\r\nbody\r\n"},
	{"LF", "From: a@b.test\nSubject: one\nTo: c@d.test\n\nbody\n"},
	{"CR before another header", "From: a@b.test\r\nSubject: one\rX-Two: two\r\nTo: c@d.test\r\n\r\nbody\r\n"},
	{"CR before a folded line", "From: a@b.test\r\nSubject: one\r two\r\nTo: c@d.test\r\n\r\nbody\r\n"},
	{"CR hiding a header", crHidingBcc},
	{"CR only", "From: a@b.test\rSubject: one\rTo: c@d.test\r\rbody\r"},
}

// The option used to return no error at all: strict mode only turned the fix for CR CR LF off.
func TestStrictRFCHeadersRejectsOtherLineBreaks(t *testing.T) {
	strictRFC(t, true)

	for _, tt := range lineBreakMessages {
		t.Run(tt.name, func(t *testing.T) {
			before := storage.CountTotal()

			id, err := save(t, tt.data)
			if err == nil || !strings.HasPrefix(err.Error(), "550 5.6.0 ") {
				t.Fatalf("id %q, error %v, want a 550 error", id, err)
			}
			if storage.CountTotal() != before {
				t.Error("the message was stored")
			}
		})
	}

	// the response the SMTP client gets is the error, as it is
	if !checkErrFormatRE.MatchString("550 5.6.0 Message headers must use <CR><LF> line breaks") {
		t.Error("the error is not sent as an SMTP response")
	}

	// controls: the same messages with <CR><LF>, and line breaks in the body only, are accepted
	for _, data := range []string{
		"From: a@b.test\r\nSubject: one\r\nTo: c@d.test\r\n\r\nbody\r\n",
		"From: a@b.test\r\nSubject: one\r\nTo: c@d.test\r\n\r\nbare\rCR\nand\nLF\r\r\nafter\r\n",
	} {
		if id, err := save(t, data); err != nil || id == "" {
			t.Errorf("a message with valid headers was rejected: %v: %q", err, data)
		}
	}
}

// By default the line breaks of the headers are fixed, and the body is as it was sent.
func TestDefaultFixesHeaderLineBreaks(t *testing.T) {
	strictRFC(t, false)

	for _, tt := range lineBreakMessages {
		t.Run(tt.name, func(t *testing.T) {
			id, err := save(t, tt.data)
			if err != nil || id == "" {
				t.Fatalf("id %q, error %v", id, err)
			}

			raw := stored(t, id)
			if hasOddBreak(headerBlock(raw)) {
				t.Errorf("the headers still have a line break which is not CRLF: %q", raw)
			}
			if _, err := mail.ReadMessage(strings.NewReader(raw)); err != nil {
				t.Errorf("the stored message does not parse: %v", err)
			}
		})
	}

	// The Return-Path is only rewritten (which also fixes the line breaks) when it is not the sender,
	// and the Bcc header only when a recipient is not in the headers, so a message which has the right
	// Return-Path, a Message-ID and every recipient has to be fixed when it is received
	for _, data := range []string{
		"Return-Path: <a@b.test>\nMessage-ID: <x@y.test>\nFrom: a@b.test\nTo: c@d.test\nSubject: one\n\nbody\n",
		"Return-Path: <a@b.test>\rMessage-ID: <x@y.test>\rFrom: a@b.test\rTo: c@d.test\rSubject: one\r\rbody\r",
		"Return-Path: <a@b.test>\r\nMessage-ID: <x@y.test>\r\nFrom: a@b.test\r\nTo: c@d.test\r\nSubject: one\rX-Hidden: yes\r\n\r\nbody\r\n",
	} {
		id, err := save(t, data)
		if err != nil || id == "" {
			t.Fatalf("id %q, error %v for %q", id, err, data)
		}
		if raw := stored(t, id); hasOddBreak(headerBlock(raw)) {
			t.Errorf("the headers still have a line break which is not CRLF: %q", raw)
		}
	}

	// a header hidden by a CR is the header it is to other software
	id, _ := save(t, crHidingBcc)
	m, err := mail.ReadMessage(strings.NewReader(stored(t, id)))
	if err != nil {
		t.Fatal(err)
	}
	if m.Header.Get("Subject") != "one" || !strings.Contains(m.Header.Get("Bcc"), "hidden@x.test") {
		t.Errorf("Subject %q, Bcc %q", m.Header.Get("Subject"), m.Header.Get("Bcc"))
	}

	// a CR is a line break, so what follows it must be a header, as it must after a LF. Otherwise
	// the message is refused, as it is with a LF there (which is how it already was).
	for _, data := range []string{
		"From: a@b.test\r\nSubject: one\rtwo\r\n\r\nbody\r\n",
		"From: a@b.test\r\nSubject: one\ntwo\r\n\r\nbody\r\n",
	} {
		if id, err := save(t, data); err == nil || !strings.Contains(err.Error(), "malformed header") {
			t.Errorf("id %q, error %v: want a malformed header error for %q", id, err, data)
		}
	}

	// the body is not changed
	body := "bare\rCR\nand\nLF\r\nafter\r\n"
	id, err = save(t, "From: a@b.test\nSubject: one\n\n"+body)
	if err != nil {
		t.Fatal(err)
	}
	if raw := stored(t, id); !strings.HasSuffix(raw, "\r\n\r\n"+body) {
		t.Errorf("the body changed: %q", raw)
	}

	// control: a message which is fine is stored as it is
	good := "From: a@b.test\r\nSubject: one\r\nTo: c@d.test\r\n\r\nbody\r\n"
	id, _ = save(t, good)
	if raw := stored(t, id); !strings.HasSuffix(raw, "\r\nSubject: one\r\nTo: c@d.test\r\n\r\nbody\r\n") {
		t.Errorf("a good message was changed: %q", raw)
	}
}

// fakeUpstream is an SMTP server which keeps the message sent to it.
func fakeUpstream(t *testing.T) (string, int, <-chan []byte) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	got := make(chan []byte, 4)

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				_ = c.SetDeadline(time.Now().Add(10 * time.Second))
				r := bufio.NewReader(c)
				write := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }

				write("220 fake ESMTP")
				for {
					line, err := r.ReadString('\n')
					if err != nil {
						return
					}
					switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
					case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
						write("250 fake")
					case strings.HasPrefix(cmd, "MAIL"), strings.HasPrefix(cmd, "RCPT"):
						write("250 ok")
					case cmd == "DATA":
						write("354 go ahead")
						var data bytes.Buffer
						for {
							l, err := r.ReadBytes('\n')
							if err != nil {
								return
							}
							if bytes.Equal(l, []byte(".\r\n")) {
								break
							}
							data.Write(l)
						}
						got <- data.Bytes()
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

	addr := ln.Addr().(*net.TCPAddr)

	return "127.0.0.1", addr.Port, got
}

// A bare CR in the headers was sent to the server as it was, where it may be a line break to that
// server. It was sent however the message was stored, as nothing was changed when no header was
// rewritten. The line breaks of the body are the SMTP client library's (it sends a LF as CRLF),
// and are not changed here.
func TestRelayAndForwardSendCRLFHeaderLineBreaks(t *testing.T) {
	msg := "From: a@b.test\rSubject: one\nBcc: hidden@x.test\r\r\nTo: c@d.test\r\n\r\nbody\rwith\nbreaks\r\n"

	check := func(t *testing.T, sent <-chan []byte) {
		t.Helper()

		select {
		case data := <-sent:
			head, body, _ := strings.Cut(string(data), "\r\n\r\n")
			if hasOddBreak(head) {
				t.Errorf("the headers sent have a line break which is not CRLF: %q", data)
			}
			if body != "body\rwith\r\nbreaks\r\n" {
				t.Errorf("the body sent changed: %q", body)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("nothing was sent")
		}
	}

	t.Run("relay", func(t *testing.T) {
		host, port, sent := fakeUpstream(t)
		orig := config.SMTPRelayConfig
		config.SMTPRelayConfig = config.SMTPRelayConfigStruct{Host: host, Port: port}
		t.Cleanup(func() { config.SMTPRelayConfig = orig })

		if err := Relay("a@b.test", []string{"c@d.test"}, []byte(msg)); err != nil {
			t.Fatal(err)
		}
		check(t, sent)
	})

	t.Run("forward", func(t *testing.T) {
		host, port, sent := fakeUpstream(t)
		orig := config.SMTPForwardConfig
		config.SMTPForwardConfig = config.SMTPForwardConfigStruct{Host: host, Port: port, To: "c@d.test"}
		t.Cleanup(func() { config.SMTPForwardConfig = orig })

		if err := forward("a@b.test", []byte(msg)); err != nil {
			t.Fatal(err)
		}
		check(t, sent)
	})

	// control: a message which is fine is sent as it is
	t.Run("control: relay of a good message", func(t *testing.T) {
		host, port, sent := fakeUpstream(t)
		orig := config.SMTPRelayConfig
		config.SMTPRelayConfig = config.SMTPRelayConfigStruct{Host: host, Port: port}
		t.Cleanup(func() { config.SMTPRelayConfig = orig })

		good := "From: a@b.test\r\nSubject: one\r\n\r\nbody\r\n"
		if err := Relay("a@b.test", []string{"c@d.test"}, []byte(good)); err != nil {
			t.Fatal(err)
		}
		select {
		case data := <-sent:
			if string(data) != good {
				t.Errorf("a good message was changed: %q", data)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("nothing was sent")
		}
	})
}
