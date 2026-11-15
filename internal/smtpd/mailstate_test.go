package smtpd

import (
	"net"
	"sync"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/smtpd/chaos"
)

// senderRecorder is a handler which keeps the sender of each message.
type senderRecorder struct {
	mu      sync.Mutex
	senders []string
}

func (r *senderRecorder) handler(_ net.Addr, from string, _ []string, _ []byte, _ *string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.senders = append(r.senders, from)

	return "", nil
}

func (r *senderRecorder) get() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.senders...)
}

// A MAIL command which is refused ends the transaction: there is no sender, so RCPT and DATA are not allowed. They
// used to go on with the sender of the MAIL before it, and the message was stored with that one.
func TestRefusedMailEndsTheTransaction(t *testing.T) {
	for _, tc := range []struct {
		name string
		mail string // the MAIL command which is refused
		code string
	}{
		{"not an address", "MAIL FROM:garbage", "501"},
		{"address which is not valid", "MAIL FROM:<a b@c.test>", "553"},
		{"AUTH which is not valid", "MAIL FROM:<second@b.test> AUTH=", "501"},
		{"SIZE which is not a number", "MAIL FROM:<second@b.test> SIZE=zzz", "501"},
		{"SIZE which is too large", "MAIL FROM:<second@b.test> SIZE=99999999999", "552"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &senderRecorder{}
			addr, _ := limitServer(t, nil, rec.handler)
			cl := newSMTPClient(t, addr)

			cl.cmd("EHLO x", "250")
			cl.cmd("MAIL FROM:<first@b.test>", "250")
			cl.cmd(tc.mail, tc.code)

			// there is no transaction
			cl.cmd("RCPT TO:<c@d.test>", "503")
			cl.cmd("DATA", "503")

			// a new one is fine
			cl.cmd("MAIL FROM:<third@b.test>", "250")
			cl.cmd("RCPT TO:<c@d.test>", "250")
			cl.cmd("DATA", "354")
			_, _ = cl.c.Write([]byte("Subject: s\r\n\r\nbody\r\n.\r\n"))
			cl.expect("250")

			if got := rec.get(); len(got) != 1 || got[0] != "third@b.test" {
				t.Fatalf("senders = %v, want only third@b.test", got)
			}
		})
	}
}

// Control: a second MAIL which is accepted replaces the first, and a MAIL which is refused because a transaction is
// in progress (nested) does not end it.
func TestAcceptedAndNestedMail(t *testing.T) {
	rec := &senderRecorder{}
	addr, _ := limitServer(t, nil, rec.handler)
	cl := newSMTPClient(t, addr)

	cl.cmd("EHLO x", "250")
	cl.cmd("MAIL FROM:<first@b.test>", "250")
	cl.cmd("MAIL FROM:<second@b.test>", "250") // replaces the first
	cl.cmd("RCPT TO:<c@d.test>", "250")

	cl.cmd("MAIL FROM:<nested@b.test>", "503") // refused, the transaction goes on
	cl.cmd("RCPT TO:<e@d.test>", "250")
	cl.cmd("DATA", "354")
	_, _ = cl.c.Write([]byte("Subject: s\r\n\r\nbody\r\n.\r\n"))
	cl.expect("250")

	if got := rec.get(); len(got) != 1 || got[0] != "second@b.test" {
		t.Fatalf("senders = %v, want only second@b.test", got)
	}
}

// A MAIL which chaos refuses ends the transaction as well.
func TestChaosRefusedMailEndsTheTransaction(t *testing.T) {
	rec := &senderRecorder{}
	addr, srv := limitServer(t, nil, rec.handler)
	cl := newSMTPClient(t, addr)

	cl.cmd("EHLO x", "250")
	cl.cmd("MAIL FROM:<first@b.test>", "250")

	// the settings are global, so they are put back once the session has ended
	origEnabled, origTriggers := chaos.Enabled, chaos.Get()
	t.Cleanup(func() {
		_ = cl.c.Close()
		_ = srv.Close()
		waitForSessions(srv, 0, 3*time.Second)
		_ = chaos.SetFromStruct(origTriggers)
		chaos.Enabled = origEnabled
	})

	if err := chaos.Set("sender", 451, 100); err != nil {
		t.Fatal(err)
	}

	cl.cmd("MAIL FROM:<second@b.test>", "451")

	if err := chaos.Set("sender", 451, 0); err != nil {
		t.Fatal(err)
	}
	cl.cmd("RCPT TO:<c@d.test>", "503")

	cl.cmd("NOOP", "250")
}
