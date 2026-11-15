package smtpd

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// rdnsServer starts a server whose reverse lookups are answered by lookup, and records the data of each message.
func rdnsServer(t *testing.T, timeout time.Duration, lookup func(ctx context.Context, ip string) ([]string, error), options ...func(*Server)) (addr string, srv *Server, messages func() []string) {
	t.Helper()

	var mu sync.Mutex
	var got []string

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv = &Server{
		Hostname:          "test",
		AppName:           "test",
		Timeout:           5 * time.Second,
		lookupAddr:        lookup,
		reverseDNSTimeout: timeout,
		XClientAllowed:    []string{"127.0.0.1"},
		MsgIDHandler: func(_ net.Addr, _ string, _ []string, data []byte, _ *string) (string, error) {
			mu.Lock()
			defer mu.Unlock()
			got = append(got, string(data))
			return "", nil
		},
	}
	for _, option := range options {
		option(srv)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close(); _ = ln.Close() })

	return ln.Addr().String(), srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

// dialFrom connects from a loopback address (127.0.0.2 is not 127.0.0.1, so a lookup can tell them apart).
func dialFrom(t *testing.T, from, addr string) net.Conn {
	t.Helper()

	d := net.Dialer{LocalAddr: &net.TCPAddr{IP: net.ParseIP(from)}}
	c, err := d.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })

	return c
}

func banner(c net.Conn, wait time.Duration) (string, error) {
	_ = c.SetReadDeadline(time.Now().Add(wait))
	return bufio.NewReader(c).ReadString('\n')
}

// hangs is a lookup which takes until it is given up on.
func hangs(ctx context.Context, _ string) ([]string, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

// A lookup which is slow does not stop other clients from connecting. The lookup used to be done by the loop which
// accepts connections.
func TestReverseDNSDoesNotBlockAccept(t *testing.T) {
	// 127.0.0.2 has a lookup which does not finish, the others are answered at once
	lookup := func(ctx context.Context, ip string) ([]string, error) {
		if ip == "127.0.0.2" {
			return hangs(ctx, ip)
		}
		return []string{"fast.example"}, nil
	}
	addr, _, _ := rdnsServer(t, 10*time.Second, lookup)

	slow := dialFrom(t, "127.0.0.2", addr)

	start := time.Now()
	other := dialFrom(t, "127.0.0.1", addr)
	line, err := banner(other, 2*time.Second)
	if err != nil || !strings.HasPrefix(line, "220 ") {
		t.Fatalf("a client was held up by the lookup of another one: %q, %v", line, err)
	}
	t.Logf("second client got its banner after %v", time.Since(start))

	// the client with the slow lookup is not held up either, as the banner does not wait for it
	line, err = banner(slow, 2*time.Second)
	if err != nil || !strings.HasPrefix(line, "220 ") {
		t.Fatalf("the banner waited for the lookup: %q, %v", line, err)
	}
}

// send runs a transaction and returns the Received header of the message which was stored.
func sendMessage(t *testing.T, c net.Conn, extra ...string) {
	t.Helper()

	br := bufio.NewReader(c)
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := br.ReadString('\n'); err != nil {
		t.Fatal(err)
	}

	cmds := append([]string{"EHLO client"}, extra...)
	cmds = append(cmds, "MAIL FROM:<a@b.test>", "RCPT TO:<c@d.test>", "DATA", "Subject: s\r\n\r\nbody\r\n.", "QUIT")
	for _, cmd := range cmds {
		_, _ = fmt.Fprintf(c, "%s\r\n", cmd)
		for {
			l, err := br.ReadString('\n')
			if err != nil {
				t.Fatalf("after %q: %v", cmd, err)
			}
			if len(l) < 4 || l[3] != '-' {
				break
			}
		}
	}
}

func receivedLine(t *testing.T, msgs []string) string {
	t.Helper()

	if len(msgs) != 1 {
		t.Fatalf("expected one message, got %d", len(msgs))
	}
	line, _, _ := strings.Cut(msgs[0], "\r\n")

	return line
}

// A lookup which takes longer than the timeout gives "unknown" in the Received header, and the message is
// received within about the timeout. A fast lookup (the control) gives the name.
func TestReverseDNSTimeout(t *testing.T) {
	addr, _, messages := rdnsServer(t, 300*time.Millisecond, hangs)

	start := time.Now()
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	took := time.Since(start)

	if got := receivedLine(t, messages()); got != "Received: from client (unknown [127.0.0.2])" {
		t.Fatalf("Received = %q", got)
	}
	if took > 2*time.Second {
		t.Fatalf("the message took %v, the lookup should have been given up on after 300ms", took)
	}

	// control: a lookup which is answered
	addr, _, messages = rdnsServer(t, 300*time.Millisecond, func(context.Context, string) ([]string, error) {
		return []string{"mail.example", "other.example"}, nil
	})
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))

	if got := receivedLine(t, messages()); got != "Received: from client (mail.example [127.0.0.2])" {
		t.Fatalf("Received = %q", got)
	}
}

