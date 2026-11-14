package pop3

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/snakeoil"
	"github.com/axllent/mailpit/internal/storage"
)

// rawClient is a minimal POP3 client which exposes exactly what is on the wire.
type rawClient struct {
	t    *testing.T
	conn net.Conn
	r    *bufio.Reader
}

func dialRaw(t *testing.T) *rawClient {
	t.Helper()

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", testingPort))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &rawClient{t: t, conn: conn, r: bufio.NewReader(conn)}
}

func (c *rawClient) send(s string) {
	c.t.Helper()

	if _, err := io.WriteString(c.conn, s); err != nil {
		c.t.Fatal(err)
	}
}

func (c *rawClient) line() (string, error) {
	_ = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	l, err := c.r.ReadString('\n')

	return strings.TrimRight(l, "\r\n"), err
}

func (c *rawClient) mustLine() string {
	c.t.Helper()

	l, err := c.line()
	if err != nil {
		c.t.Fatalf("reading a line: %v", err)
	}

	return l
}

// multi reads the lines of a multi-line response, as sent on the wire, up to the terminating ".".
func (c *rawClient) multi() []string {
	c.t.Helper()

	var lines []string
	for {
		l := c.mustLine()
		if l == "." {
			return lines
		}
		lines = append(lines, l)
	}
}

func (c *rawClient) login() {
	c.t.Helper()

	if g := c.mustLine(); !strings.HasPrefix(g, "+OK") {
		c.t.Fatalf("greeting: %q", g)
	}
	c.send("USER username\r\nPASS password\r\n")
	for range 2 {
		if l := c.mustLine(); !strings.HasPrefix(l, "+OK") {
			c.t.Fatalf("login: %q", l)
		}
	}
}

func storeRaw(t *testing.T, raw string) {
	t.Helper()

	b := []byte(raw)
	if _, err := storage.Store(&b, nil); err != nil {
		t.Fatalf("storing message: %v", err)
	}
}

func bodyLines(n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&sb, "line%d\r\n", i)
	}

	return sb.String()
}

func TestTopOfMessage(t *testing.T) {
	crlf := "A: 1\r\nB: 2\r\n\r\nl1\r\nl2\r\nl3\r\n"
	lf := "A: 1\nB: 2\n\nl1\nl2\nl3\n"

	tests := []struct {
		name string
		raw  string
		nr   int
		want string
	}{
		{"zero lines is the headers and the blank line", crlf, 0, "A: 1\r\nB: 2\r\n\r\n"},
		{"one line", crlf, 1, "A: 1\r\nB: 2\r\n\r\nl1\r\n"},
		{"two lines of three", crlf, 2, "A: 1\r\nB: 2\r\n\r\nl1\r\nl2\r\n"},
		{"all lines", crlf, 3, crlf},
		{"more lines than the body has", crlf, 100, crlf},
		{"negative is the same as zero", crlf, -5, "A: 1\r\nB: 2\r\n\r\n"},
		{"LF line endings", lf, 2, "A: 1\nB: 2\n\nl1\nl2\n"},
		{"last body line without a line ending", "A: 1\r\n\r\nl1\r\nl2", 5, "A: 1\r\n\r\nl1\r\nl2"},
		{"no body", "A: 1\r\nB: 2\r\n", 3, "A: 1\r\nB: 2\r\n"},
		{"blank line and nothing after it", "A: 1\r\n\r\n", 3, "A: 1\r\n\r\n"},
		{"empty message", "", 3, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(topOfMessage([]byte(tt.raw), tt.nr)); got != tt.want {
				t.Fatalf("topOfMessage(%q, %d) = %q, want %q", tt.raw, tt.nr, got, tt.want)
			}
		})
	}
}

func TestSendMultiline(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"control: plain lines", "a\r\nb\r\n", "a\r\nb\r\n"},
		{"no final line ending gets one", "a\r\nb", "a\r\nb\r\n"},
		{"LF endings become CRLF", "a\nb\n", "a\r\nb\r\n"},
		{"blank lines are kept", "a\r\n\r\n\r\nb\r\n", "a\r\n\r\n\r\nb\r\n"},
		{"empty", "", ""},
		{"dot inside a line is left alone", "a.b\r\nc.\r\n", "a.b\r\nc.\r\n"},
		{"dot line in the middle", "a\r\n.\r\nb\r\n", "a\r\n..\r\nb\r\n"},
		{"line starting with a dot in the middle", "a\r\n.b\r\n", "a\r\n..b\r\n"},
		{"two dots", "a\r\n..\r\n", "a\r\n...\r\n"},
		// the first line must be stuffed like any other
		{"dot line first", ".\r\nb\r\n", "..\r\nb\r\n"},
		{"line starting with a dot first", ".x\r\n", "..x\r\n"},
		{"only a dot, no line ending", ".", "..\r\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()

			got := make(chan string, 1)
			go func() {
				b, _ := io.ReadAll(client)
				got <- string(b)
			}()

			if err := sendMultiline(server, []byte(tt.in)); err != nil {
				t.Fatal(err)
			}
			_ = server.Close()

			if out := <-got; out != tt.want {
				t.Fatalf("sendMultiline(%q) wrote %q, want %q", tt.in, out, tt.want)
			}
		})
	}
}

