package pop3

import (
	"bufio"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeServer returns a client connection to a server which answers every command with reply, and
// stops when the client closes the connection.
func fakeServer(t *testing.T, reply string) *testConn {
	t.Helper()

	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })

	go func() {
		r := bufio.NewReader(server)
		for {
			if _, err := r.ReadString('\n'); err != nil {
				return
			}
			if _, err := io.WriteString(server, reply); err != nil {
				return
			}
		}
	}()

	return &testConn{conn: client, r: bufio.NewReader(client), w: bufio.NewWriter(client), timeout: 2 * time.Second}
}

func TestClientRemovesByteStuffing(t *testing.T) {
	tests := []struct {
		name, reply, want string
	}{
		{"control: no dots", "+OK\r\nplain\r\nlines\r\n.\r\n", "plain\r\nlines\r\n"},
		{"control: dot inside a line", "+OK\r\na.b\r\nc.\r\n.\r\n", "a.b\r\nc.\r\n"},
		{"exploit: stuffed dot line", "+OK\r\n..\r\n.\r\n", ".\r\n"},
		{"exploit: stuffed line starting with a dot", "+OK\r\n..dot line\r\n...two\r\nplain\r\n.\r\n", ".dot line\r\n..two\r\nplain\r\n"},
		{"empty message", "+OK\r\n.\r\n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := fakeServer(t, tt.reply).RetrRaw(1)
			if err != nil {
				t.Fatal(err)
			}
			if b.String() != tt.want {
				t.Fatalf("got %q, want %q", b.String(), tt.want)
			}
		})
	}
}

// A line is returned whole whatever its length. It used to be split at the size of the read buffer.
func TestClientReadsLongLinesWhole(t *testing.T) {
	for _, n := range []int{100, 4095, 4096, 4097, 100000} {
		line := strings.Repeat("a", n)
		b, err := fakeServer(t, "+OK\r\n"+line+"\r\nnext\r\n.\r\n").RetrRaw(1)
		if err != nil {
			t.Fatal(err)
		}
		if want := line + "\r\nnext\r\n"; b.String() != want {
			t.Fatalf("a line of %d bytes came back as %d bytes in %d lines", n, b.Len(), strings.Count(b.String(), "\r\n"))
		}
	}
}

// Replies that are too short are errors. They used to panic with an index out of range.
func TestClientShortRepliesAreErrors(t *testing.T) {
	stat := func(c *testConn) error { _, _, err := c.Stat(); return err }
	list := func(c *testConn) error { _, err := c.List(0); return err }
	uidl := func(c *testConn) error { _, err := c.Uidl(0); return err }

	tests := []struct {
		name    string
		call    func(c *testConn) error
		reply   string
		wantErr bool
	}{
		{"control: STAT", stat, "+OK 2 100\r\n", false},
		{"control: STAT of an empty mailbox", stat, "+OK 0 0\r\n", false},
		{"exploit: STAT without counts", stat, "+OK\r\n", true},
		{"exploit: STAT without a size", stat, "+OK 2\r\n", true},
		{"control: LIST", list, "+OK\r\n1 100\r\n.\r\n", false},
		{"exploit: LIST line without a size", list, "+OK\r\n1\r\n.\r\n", true},
		{"control: UIDL", uidl, "+OK\r\n1 abc\r\n.\r\n", false},
		{"exploit: UIDL line without a unique id", uidl, "+OK\r\n1\r\n.\r\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call(fakeServer(t, tt.reply))
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// A server that stops responding fails the call promptly instead of hanging the test.
func TestClientTimesOutOnASilentServer(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	go func() { _, _ = io.Copy(io.Discard, server) }() // reads the command, never answers

	c := &testConn{conn: client, r: bufio.NewReader(client), w: bufio.NewWriter(client), timeout: 300 * time.Millisecond}

	start := time.Now()
	if err := c.Noop(); err == nil {
		t.Fatal("expected an error")
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("took %v to give up", d)
	}
}
