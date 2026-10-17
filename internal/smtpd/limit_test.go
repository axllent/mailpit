package smtpd

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

func TestMaxConnections(t *testing.T) {
	maxConnections = 3
	defer func() { maxConnections = 1000 }()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	srv := &Server{DisableReverseDNS: true, Hostname: "test", AppName: "test"}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Shutdown(t.Context()); _ = ln.Close() }()

	connect := func() (net.Conn, string) {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		line, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			t.Fatalf("failed to read banner: %v", err)
		}
		return c, line
	}

	// control: connections within the limit get a normal banner
	var held []net.Conn
	for range 3 {
		c, banner := connect()
		if !strings.HasPrefix(banner, "220 ") {
			t.Fatalf("expected 220 banner, got %q", banner)
		}
		held = append(held, c)
	}

	// exploit: the next connection is refused with 421
	c, banner := connect()
	_ = c.Close()
	if !strings.HasPrefix(banner, "421 ") {
		t.Fatalf("expected 421 beyond the limit, got %q", banner)
	}

	// control: once a session ends its slot is reusable
	_ = held[0].Close()
	var reused bool
	for range 20 {
		c, banner := connect()
		_ = c.Close()
		if strings.HasPrefix(banner, "220 ") {
			reused = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !reused {
		t.Fatal("slot was not reusable after a session closed")
	}

	for _, c := range held {
		_ = c.Close()
	}
}