// A large message is sent intact: the output must be exactly the message with one dot added to the
// lines that start with one, and removing it again must give the message back.
func TestSendMultilineLargeMessageIsIntact(t *testing.T) {
	var lines []string
	for size, i := 0, 0; size < 8<<20; i++ {
		var l string
		switch i % 4 {
		case 0:
			l = ".leading dot"
		case 1:
			l = "normal line of text with some length to it"
		case 2:
			l = ".."
		default:
			l = ""
		}
		lines = append(lines, l)
		size += len(l) + 2
	}
	in := strings.Join(lines, "\r\n") + "\r\n"

	var want strings.Builder
	for _, l := range lines {
		if strings.HasPrefix(l, ".") {
			_, _ = want.WriteString(".")
		}
		_, _ = want.WriteString(l + "\r\n")
	}

	client, server := net.Pipe()
	defer client.Close()

	got := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(client)
		got <- string(b)
	}()

	if err := sendMultiline(server, []byte(in)); err != nil {
		t.Fatal(err)
	}
	_ = server.Close()

	out := <-got
	if out != want.String() {
		t.Fatalf("output differs from the expected stuffed message (%d bytes, want %d)", len(out), want.Len())
	}

	// what a client does: remove one leading dot from every line that has one
	var unstuffed []string
	for _, l := range strings.Split(strings.TrimSuffix(out, "\r\n"), "\r\n") {
		unstuffed = append(unstuffed, strings.TrimPrefix(l, "."))
	}
	if len(unstuffed) != len(lines) {
		t.Fatalf("%d lines, want %d", len(unstuffed), len(lines))
	}
	for i := range lines {
		if unstuffed[i] != lines[i] {
			t.Fatalf("line %d: %q, want %q", i, unstuffed[i], lines[i])
		}
	}
}

// TOP returns only the number of body lines requested. It used to return the whole body for any
// value of one or more.
func TestTopReturnsRequestedLines(t *testing.T) {
	setup()

	storeRaw(t, "From: a@b.c\r\nSubject: s\r\n\r\n"+bodyLines(10))

	for _, tt := range []struct {
		n    int
		want int
	}{{0, 0}, {1, 1}, {3, 3}, {10, 10}, {50, 10}} {
		c := dialRaw(t)
		c.login()
		c.send(fmt.Sprintf("TOP 1 %d\r\n", tt.n))

		if l := c.mustLine(); !strings.HasPrefix(l, "+OK") {
			t.Fatalf("TOP: %q", l)
		}

		var body []string
		var blank int
		sawHeadersEnd := false
		for _, l := range c.multi() {
			switch {
			case strings.HasPrefix(l, "line"):
				body = append(body, l)
			case l == "":
				blank++
				sawHeadersEnd = true
			}
		}

		if len(body) != tt.want {
			t.Errorf("TOP 1 %d returned %d body lines, want %d", tt.n, len(body), tt.want)
		}
		if !sawHeadersEnd || blank != 1 {
			t.Errorf("TOP 1 %d returned %d blank lines, want exactly the one ending the headers", tt.n, blank)
		}
	}

	// control: RETR still returns the whole message
	c := dialRaw(t)
	c.login()
	c.send("RETR 1\r\n")
	c.mustLine()
	var all int
	for _, l := range c.multi() {
		if strings.HasPrefix(l, "line") {
			all++
		}
	}
	if all != 10 {
		t.Errorf("RETR returned %d body lines, want 10", all)
	}
}

