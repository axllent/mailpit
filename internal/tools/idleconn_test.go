package tools

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"
)

// A read which gets nothing for the time limit fails with a timeout, where it would otherwise wait for ever.
func TestIdleConnReadTimesOut(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = a.Close(); _ = b.Close() }()

	c := NewIdleConn(a, 150*time.Millisecond)

	start := time.Now()
	_, err := c.Read(make([]byte, 1))
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if took := time.Since(start); took < 100*time.Millisecond || took > 2*time.Second {
		t.Fatalf("took %v, the limit is 150ms", took)
	}
}

// A write which is not taken for the time limit fails with a timeout.
func TestIdleConnWriteTimesOut(t *testing.T) {
	a, b := net.Pipe() // nobody reads b
	defer func() { _ = a.Close(); _ = b.Close() }()

	c := NewIdleConn(a, 150*time.Millisecond)

	// with a limit on how long to wait, so a write which has none fails the test rather than waiting for ever
	done := make(chan error, 1)
	go func() {
		_, err := c.Write([]byte("x"))
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("err = %v, want a timeout", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the write was still waiting after 5 seconds")
	}
}

// The limit is for each read and write, not for all of them: reads which each get something in time go on, for much
// longer than the limit (control: it is the pause which is not allowed).
func TestIdleConnLimitIsForEachRead(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = a.Close(); _ = b.Close() }()

	go func() {
		for range 10 {
			time.Sleep(100 * time.Millisecond)
			if _, err := b.Write([]byte("x")); err != nil {
				return
			}
		}
	}()

	c := NewIdleConn(a, 250*time.Millisecond)
	start := time.Now()
	for i := range 10 {
		if _, err := c.Read(make([]byte, 1)); err != nil {
			t.Fatalf("read %d after %v: %v", i, time.Since(start), err)
		}
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Fatal("the reads were not slow enough to show anything")
	}

	// and a pause which is longer is not
	if _, err := c.Read(make([]byte, 1)); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("err = %v, want a timeout after a pause", err)
	}
}

// The limit can be changed for the reads and writes which follow.
func TestIdleConnSetTimeout(t *testing.T) {
	a, b := net.Pipe()
	defer func() { _ = a.Close(); _ = b.Close() }()

	c := NewIdleConn(a, 100*time.Millisecond)

	go func() {
		time.Sleep(400 * time.Millisecond)
		_, _ = b.Write([]byte("x"))
	}()

	// longer than the answer takes
	c.SetTimeout(2 * time.Second)
	if _, err := c.Read(make([]byte, 1)); err != nil {
		t.Fatalf("with a longer limit: %v", err)
	}

	// and back to a short one
	c.SetTimeout(100 * time.Millisecond)
	if _, err := c.Read(make([]byte, 1)); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("with a short limit: err = %v", err)
	}
}
