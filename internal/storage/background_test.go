package storage

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

// stopBackground must cancel the background goroutines and only return once they have finished, as they
// read the database and the configuration which InitDB then changes.
func TestStopBackgroundWaitsForGoroutines(t *testing.T) {
	var started, finished atomic.Bool

	goBackground(func(ctx context.Context) {
		started.Store(true)
		<-ctx.Done()
		time.Sleep(100 * time.Millisecond) // still working when asked to stop
		finished.Store(true)
	})

	for !started.Load() {
		time.Sleep(time.Millisecond)
	}

	stopBackground()

	if !finished.Load() {
		t.Error("stopBackground returned before the goroutine finished")
	}

	// control: a goroutine started afterwards is not cancelled by the earlier stop
	var cancelled atomic.Bool
	release := make(chan struct{})
	goBackground(func(ctx context.Context) {
		select {
		case <-ctx.Done():
			cancelled.Store(true)
		case <-release:
		}
	})
	time.Sleep(50 * time.Millisecond)
	if cancelled.Load() {
		t.Error("a goroutine started after stopBackground was cancelled")
	}
	close(release)
	stopBackground()
}

// The delayed stats broadcast must not outlive stopBackground and must not leave itself marked as pending,
// or no later broadcast would ever be sent.
func TestBroadcastMailboxStatsStops(t *testing.T) {
	setup("")

	BroadcastMailboxStats()
	if !bcStatsDelay.Load() {
		t.Fatal("a broadcast was not scheduled")
	}

	stopBackground()

	if bcStatsDelay.Load() {
		t.Error("a stopped broadcast is still marked as pending")
	}
}