// A body line starting with a dot must be stuffed in both RETR and TOP. If the first body line was
// not, a body of ".", then "+OK injected" would end the response early and the rest of the message
// would be read by the client as the replies to its next commands.
func TestDotStuffingOnTheWire(t *testing.T) {
	setup()

	tests := []struct {
		name string
		body string
		want []string // the body lines as they must be on the wire
	}{
		{"control: ordinary first line", "x\r\n+OK not an injection\r\nz\r\n", []string{"x", "+OK not an injection", "z"}},
		{"control: dot later in the body", "x\r\n.\r\nz\r\n", []string{"x", "..", "z"}},
		{"exploit: first line is a dot", ".\r\n+OK injected response\r\nz\r\n", []string{"..", "+OK injected response", "z"}},
		{"exploit: first line starts with a dot", ".x\r\nz\r\n", []string{"..x", "z"}},
		{"exploit: first line is two dots", "..\r\nz\r\n", []string{"...", "z"}},
	}

	for _, tt := range tests {
		for _, cmd := range []string{"TOP 1 100", "RETR 1"} {
			t.Run(tt.name+" "+cmd, func(t *testing.T) {
				if err := storage.DeleteAllMessages(); err != nil {
					t.Fatal(err)
				}
				storeRaw(t, "From: a@b.c\r\nSubject: s\r\n\r\n"+tt.body)

				c := dialRaw(t)
				c.login()
				// the next command is sent at once, as a pipelining client would
				c.send(cmd + "\r\nNOOP\r\n")

				if l := c.mustLine(); !strings.HasPrefix(l, "+OK") {
					t.Fatalf("%s: %q", cmd, l)
				}
				lines := c.multi()

				// the reply to NOOP must be the next thing sent, not part of the message
				if l := c.mustLine(); l != "+OK" {
					t.Fatalf("reply to NOOP = %q, the response to %s ended early or was corrupted (lines %q)", l, cmd, lines)
				}

				// everything after the blank line ending the headers is the body
				var body []string
				for i, l := range lines {
					if l == "" {
						body = lines[i+1:]
						break
					}
				}
				if strings.Join(body, "|") != strings.Join(tt.want, "|") {
					t.Fatalf("body on the wire = %q, want %q", body, tt.want)
				}
			})
		}
	}
}

func TestFailedLoginsAreDelayed(t *testing.T) {
	setup()

	// control: a successful login is not delayed
	c := dialRaw(t)
	start := time.Now()
	c.login()
	if d := time.Since(start); d >= failedLoginDelay {
		t.Fatalf("a successful login took %v", d)
	}

	// exploit: guesses pipelined in one write are still answered one delay apart
	c = dialRaw(t)
	c.mustLine()
	c.send("USER username\r\n")
	c.mustLine()

	const guesses = 3
	start = time.Now()
	c.send(strings.Repeat("PASS wrong\r\n", guesses))
	for range guesses {
		if l := c.mustLine(); !strings.HasPrefix(l, "-ERR") {
			t.Fatalf("got %q", l)
		}
	}
	if d := time.Since(start); d < guesses*failedLoginDelay {
		t.Fatalf("%d failed logins were answered in %v, want at least %v", guesses, d, guesses*failedLoginDelay)
	}
}

func TestIdleTimeoutBeforeLogin(t *testing.T) {
	setup()

	// exploit: a client that does nothing is disconnected after authTimeout
	c := dialRaw(t)
	c.mustLine()
	start := time.Now()
	if _, err := c.line(); err != io.EOF {
		t.Fatalf("expected the server to close the connection, got %v", err)
	}
	if d := time.Since(start); d > 3*authTimeout {
		t.Fatalf("took %v to disconnect an idle client", d)
	}

	// control: after logging in the longer timeout applies, so the same wait is fine
	c = dialRaw(t)
	c.login()
	time.Sleep(authTimeout + 500*time.Millisecond)
	c.send("NOOP\r\n")
	if l := c.mustLine(); l != "+OK" {
		t.Fatalf("a logged in client was disconnected after being idle: %q", l)
	}
}

func TestTLSHandshakeTimeout(t *testing.T) {
	cert, key := snakeoil.Public("sans:localhost"), snakeoil.Private("sans:localhost")
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{Certificates: []tls.Certificate{cer}, MinVersion: tls.VersionTLS12})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	const limit = 2
	go serve(ln, limit)

	tlsClient := func() (string, error) {
		d := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 3 * time.Second}, Config: &tls.Config{InsecureSkipVerify: true}} // #nosec
		conn, err := d.Dial("tcp", ln.Addr().String())
		if err != nil {
			return "", err
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		return bufio.NewReader(conn).ReadString('\n')
	}

	// control: a normal client is greeted
	if g, err := tlsClient(); err != nil || !strings.HasPrefix(g, "+OK") {
		t.Fatalf("control client: %q, %v", g, err)
	}

	// the control client's slot is released after the server notices it has gone
	time.Sleep(300 * time.Millisecond)

	// exploit: clients that connect and never start the handshake fill every slot...
	var idle []net.Conn
	for range limit {
		c, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		idle = append(idle, c)
	}
	defer func() {
		for _, c := range idle {
			_ = c.Close()
		}
	}()
	time.Sleep(300 * time.Millisecond)
	if g, err := tlsClient(); err == nil {
		t.Fatalf("expected the idle clients to hold every slot, but a client was greeted: %q", g)
	}

	// ...but only until the handshake times out
	time.Sleep(handshakeTimeout + 500*time.Millisecond)
	if g, err := tlsClient(); err != nil || !strings.HasPrefix(g, "+OK") {
		t.Fatalf("a client was not served after the handshake timeout: %q, %v", g, err)
	}
}

