package rspamd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const okBody = `{"score":1,"required_score":15,"action":"no action","symbols":{}}`

// countingServer answers every check with body (after delay), and records how many requests it
// received and the most that were being handled at once.
type countingServer struct {
	*httptest.Server
	hits, current, peak atomic.Int64
	status              atomic.Int64
	body                atomic.Value
}

func newCountingServer(t *testing.T, body string, delay time.Duration) *countingServer {
	t.Helper()

	s := &countingServer{}
	s.status.Store(http.StatusOK)
	s.body.Store(body)
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		cur := s.current.Add(1)
		defer s.current.Add(-1)
		for {
			p := s.peak.Load()
			if cur <= p || s.peak.CompareAndSwap(p, cur) {
				break
			}
		}
		time.Sleep(delay)
		w.WriteHeader(int(s.status.Load()))
		_, _ = w.Write([]byte(s.body.Load().(string)))
	}))
	t.Cleanup(s.Close)

	if _, err := SetService(s.URL); err != nil {
		t.Fatal(err)
	}
	resetCache()

	return s
}

func resetCache() {
	cacheMu.Lock()
	cache = make(map[string]cachedResult)
	cacheMu.Unlock()
}

func TestRound2dp(t *testing.T) {
	tests := []struct {
		name    string
		in      float64
		want    float64
		wantFin bool
	}{
		{"control: rounds down", 1.234, 1.23, true},
		{"control: rounds up", 1.236, 1.24, true},
		{"control: already two places", 12.5, 12.5, true},
		{"control: zero", 0, 0, true},
		{"control: negative", -3.456, -3.46, true},
		{"control: large", 1e15, 1e15, true},
		{"control: just below where it overflowed", 1.7e306, 1.7e306, true},
		{"exploit: where n*100 overflows", 1.8e306, 1.8e306, true},
		{"exploit: 1e307", 1e307, 1e307, true},
		{"exploit: largest float", math.MaxFloat64, math.MaxFloat64, true},
		{"exploit: negative", -1e307, -1e307, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := round2dp(tt.in)
			if math.IsInf(got, 0) || math.IsNaN(got) {
				t.Fatalf("round2dp(%g) = %g", tt.in, got)
			}
			if got != tt.want {
				t.Fatalf("round2dp(%g) = %g, want %g", tt.in, got, tt.want)
			}
		})
	}
}

// A huge score from the server must give a result which can be sent as JSON.
func TestCheckWithAHugeScore(t *testing.T) {
	newCountingServer(t, `{"score":1e307,"required_score":1e307,"action":"reject"}`, 0)

	res, err := Check([]byte("x"))
	if err != nil || res.Error != "" {
		t.Fatalf("%v %q", err, res.Error)
	}
	if _, err := json.Marshal(res); err != nil {
		t.Fatalf("the result cannot be encoded: %v", err)
	}
}

// Rspamd does not redirect, so a redirect is an error and is not followed. It used to be, sending
// the message (for 307 and 308) and always the password to wherever it pointed.
func TestRedirectsAreNotFollowed(t *testing.T) {
	var targetHits atomic.Int64
	var gotPassword atomic.Value
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		gotPassword.Store(r.Header.Get("Password"))
		_, _ = w.Write([]byte(okBody))
	}))
	defer target.Close()

	for _, code := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target.URL+"/elsewhere", code)
			}))
			defer front.Close()

			targetHits.Store(0)
			gotPassword.Store("")
			if _, err := SetService("http://s3cret@" + strings.TrimPrefix(front.URL, "http://")); err != nil {
				t.Fatal(err)
			}

			res, err := Check([]byte("Subject: private\r\n\r\nbody"))
			if err != nil {
				t.Fatal(err)
			}
			if res.Error == "" || !strings.Contains(res.Error, "redirect") {
				t.Fatalf("expected a redirect error, got %q", res.Error)
			}
			if targetHits.Load() != 0 || gotPassword.Load() != "" {
				t.Fatalf("the redirect was followed (%d requests), the other server got the password %q", targetHits.Load(), gotPassword.Load())
			}
			if err := Ping(); err == nil {
				t.Fatal("ping followed a redirect")
			}
		})
	}

	// control: a server which does not redirect works, and gets the password
	if _, err := SetService("http://s3cret@" + strings.TrimPrefix(target.URL, "http://")); err != nil {
		t.Fatal(err)
	}
	targetHits.Store(0)
	if res, _ := Check([]byte("x")); res.Error != "" || gotPassword.Load() != "s3cret" {
		t.Fatalf("control failed: %+v, password %q", res, gotPassword.Load())
	}
}

