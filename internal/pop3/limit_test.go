package pop3

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/logger"
)

func TestMaxConnections(t *testing.T) {
	_ = logger.Log() // initialise before concurrent use
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go serve(ln, 2)

	connect := func() (net.Conn, string) {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
		line, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			t.Fatalf("failed to read greeting: %v", err)
		}
		return c, line
	}

	// control: connections within the limit are greeted normally
	var held []net.Conn
	for range 2 {
		c, greeting := connect()
		if !strings.HasPrefix(greeting, "+OK") {
			t.Fatalf("expected +OK greeting, got %q", greeting)
		}
		held = append(held, c)
	}

	// exploit: the next connection is refused
	c, greeting := connect()
	_ = c.Close()
	if !strings.HasPrefix(greeting, "-ERR") {
		t.Fatalf("expected -ERR beyond the limit, got %q", greeting)
	}

	// control: a slot is reusable after a client disconnects
	_ = held[0].Close()
	var reused bool
	for range 20 {
		c, greeting := connect()
		_ = c.Close()
		if strings.HasPrefix(greeting, "+OK") {
			reused = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !reused {
		t.Fatal("slot was not reusable after a client disconnected")
	}

	_ = held[1].Close()
}
