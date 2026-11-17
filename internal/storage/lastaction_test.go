package storage

import (
	"math"
	"sync"
	"testing"
	"time"
)

func TestLastAction(t *testing.T) {
	var l lastAction

	// control: a database that has never been used is idle for the longest possible time, which
	// is what the zero time.Time it replaced gave
	if got := l.since(); got != time.Duration(math.MaxInt64) {
		t.Fatalf("since() before any use = %v", got)
	}

	l.touch()
	if got := l.since(); got < 0 || got > time.Second {
		t.Fatalf("since() just after touch() = %v", got)
	}

	time.Sleep(50 * time.Millisecond)
	if got := l.since(); got < 50*time.Millisecond || got > 5*time.Second {
		t.Fatalf("since() 50ms after touch() = %v", got)
	}

	// touching again starts the idle time over
	l.touch()
	if got := l.since(); got > 40*time.Millisecond {
		t.Fatalf("since() just after a second touch() = %v", got)
	}
}

// Every database operation records its use from its own goroutine, while the cron job reads it.
// With a plain time.Time this is a data race, which the race detector reports.
func TestLastActionConcurrentUse(t *testing.T) {
	var (
		l    lastAction
		wg   sync.WaitGroup
		stop = make(chan struct{})
	)

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 2000 {
				l.touch()
			}
		}()
	}

	readers := make(chan time.Duration, 4)
	for range 4 {
		go func() {
			lowest := time.Duration(math.MaxInt64)
			for {
				select {
				case <-stop:
					readers <- lowest
					return
				default:
					if d := l.since(); d < lowest {
						lowest = d
					}
				}
			}
		}()
	}

	wg.Wait()
	close(stop)

	for range 4 {
		if d := <-readers; d < 0 {
			t.Fatalf("since() was negative: %v", d)
		}
	}
	if got := l.since(); got < 0 || got > time.Second {
		t.Fatalf("since() after the last touch() = %v", got)
	}
}
