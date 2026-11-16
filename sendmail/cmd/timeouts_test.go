package cmd

import (
	"bufio"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// behaviour is how a server which is slow to answer behaves.
type behaviour struct {
	silent     bool          // accepts the connection and never says anything
	replyDelay time.Duration // before each answer
	dataDelay  time.Duration // before the answer to the end of the message
	stopAfter  string        // stops answering after this command (a prefix, in upper case)
	quitDelay  time.Duration // before the answer to QUIT
}

func slowServer(t *testing.T, network, addr string, b behaviour) string {
	t.Helper()

	ln, err := net.Listen(network, addr)
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
				_ = c.SetDeadline(time.Now().Add(30 * time.Second))

				if b.silent {
					buf := make([]byte, 512)
					for {
						if _, err := c.Read(buf); err != nil {
							return
						}
					}
				}

				w := func(delay time.Duration, s string) {
					time.Sleep(delay)
					_, _ = c.Write([]byte(s + "\r\n"))
				}
				r := bufio.NewReader(c)
				w(b.replyDelay, "220 fake")
				for {
					l, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd := strings.ToUpper(strings.TrimSpace(l))
					if b.stopAfter != "" && strings.HasPrefix(cmd, b.stopAfter) {
						time.Sleep(10 * time.Second) // never answers
						return
					}
					switch {
					case cmd == "DATA":
						w(b.replyDelay, "354 go")
						for {
							d, err := r.ReadString('\n')
							if err != nil {
								return
							}
							if d == ".\r\n" {
								break
							}
						}
						w(b.dataDelay, "250 queued")
					case cmd == "QUIT":
						w(b.quitDelay, "221 bye")
						return
					default:
						w(b.replyDelay, "250 ok")
					}
				}
			}()
		}
	}()

	return ln.Addr().String()
}

func withTimeouts(t *testing.T, dial, idle, reply time.Duration) {
	t.Helper()

	origDial, origIdle, origReply := dialTimeout, idleTimeout, replyTimeout
	dialTimeout, idleTimeout, replyTimeout = dial, idle, reply
	t.Cleanup(func() { dialTimeout, idleTimeout, replyTimeout = origDial, origIdle, origReply })
}

// send makes the Send, and gives up on it (the test fails) if it is still going after the time given.
func send(t *testing.T, addr string, giveUp time.Duration) (error, time.Duration) {
	t.Helper()

	type result struct {
		err  error
		took time.Duration
	}
	done := make(chan result, 1)
	go func() {
		start := time.Now()
		err := Send(addr, "a@b.test", []string{"c@d.test"}, []byte("Subject: x\r\n\r\nbody\r\n"))
		done <- result{err, time.Since(start)}
	}()

	select {
	case r := <-done:
		return r.err, r.took
	case <-time.After(giveUp):
		t.Fatalf("Send was still going after %v", giveUp)
		return nil, 0
	}
}

// each way of connecting: TCP, and a Unix socket
func servers(t *testing.T, b behaviour) map[string]string {
	t.Helper()

	return map[string]string{
		"tcp":  slowServer(t, "tcp", "127.0.0.1:0", b),
		"unix": "unix:" + slowServer(t, "unix", filepath.Join(t.TempDir(), "smtp.sock"), b),
	}
}

// A server which accepts the connection and says nothing (so Mailpit is stuck), or which stops answering part way,
// is given up on after the time limit. Send waited for ever, which held up the program which was sending (as
// sendmail, the PHP request which called mail()).
func TestSendGivesUpOnAServerWhichStopsAnswering(t *testing.T) {
	withTimeouts(t, time.Second, 300*time.Millisecond, 2*time.Second)

	for name, b := range map[string]behaviour{
		"never says anything":   {silent: true},
		"stops after EHLO":      {stopAfter: "MAIL"},
		"stops after MAIL":      {stopAfter: "RCPT"},
		"stops before the data": {stopAfter: "DATA"},
	} {
		for network, addr := range servers(t, b) {
			t.Run(name+" "+network, func(t *testing.T) {
				err, took := send(t, addr, 10*time.Second)
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "deadline") {
					t.Fatalf("err = %v, want a timeout", err)
				}
				if took > 3*time.Second {
					t.Fatalf("took %v, the time limit is 300ms", took)
				}
			})
		}
	}
}

// A server which is slow, but which does answer each time, is waited for: the limit is for a pause, not for how long
// all of it takes. It is a message, with each of the seven answers taking more than half of the limit.
func TestSendWaitsForAServerWhichIsSlowButMoving(t *testing.T) {
	withTimeouts(t, time.Second, 500*time.Millisecond, 2*time.Second)

	for network, addr := range servers(t, behaviour{replyDelay: 300 * time.Millisecond}) {
		t.Run(network, func(t *testing.T) {
			err, took := send(t, addr, 20*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			if took < 1500*time.Millisecond {
				t.Fatalf("took %v, which is not long enough to show anything", took)
			}
		})
	}
}

// The answer to the end of the message has more time, as Mailpit can wait for its turn to store it before it answers
// (up to two minutes), and giving up sooner would report an error for a message which is stored (and send it again).
// A server which takes longer than that is given up on, and so is one which answers other things as slowly.
func TestSendWaitsLongerForTheReplyToTheMessage(t *testing.T) {
	for name, tc := range map[string]struct {
		idle, reply, dataDelay time.Duration
		fails                  bool
	}{
		"longer than the limit, within the reply limit": {300 * time.Millisecond, 3 * time.Second, 900 * time.Millisecond, false},
		"longer than the reply limit":                   {300 * time.Millisecond, 600 * time.Millisecond, 3 * time.Second, true},
		"control: the same limit for everything":        {300 * time.Millisecond, 300 * time.Millisecond, 900 * time.Millisecond, true},
		"control: quick":                                {300 * time.Millisecond, 3 * time.Second, 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			withTimeouts(t, time.Second, tc.idle, tc.reply)

			for network, addr := range servers(t, behaviour{dataDelay: tc.dataDelay}) {
				err, _ := send(t, addr, 20*time.Second)
				if (err != nil) != tc.fails {
					t.Errorf("%s: err = %v, want an error: %v", network, err, tc.fails)
				}
			}
		})
	}
}

// Control: a server which is not there is an error at once, as it was.
func TestSendToAServerWhichIsNotThere(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	_ = ln.Close()

	for name, a := range map[string]string{"tcp": addr, "unix": "unix:" + filepath.Join(t.TempDir(), "none.sock")} {
		err, took := send(t, a, 5*time.Second)
		if err == nil || took > 2*time.Second {
			t.Errorf("%s: %v after %v", name, err, took)
		}
	}
}

// Once the server has said that it has the message it is sent, whatever the server does after that: it is only being
// told that we are going. An error here would be taken for a message which was not sent, and it would be sent again.
func TestSendIsSentWhateverTheServerDoesAfterIt(t *testing.T) {
	// the limit for the reply to the message is long, which QUIT must not have (the server takes 5 seconds)
	withTimeouts(t, time.Second, 300*time.Millisecond, 6*time.Second)

	for network, addr := range servers(t, behaviour{quitDelay: 5 * time.Second}) {
		t.Run(network, func(t *testing.T) {
			err, took := send(t, addr, 10*time.Second)
			if err != nil {
				t.Fatalf("a message which the server accepted is an error: %v", err)
			}
			if took > 3*time.Second {
				t.Fatalf("took %v, the limit is 300ms", took)
			}
		})
	}
}
