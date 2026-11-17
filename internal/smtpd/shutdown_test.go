package smtpd

import (
	"bufio"
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func shutdownServer(t *testing.T) (string, *Server) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{DisableReverseDNS: true, Hostname: "t", AppName: "t"}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close(); _ = ln.Close() })

	return ln.Addr().String(), srv
}

// openSession connects and waits for the banner, so the session is open.
func openSession(t *testing.T, addr string) net.Conn {
	t.Helper()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	if _, err := bufio.NewReader(c).ReadString('\n'); err != nil {
		t.Fatal(err)
	}

	return c
}

// Shutdown waits for a session which is open until the context is done. It used to return at once.
func TestShutdownWaitsUntilTheContextIsDone(t *testing.T) {
	addr, srv := shutdownServer(t)
	openSession(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := srv.Shutdown(ctx)
	took := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown returned %v after %v with a session still open, want the context's error", err, took)
	}
	if took < 400*time.Millisecond {
		t.Fatalf("Shutdown returned after %v, it should have waited for the context", took)
	}
}

// Shutdown returns when the last session has ended, and not before.
func TestShutdownReturnsWhenSessionsEnd(t *testing.T) {
	addr, srv := shutdownServer(t)
	c := openSession(t, addr)

	go func() {
		time.Sleep(400 * time.Millisecond)
		_ = c.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := srv.Shutdown(ctx)
	took := time.Since(start)

	if err != nil || srv.openSessions.Load() != 0 {
		t.Fatalf("Shutdown returned %v with %d sessions open", err, srv.openSessions.Load())
	}
	if took < 300*time.Millisecond || took > 3*time.Second {
		t.Fatalf("Shutdown took %v, the session ended after 400ms", took)
	}
}

// Control: with no sessions open it returns at once (and can be called again).
func TestShutdownWithoutSessions(t *testing.T) {
	_, srv := shutdownServer(t)

	start := time.Now()
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); took > time.Second {
		t.Fatalf("Shutdown took %v with no sessions open", took)
	}

	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("a second Shutdown returned %v", err)
	}
}
