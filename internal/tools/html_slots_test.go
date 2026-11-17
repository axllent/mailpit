package tools

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// endsAfterTheFirstCheck is a context which is not over the first time it is asked, and is over (with its
// channel closed) from then on: the request ending after the first check, when a slot is free.
type endsAfterTheFirstCheck struct {
	context.Context
	asked atomic.Int32
	done  chan struct{}
}

func newEndsAfterTheFirstCheck() *endsAfterTheFirstCheck {
	done := make(chan struct{})
	close(done)

	return &endsAfterTheFirstCheck{Context: context.Background(), done: done}
}

func (c *endsAfterTheFirstCheck) Done() <-chan struct{} { return c.done }

func (c *endsAfterTheFirstCheck) Err() error {
	if c.asked.Add(1) == 1 {
		return nil
	}

	return context.Canceled
}

func smallSlots(t *testing.T, n int) {
	t.Helper()

	orig := htmlSlots
	htmlSlots = make(chan struct{}, n)
	t.Cleanup(func() { htmlSlots = orig })
}

func TestAcquireHTMLSlot(t *testing.T) {
	smallSlots(t, 2)

	// control: up to the limit is taken at once, and given back
	var releases []func()
	for range 2 {
		release, err := AcquireHTMLSlot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		releases = append(releases, release)
	}
	if len(htmlSlots) != 2 {
		t.Fatalf("%d slots held, want 2", len(htmlSlots))
	}

	// exploit: with every slot in use another waits, and gives up when the request is over
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if release, err := AcquireHTMLSlot(ctx); !errors.Is(err, context.DeadlineExceeded) || release != nil {
		t.Fatalf("got a slot %v, error %v", release != nil, err)
	}
	if d := time.Since(start); d < 100*time.Millisecond || d > 2*time.Second {
		t.Errorf("waited %v", d)
	}
	if len(htmlSlots) != 2 {
		t.Errorf("%d slots held after giving up, want 2", len(htmlSlots))
	}

	// control: once one is given back, another can be taken
	releases[0]()
	release, err := AcquireHTMLSlot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release()
	releases[1]()
	if len(htmlSlots) != 0 {
		t.Errorf("%d slots held, want none", len(htmlSlots))
	}
}

// A request which is over takes no slot, even when one is free. Both being ready is where a select
// could choose either, so it is tried many times.
func TestAcquireHTMLSlotForARequestWhichIsOver(t *testing.T) {
	smallSlots(t, 2)

	over, cancel := context.WithCancel(context.Background())
	cancel()
	for range 100 {
		if release, err := AcquireHTMLSlot(over); !errors.Is(err, context.Canceled) || release != nil {
			t.Fatalf("got a slot %v, error %v", release != nil, err)
		}
	}

	for range 300 {
		if release, err := AcquireHTMLSlot(newEndsAfterTheFirstCheck()); !errors.Is(err, context.Canceled) || release != nil {
			t.Fatalf("got a slot %v, error %v", release != nil, err)
		}
	}

	if len(htmlSlots) != 0 {
		t.Fatalf("%d slots held", len(htmlSlots))
	}
}

// Removing the redirects waits for its turn, and stops waiting when the request is over.
func TestStripHTMLRedirectsStopsWaitingWhenTheRequestIsOver(t *testing.T) {
	smallSlots(t, 1)
	htmlSlots <- struct{}{}

	withMeta := `<meta charset="utf-8"><p>x</p>`

	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := StripHTMLRedirects(ctx, withMeta); result <- err }()

	select {
	case err := <-result:
		t.Fatalf("returned while all the slots were in use: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("error %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("still waiting after the request was over")
	}

	<-htmlSlots
	if len(htmlSlots) != 0 {
		t.Errorf("%d slots held", len(htmlSlots))
	}

	// control: a request which is not over is done once there is a slot
	if out, err := StripHTMLRedirects(context.Background(), withMeta); err != nil || out != withMeta {
		t.Errorf("%q, %v", out, err)
	}
}