// A lookup which has an error, or no names, is "unknown" as well (as it was).
func TestReverseDNSFailure(t *testing.T) {
	for name, lookup := range map[string]func(context.Context, string) ([]string, error){
		"error": func(context.Context, string) ([]string, error) { return nil, fmt.Errorf("no such host") },
		"empty": func(context.Context, string) ([]string, error) { return nil, nil },
	} {
		t.Run(name, func(t *testing.T) {
			addr, _, messages := rdnsServer(t, time.Second, lookup)
			sendMessage(t, dialFrom(t, "127.0.0.2", addr))

			if got := receivedLine(t, messages()); got != "Received: from client (unknown [127.0.0.2])" {
				t.Fatalf("Received = %q", got)
			}
		})
	}
}

// Nothing is looked up when it is disabled.
func TestReverseDNSDisabled(t *testing.T) {
	var calls atomic.Int32
	count := func(context.Context, string) ([]string, error) {
		calls.Add(1)
		return []string{"mail.example"}, nil
	}

	addr, _, messages := rdnsServer(t, time.Second, count, func(srv *Server) { srv.DisableReverseDNS = true })
	sendMessage(t, dialFrom(t, "127.0.0.1", addr), "XCLIENT ADDR=10.1.2.3")

	if calls.Load() != 0 {
		t.Fatalf("%d lookups with reverse DNS disabled", calls.Load())
	}
	if got := receivedLine(t, messages()); got != "Received: from client (unknown [10.1.2.3])" {
		t.Fatalf("Received = %q", got)
	}
}

// The lookup of XCLIENT is bounded as well, and the lookup which was started when the client connected is for
// another address, so its name must not end up in the message.
func TestReverseDNSXClient(t *testing.T) {
	lookup := func(ctx context.Context, ip string) ([]string, error) {
		if ip == "10.1.2.3" {
			return hangs(ctx, ip)
		}
		return []string{"connected.example"}, nil
	}

	addr, _, messages := rdnsServer(t, 300*time.Millisecond, lookup)

	start := time.Now()
	sendMessage(t, dialFrom(t, "127.0.0.1", addr), "XCLIENT ADDR=10.1.2.3")
	took := time.Since(start)

	if got := receivedLine(t, messages()); got != "Received: from client (unknown [10.1.2.3])" {
		t.Fatalf("Received = %q", got)
	}
	if took > 2*time.Second {
		t.Fatalf("the message took %v, the lookup should have been given up on after 300ms", took)
	}

	// control: without XCLIENT the name of the connection is used
	addr, _, messages = rdnsServer(t, 300*time.Millisecond, lookup)
	sendMessage(t, dialFrom(t, "127.0.0.1", addr))

	if got := receivedLine(t, messages()); got != "Received: from client (connected.example [127.0.0.1])" {
		t.Fatalf("Received = %q", got)
	}
}
