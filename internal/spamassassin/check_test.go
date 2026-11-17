package spamassassin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// countingSpamd answers every check with reply (after delay), and records how many checks it received and the
// most that were being handled at once. It is the service for the test.
type countingSpamd struct {
	hits, current, peak atomic.Int64
	reply               atomic.Value
}

func newCountingSpamd(t *testing.T, reply string, delay time.Duration) *countingSpamd {
	t.Helper()

	s := &countingSpamd{}
	s.reply.Store(reply)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = c.Close() }()

				s.hits.Add(1)
				cur := s.current.Add(1)
				defer s.current.Add(-1)
				for {
					p := s.peak.Load()
					if cur <= p || s.peak.CompareAndSwap(p, cur) {
						break
					}
				}

				_, _ = io.Copy(io.Discard, c)
				time.Sleep(delay)
				_, _ = io.WriteString(c, s.reply.Load().(string))
			}()
		}
	}()

	use(t, ln.Addr().String())
	resetCache()
	t.Cleanup(resetCache)

	return s
}

func resetCache() {
	cacheMu.Lock()
	cache = make(map[string]cachedResult)
	cacheMu.Unlock()
}

func TestCheckMessageLimitsConcurrentChecks(t *testing.T) {
	srv := newCountingSpamd(t, report, 100*time.Millisecond)

	run := func(f func(i int)) {
		var wg sync.WaitGroup
		for i := range 16 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				f(i)
			}()
		}
		wg.Wait()
	}

	// control: without the limit all of them reach the service together
	run(func(int) { _, _ = Check([]byte("x")) })
	if srv.peak.Load() <= maxConcurrentChecks {
		t.Fatalf("the control only reached %d at once, so it shows nothing", srv.peak.Load())
	}

	// exploit: through CheckMessage, no more than the limit at once, however many ask
	srv.peak.Store(0)
	srv.hits.Store(0)
	run(func(i int) {
		res, err := CheckMessage(context.Background(), fmt.Sprintf("id-%d", i), func() ([]byte, error) { return []byte("x"), nil })
		if err != nil || res.Error != "" {
			t.Errorf("check %d: %v %q", i, err, res.Error)
		}
	})
	if got := srv.peak.Load(); got > maxConcurrentChecks || got == 0 {
		t.Fatalf("%d checks reached the service at once, want at most %d", got, maxConcurrentChecks)
	}
	if got := srv.hits.Load(); got != 16 {
		t.Fatalf("the service received %d checks, want 16", got)
	}
}

func TestCheckMessageCachesResults(t *testing.T) {
	srv := newCountingSpamd(t, report, 0)

	var loads atomic.Int64
	load := func() ([]byte, error) { loads.Add(1); return []byte("x"), nil }

	// control: different messages are each checked
	for i := range 5 {
		if _, err := CheckMessage(context.Background(), fmt.Sprintf("other-%d", i), load); err != nil {
			t.Fatal(err)
		}
	}
	if srv.hits.Load() != 5 || loads.Load() != 5 {
		t.Fatalf("5 different messages: %d checks, %d loads", srv.hits.Load(), loads.Load())
	}

	// exploit: the same message asked for again and again is only loaded and sent once
	srv.hits.Store(0)
	loads.Store(0)
	for range 20 {
		res, err := CheckMessage(context.Background(), "same", load)
		if err != nil || res.Score != 0.7 {
			t.Fatalf("%v %+v", err, res)
		}
	}
	if srv.hits.Load() != 1 || loads.Load() != 1 {
		t.Fatalf("one message asked for 20 times: %d checks, %d loads", srv.hits.Load(), loads.Load())
	}
}