func TestSetServiceAddresses(t *testing.T) {
	valid := []struct{ in, want string }{
		{"host:11334", "http://host:11334"},
		{"host", "http://host"},
		{"http://host:11334", "http://host:11334"},
		{"https://host:11334", "https://host:11334"},
		{"HTTPS://host", "https://host"},
		{" http://host:11334/ ", "http://host:11334"},
		{"http://host:11334/rspamd//", "http://host:11334/rspamd"},
		{"[::1]:11334", "http://[::1]:11334"},
		{"http://[::1]:11334", "http://[::1]:11334"},
		{"127.0.0.1:11334", "http://127.0.0.1:11334"},
		{"https://:pass@host:11334", "https://host:11334"},
		{"https://pass@host:11334/path", "https://host:11334/path"},
		{"http://host:65535", "http://host:65535"},
		{"http://host:1", "http://host:1"},
	}
	for _, tt := range valid {
		shown, err := SetService(tt.in)
		if err != nil || shown != tt.want {
			t.Errorf("SetService(%q) = %q, %v, want %q", tt.in, shown, err, tt.want)
		}
	}

	invalid := []struct{ name, in string }{
		{"query", "http://host:11334?x=1"},
		{"empty query", "http://host:11334?"},
		{"fragment", "http://host:11334#frag"},
		{"other scheme", "ftp://host:1"},
		{"file scheme", "file:///etc/passwd"},
		{"no host", "http://"},
		{"no host with a port", "http://:11334"},
		{"only a separator", "://"},
		{"port zero", "http://host:0"},
		{"port too large", "http://host:99999"},
		{"port not a number", "http://host:abc"},
		{"empty", ""},
		{"spaces", "  "},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			// start from a known good service
			_, _ = SetService("http://:keep@good:1")

			shown, err := SetService(tt.in)
			if err == nil || shown != "" {
				t.Fatalf("SetService(%q) = %q, %v, expected an error", tt.in, shown, err)
			}
			// the previous service, and its password, are not changed by a rejected address
			if baseURL != "http://good:1" || password != "keep" {
				t.Fatalf("a rejected address changed the service to %q (password %q)", baseURL, password)
			}
		})
	}
}

func TestCheckMessageLimitsConcurrentChecks(t *testing.T) {
	srv := newCountingServer(t, okBody, 100*time.Millisecond)

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

	// control: without the limit all of them reach the server together
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
		t.Fatalf("%d checks reached the server at once, want at most %d", got, maxConcurrentChecks)
	}
	if got := srv.hits.Load(); got != 16 {
		t.Fatalf("the server received %d requests, want 16", got)
	}
}

func TestCheckMessageCachesResults(t *testing.T) {
	srv := newCountingServer(t, okBody, 0)

	var loads atomic.Int64
	load := func() ([]byte, error) { loads.Add(1); return []byte("x"), nil }

	// control: different messages are each checked
	for i := range 5 {
		if _, err := CheckMessage(context.Background(), fmt.Sprintf("other-%d", i), load); err != nil {
			t.Fatal(err)
		}
	}
	if srv.hits.Load() != 5 || loads.Load() != 5 {
		t.Fatalf("5 different messages: %d requests, %d loads", srv.hits.Load(), loads.Load())
	}

	// exploit: the same message asked for again and again is only loaded and sent once
	srv.hits.Store(0)
	loads.Store(0)
	for range 20 {
		res, err := CheckMessage(context.Background(), "same", load)
		if err != nil || res.Score != 1 {
			t.Fatalf("%v %+v", err, res)
		}
	}
	if srv.hits.Load() != 1 || loads.Load() != 1 {
		t.Fatalf("one message asked for 20 times: %d requests, %d loads", srv.hits.Load(), loads.Load())
	}
}

func TestCheckMessageDoesNotCacheFailures(t *testing.T) {
	srv := newCountingServer(t, `{"error":"temporarily unavailable"}`, 0)
	srv.status.Store(http.StatusServiceUnavailable)

	load := func() ([]byte, error) { return []byte("x"), nil }

	res, err := CheckMessage(context.Background(), "m", load)
	if err != nil || res.Error == "" {
		t.Fatalf("expected a result error, got %v %+v", err, res)
	}

	// Rspamd recovers, and the same message is checked again rather than failing from the cache
	srv.status.Store(http.StatusOK)
	srv.body.Store(okBody)
	res, err = CheckMessage(context.Background(), "m", load)
	if err != nil || res.Error != "" || srv.hits.Load() != 2 {
		t.Fatalf("the failure was cached: %v %+v (%d requests)", err, res, srv.hits.Load())
	}

	// and now the good result is the one which is kept
	if _, err := CheckMessage(context.Background(), "m", load); err != nil || srv.hits.Load() != 2 {
		t.Fatalf("the good result was not cached (%d requests)", srv.hits.Load())
	}
}

func TestCheckMessageLoadErrors(t *testing.T) {
	srv := newCountingServer(t, okBody, 0)

	wantErr := errors.New("no such message")
	_, err := CheckMessage(context.Background(), "gone", func() ([]byte, error) { return nil, wantErr })
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v", err)
	}
	if srv.hits.Load() != 0 {
		t.Fatal("a message which could not be loaded was sent to Rspamd")
	}
}

func TestCheckMessageResultsExpire(t *testing.T) {
	srv := newCountingServer(t, okBody, 0)

	orig := resultCacheTTL
	resultCacheTTL = 150 * time.Millisecond
	t.Cleanup(func() { resultCacheTTL = orig })

	load := func() ([]byte, error) { return []byte("x"), nil }
	_, _ = CheckMessage(context.Background(), "m", load)
	_, _ = CheckMessage(context.Background(), "m", load)
	if srv.hits.Load() != 1 {
		t.Fatalf("%d requests before the result expired, want 1", srv.hits.Load())
	}

	time.Sleep(300 * time.Millisecond)
	_, _ = CheckMessage(context.Background(), "m", load)
	if srv.hits.Load() != 2 {
		t.Fatalf("%d requests after the result expired, want 2", srv.hits.Load())
	}
}

// A request waiting for its turn stops waiting when the client goes away, without loading the message.
func TestCheckMessageStopsWaitingWhenCancelled(t *testing.T) {
	newCountingServer(t, okBody, 600*time.Millisecond)

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
