package smtpd

import (
	"bufio"
	"crypto/tls"
	"net"
	"strings"
	"testing"
	"time"
)

// tlsListenerServer starts a server which only accepts TLS connections (as with --smtp-require-tls).
func tlsListenerServer(t *testing.T, timeout time.Duration) (string, *Server) {
	t.Helper()

	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}

	srv := &Server{DisableReverseDNS: true, Hostname: "test", AppName: "test", Timeout: timeout, TLSConfig: cfg, TLSListener: true}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close(); _ = ln.Close() })

	return ln.Addr().String(), srv
}

func waitForSessions(srv *Server, want int32, wait time.Duration) bool {
	deadline := time.Now().Add(wait)
	for time.Now().Before(deadline) {
		if srv.openSessions.Load() == want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return srv.openSessions.Load() == want
}

// tlsBanner completes the handshake and returns the banner.
func tlsBanner(addr string) (string, error) {
	c, err := tls.DialWithDialer(&net.Dialer{Timeout: 2 * time.Second}, "tcp", addr, &tls.Config{InsecureSkipVerify: true}) // #nosec
	if err != nil {
		return "", err
	}
	defer func() { _ = c.Close() }()
	_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))

	return bufio.NewReader(c).ReadString('\n')
}

// A client which connects and never starts the handshake is dropped after the timeout, as is one which stops half
// way through it. The session used to wait for ever, as the timeouts started after the banner.
func TestTLSListenerDropsSilentClients(t *testing.T) {
	addr, srv := tlsListenerServer(t, 300*time.Millisecond)

	var clients []net.Conn
	for _, hello := range []string{"", "\x16\x03\x01"} {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		if hello != "" {
			_, _ = c.Write([]byte(hello))
		}
		clients = append(clients, c)
		t.Cleanup(func() { _ = c.Close() })
	}

	if !waitForSessions(srv, 0, 2*time.Second) {
		t.Fatalf("%d sessions are still open after 2s (the timeout is 300ms)", srv.openSessions.Load())
	}

	// the server has closed the connections
	for _, c := range clients {
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		if _, err := c.Read(make([]byte, 1)); err == nil || strings.Contains(err.Error(), "i/o timeout") {
			t.Fatalf("the connection is still open (%v)", err)
		}
	}
}

// Silent clients must not use up the connection limit: once they time out a client can connect.
func TestTLSListenerSlotsAreFreed(t *testing.T) {
	maxConnections = 3
	defer func() { maxConnections = 1000 }()

	addr, srv := tlsListenerServer(t, 300*time.Millisecond)

	for range 3 {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Close() })
	}
	if !waitForSessions(srv, 3, time.Second) {
		t.Fatalf("expected 3 sessions, have %d", srv.openSessions.Load())
	}

	// the limit is reached
	if line, err := tlsBanner(addr); err == nil && strings.HasPrefix(line, "220 ") {
		t.Fatalf("a client was served beyond the limit: %q", line)
	}

	// the silent clients time out, and the slots can be used
	if !waitForSessions(srv, 0, 2*time.Second) {
		t.Fatalf("%d sessions are still open", srv.openSessions.Load())
	}
	line, err := tlsBanner(addr)
	if err != nil || !strings.HasPrefix(line, "220 ") {
		t.Fatalf("no banner after the slots were freed: %q, %v", line, err)
	}
}

// Control: a client which handshakes (even a little late) is served as normal, and a session still ends by itself
// when it is idle after that.
func TestTLSListenerServesClients(t *testing.T) {
	addr, srv := tlsListenerServer(t, 600*time.Millisecond)

	// the handshake starts a little after connecting, but within the timeout
	raw, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(150 * time.Millisecond)
	c := tls.Client(raw, &tls.Config{InsecureSkipVerify: true}) // #nosec
	t.Cleanup(func() { _ = c.Close() })

	_ = c.SetDeadline(time.Now().Add(2 * time.Second))
	br := bufio.NewReader(c)
	line, err := br.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "220 ") {
		t.Fatalf("no banner: %q, %v", line, err)
	}

	_, _ = c.Write([]byte("NOOP\r\n"))
	if line, err = br.ReadString('\n'); err != nil || !strings.HasPrefix(line, "250 ") {
		t.Fatalf("NOOP: %q, %v", line, err)
	}

	// idle: the session is ended by the timeout, with the message
	if line, err = br.ReadString('\n'); err != nil || !strings.HasPrefix(line, "421 ") {
		t.Fatalf("expected the timeout message, got %q, %v", line, err)
	}
	if !waitForSessions(srv, 0, 2*time.Second) {
		t.Fatalf("%d sessions are still open", srv.openSessions.Load())
	}
}