func TestCheckMessageDoesNotCacheFailures(t *testing.T) {
	srv := newCountingSpamd(t, "SPAMD/1.1 5 EX_UNAVAILABLE\r\n\r\n", 0)
	load := func() ([]byte, error) { return []byte("x"), nil }

	res, err := CheckMessage(context.Background(), "m", load)
	if err != nil || res.Error == "" {
		t.Fatalf("expected a result error, got %v %+v", err, res)
	}

	// the service recovers, and the same message is checked again rather than failing from the cache
	srv.reply.Store(report)
	res, err = CheckMessage(context.Background(), "m", load)
	if err != nil || res.Error != "" || srv.hits.Load() != 2 {
		t.Fatalf("the failure was cached: %v %+v (%d checks)", err, res, srv.hits.Load())
	}

	// and now the good result is the one which is kept
	if _, err := CheckMessage(context.Background(), "m", load); err != nil || srv.hits.Load() != 2 {
		t.Fatalf("the good result was not cached (%d checks)", srv.hits.Load())
	}
}

func TestCheckMessageLoadErrors(t *testing.T) {
	srv := newCountingSpamd(t, report, 0)

	wantErr := errors.New("no such message")
	_, err := CheckMessage(context.Background(), "gone", func() ([]byte, error) { return nil, wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v", err)
	}
	if srv.hits.Load() != 0 {
		t.Fatal("a message which could not be loaded was sent to SpamAssassin")
	}
}

func TestCheckMessageResultsExpire(t *testing.T) {
	srv := newCountingSpamd(t, report, 0)

	orig := resultCacheTTL
	resultCacheTTL = 150 * time.Millisecond
	t.Cleanup(func() { resultCacheTTL = orig })

	load := func() ([]byte, error) { return []byte("x"), nil }
	_, _ = CheckMessage(context.Background(), "m", load)
	_, _ = CheckMessage(context.Background(), "m", load)
	if srv.hits.Load() != 1 {
		t.Fatalf("%d checks before the result expired, want 1", srv.hits.Load())
	}

	time.Sleep(300 * time.Millisecond)
	_, _ = CheckMessage(context.Background(), "m", load)
	if srv.hits.Load() != 2 {
		t.Fatalf("%d checks after the result expired, want 2", srv.hits.Load())
	}
}

// A request waiting for its turn stops waiting when the client goes away, without loading the message.
func TestCheckMessageStopsWaitingWhenCancelled(t *testing.T) {
	newCountingSpamd(t, report, 600*time.Millisecond)

	var wg sync.WaitGroup
	for i := range maxConcurrentChecks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = CheckMessage(context.Background(), fmt.Sprintf("busy-%d", i), func() ([]byte, error) { return []byte("x"), nil })
		}()
	}
	time.Sleep(150 * time.Millisecond) // every slot is now in use

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	loaded := false
	start := time.Now()
	_, err := CheckMessage(ctx, "waiting", func() ([]byte, error) { loaded = true; return []byte("x"), nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if loaded {
		t.Fatal("the message was loaded without a free slot")
	}
	if d := time.Since(start); d > 400*time.Millisecond {
		t.Fatalf("took %v to give up", d)
	}

	wg.Wait()
}

func TestResultCacheIsBounded(t *testing.T) {
	resetCache()
	t.Cleanup(resetCache)

	// exploit: far more messages than the limit
	for i := range maxCachedResults + 500 {
		storeResult(fmt.Sprintf("id-%d", i), Result{Score: 1})
	}
	cacheMu.Lock()
	n := len(cache)
	cacheMu.Unlock()
	if n != maxCachedResults {
		t.Fatalf("%d results cached, want %d", n, maxCachedResults)
	}

	// control: once they have expired there is room again
	cacheMu.Lock()
	for k, v := range cache {
		v.expires = time.Now().Add(-time.Second)
		cache[k] = v
	}
	cacheMu.Unlock()
	storeResult("fresh", Result{Score: 2})
	if r, ok := cachedResultFor("fresh"); !ok || r.Score != 2 {
		t.Fatal("a result was not kept once old ones had expired")
	}
}

// With no service defined the check is an error, which is not kept.
func TestCheckMessageWithoutAService(t *testing.T) {
	use(t, "")
	resetCache()
	t.Cleanup(resetCache)

	if _, err := CheckMessage(context.Background(), "m", func() ([]byte, error) { return []byte("x"), nil }); err == nil {
		t.Fatal("expected an error without a service")
	}
	if _, ok := cachedResultFor("m"); ok {
		t.Fatal("an error was cached")
	}
}
