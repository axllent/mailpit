package connlimit

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	l := New("test", 2)

	// control: up to the limit succeeds
	if !l.TryAcquire() || !l.TryAcquire() {
		t.Fatal("expected slots within the limit to be acquired")
	}

	// exploit: one more is refused
	if l.TryAcquire() {
		t.Fatal("expected acquire beyond the limit to fail")
	}

	// control: a released slot is reusable
	l.Release()
	if !l.TryAcquire() {
		t.Fatal("expected released slot to be reusable")
	}
}

func TestListenerBlocksAtLimit(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln := Listener("test", base, 2)
	defer func() { _ = ln.Close() }()

	accepted := make(chan net.Conn, 4)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			accepted <- c
		}
	}()

	dial := func() net.Conn {
		c, err := net.Dial("tcp", base.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = c.Close() })
		return c
	}
	wait := func(want bool) net.Conn {
		select {
		case c := <-accepted:
			if !want {
				t.Fatal("connection accepted beyond the limit")
			}
			return c
		case <-time.After(300 * time.Millisecond):
			if want {
				t.Fatal("expected connection to be accepted")
			}
			return nil
		}
	}

	// control: two connections within the limit are accepted
	dial()
	dial()
	c1 := wait(true)
	wait(true)

	// exploit: the third is not accepted while the others are open
	dial()
	wait(false)

	// closing a connection (twice, to check release is idempotent) frees one slot only
	_ = c1.Close()
	_ = c1.Close()
	wait(true)
	dial()
	wait(false)
}

func TestRetryable(t *testing.T) {
	if Retryable(net.ErrClosed) {
		t.Error("closed listener must not be retryable")
	}
	if !Retryable(errors.New("accept tcp: too many open files")) {
		t.Error("other accept errors must be retryable")
	}
}

func TestListenerCloseUnblocksAccept(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln := Listener("test", base, 1)

	// fill the only slot
	go func() { _, _ = net.Dial("tcp", base.Addr().String()) }()
	if _, err := ln.Accept(); err != nil {
		t.Fatal(err)
	}

	// this Accept waits for a slot until the listener is closed
	result := make(chan error, 1)
	go func() {
		_, err := ln.Accept()
		result <- err
	}()
	time.Sleep(100 * time.Millisecond)
	_ = ln.Close()

	select {
	case err := <-result:
		if !errors.Is(err, net.ErrClosed) {
			t.Fatalf("expected net.ErrClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not return after Close")
	}
}