// A client that asks for a message and never reads it must be dropped, not hold the connection
// for ever. A pipe has no buffering, so the server blocks on the first byte the client does not read.
func TestClientThatStopsReadingIsDropped(t *testing.T) {
	setup()

	if err := storage.DeleteAllMessages(); err != nil {
		t.Fatal(err)
	}
	storeRaw(t, "From: a@b.c\r\nSubject: s\r\n\r\n"+bodyLines(50))

	start := func() (net.Conn, *bufio.Reader, chan struct{}) {
		client, server := net.Pipe()
		t.Cleanup(func() { _ = client.Close() })

		done := make(chan struct{})
		go func() {
			handleClient(server)
			close(done)
		}()

		r := bufio.NewReader(client)
		readLine := func() string {
			_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
			l, err := r.ReadString('\n')
			if err != nil {
				t.Fatalf("reading: %v", err)
			}

			return strings.TrimRight(l, "\r\n")
		}

		readLine() // greeting
		for _, cmd := range []string{"USER username", "PASS password"} {
			if _, err := io.WriteString(client, cmd+"\r\n"); err != nil {
				t.Fatal(err)
			}
			if l := readLine(); !strings.HasPrefix(l, "+OK") {
				t.Fatalf("%s: %q", cmd, l)
			}
		}

		return client, r, done
	}

	// control: a client that reads the message is not dropped and can carry on
	client, r, done := start()
	_, _ = io.WriteString(client, "RETR 1\r\n")
	lines := 0
	for {
		_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
		l, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading the message: %v", err)
		}
		if strings.TrimRight(l, "\r\n") == "." {
			break
		}
		lines++
	}
	if lines < 50 {
		t.Fatalf("only %d lines received", lines)
	}
	_, _ = io.WriteString(client, "NOOP\r\n")
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	if l, err := r.ReadString('\n'); err != nil || !strings.HasPrefix(l, "+OK") {
		t.Fatalf("the connection was dropped while the client was reading: %q, %v", l, err)
	}
	_ = client.Close()
	<-done

	// exploit: a client that asks for the message and then reads nothing is dropped
	client, _, done = start()
	_, _ = io.WriteString(client, "RETR 1\r\n")
	select {
	case <-done:
	case <-time.After(10 * writeTimeout):
		t.Fatal("the server is still waiting to write to a client that is not reading")
	}
}

// A panic while serving one client must not stop the server, or leak the client's connection slot.
func TestPanicWhileServingIsRecovered(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var served int
	go serveWith(ln, 1, func(conn net.Conn) {
		defer conn.Close()
		served++
		if served == 1 {
			panic("boom")
		}
		_, _ = io.WriteString(conn, "+OK hello\r\n")
	})

	greeting := func() (string, error) {
		conn, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			return "", err
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))

		return bufio.NewReader(conn).ReadString('\n')
	}

	// the first connection panics, so the client only sees it closed
	if g, err := greeting(); err != io.EOF {
		t.Fatalf("first connection: %q, %v", g, err)
	}

	// control: the next connection works, which also shows the single slot was released
	var g string
	for range 20 {
		if g, err = greeting(); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil || g != "+OK hello\r\n" {
		t.Fatalf("second connection: %q, %v", g, err)
	}
}

func TestClientErrorBroadcastsAreLimited(t *testing.T) {
	setup()

	// let the limiter refill, as earlier tests also produce errors
	time.Sleep(2500 * time.Millisecond)

	badCommands := func(n int) {
		c := dialRaw(t)
		c.mustLine()
		c.send(strings.Repeat("BAD\r\n", n))
		for range n {
			if l := c.mustLine(); !strings.HasPrefix(l, "-ERR") {
				t.Fatalf("got %q", l)
			}
		}
	}

	// the server replies first and then broadcasts, so wait for the count to stop changing
	settled := func(before int64) int64 {
		last, stable := before, 0
		for range 50 {
			time.Sleep(50 * time.Millisecond)
			if now := broadcasts.Load(); now != last {
				last, stable = now, 0
			} else if stable++; stable >= 4 {
				break
			}
		}

		return last - before
	}

	// control: a few errors are all reported
	before := broadcasts.Load()
	badCommands(3)
	if got := settled(before); got != 3 {
		t.Fatalf("%d of 3 errors were broadcast", got)
	}

	// exploit: a flood of errors is not
	before = broadcasts.Load()
	badCommands(200)
	if got := settled(before); got < 1 || got > 20 {
		t.Fatalf("%d of 200 errors were broadcast, want a handful", got)
	}
}
