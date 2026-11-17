package smtpd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// counting answers lookups with the name of the address, and counts how many it was asked for.
func counting(calls *atomic.Int32) func(context.Context, string) ([]string, error) {
	return func(_ context.Context, ip string) ([]string, error) {
		n := calls.Add(1)
		return []string{fmt.Sprintf("host%d.example", n)}, nil
	}
}

// A client which connects again is not looked up again. A different address (the control) is.
func TestHostCacheRemembersAHost(t *testing.T) {
	var calls atomic.Int32
	addr, _, messages := rdnsServer(t, time.Second, counting(&calls))

	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))

	if calls.Load() != 1 {
		t.Fatalf("%d lookups for two connections from one address, want 1", calls.Load())
	}
	for _, m := range messages() {
		if line, _, _ := strings.Cut(m, "\r\n"); line != "Received: from client (host1.example [127.0.0.2])" {
			t.Fatalf("Received = %q", line)
		}
	}

	// control: another address is looked up (and is not given the name of the first)
	sendMessage(t, dialFrom(t, "127.0.0.3", addr))

	if calls.Load() != 2 {
		t.Fatalf("%d lookups after a third connection from another address, want 2", calls.Load())
	}
	msgs := messages()
	if line, _, _ := strings.Cut(msgs[len(msgs)-1], "\r\n"); line != "Received: from client (host2.example [127.0.0.3])" {
		t.Fatalf("Received = %q", line)
	}
}

// A lookup which took too long is remembered, so the next connection from the same address does not wait for the
// timeout again. This is what made a client with a slow lookup slow for every connection.
func TestHostCacheRemembersATimeout(t *testing.T) {
	var calls atomic.Int32
	slow := func(ctx context.Context, ip string) ([]string, error) {
		calls.Add(1)
		return hangs(ctx, ip)
	}
	addr, _, _ := rdnsServer(t, 400*time.Millisecond, slow)

	start := time.Now()
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	first := time.Since(start)

	start = time.Now()
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	second := time.Since(start)

	if first < 350*time.Millisecond {
		t.Fatalf("the first message took %v, it should have waited for the lookup", first)
	}
	if second > 200*time.Millisecond || calls.Load() != 1 {
		t.Fatalf("the second message took %v with %d lookups, it should not have waited", second, calls.Load())
	}

	// control: another address waits for its own lookup
	start = time.Now()
	sendMessage(t, dialFrom(t, "127.0.0.3", addr))
	if took := time.Since(start); took < 350*time.Millisecond || calls.Load() != 2 {
		t.Fatalf("another address took %v with %d lookups, it should have waited", took, calls.Load())
	}
}

// Names are remembered for the ttl, failures for the (shorter) failure ttl.
func TestHostCacheExpires(t *testing.T) {
	var found, missing atomic.Int32
	lookup := func(_ context.Context, ip string) ([]string, error) {
		if ip == "127.0.0.2" {
			found.Add(1)
			return []string{"found.example"}, nil
		}
		missing.Add(1)
		return nil, fmt.Errorf("no such host")
	}

	addr, _, _ := rdnsServer(t, time.Second, lookup, func(srv *Server) {
		srv.hosts.ttl = time.Hour
		srv.hosts.failureTTL = 150 * time.Millisecond
	})

	for range 2 {
		sendMessage(t, dialFrom(t, "127.0.0.2", addr))
		sendMessage(t, dialFrom(t, "127.0.0.3", addr))
	}
	if found.Load() != 1 || missing.Load() != 1 {
		t.Fatalf("found %d, missing %d lookups, want 1 and 1", found.Load(), missing.Load())
	}

	time.Sleep(250 * time.Millisecond)

	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	sendMessage(t, dialFrom(t, "127.0.0.3", addr))

	// the failure has expired and is looked up again, the name has not
	if found.Load() != 1 || missing.Load() != 2 {
		t.Fatalf("after the failure ttl: found %d, missing %d lookups, want 1 and 2", found.Load(), missing.Load())
	}

	// a name which has expired is looked up again too
	addr, _, messages := rdnsServer(t, time.Second, counting(&found), func(srv *Server) { srv.hosts.ttl = 100 * time.Millisecond })
	found.Store(0)
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))
	time.Sleep(200 * time.Millisecond)
	sendMessage(t, dialFrom(t, "127.0.0.2", addr))

	msgs := messages()
	if line, _, _ := strings.Cut(msgs[1], "\r\n"); found.Load() != 2 || line != "Received: from client (host2.example [127.0.0.2])" {
		t.Fatalf("after the ttl: %d lookups, Received = %q", found.Load(), line)
	}
}

// The cache has a limit. A new address is not remembered when it is full, and what is there is kept (the control),
// unless it has expired.
func TestHostCacheLimit(t *testing.T) {
	c := &hostCache{max: 3, ttl: time.Hour, failureTTL: 50 * time.Millisecond}

	for i := range 5 {
		c.put("10.0.0."+strconv.Itoa(i), "h", true)
	}
	if len(c.entries) != 3 {
		t.Fatalf("%d entries, the limit is 3", len(c.entries))
	}
	// control: the first ones are still there, the later ones were not remembered
	for i, want := range []bool{true, true, true, false, false} {
		if _, ok := c.get("10.0.0." + strconv.Itoa(i)); ok != want {
			t.Fatalf("10.0.0.%d remembered = %v, want %v", i, ok, want)
		}
	}

	// an address which is there can be updated when it is full
	c.put("10.0.0.1", "other", true)
	if h, _ := c.get("10.0.0.1"); h != "other" {
		t.Fatalf("host = %q, want updated", h)
	}

	// expired entries make room
	c = &hostCache{max: 3, ttl: time.Hour, failureTTL: 50 * time.Millisecond}
	for i := range 3 {
		c.put("10.0.1."+strconv.Itoa(i), "", false)
	}
	time.Sleep(100 * time.Millisecond)
	c.put("10.0.1.9", "new", true)
	if h, ok := c.get("10.0.1.9"); !ok || h != "new" {
		t.Fatalf("a new address was not remembered after the others expired (%q, %v)", h, ok)
	}
	if len(c.entries) > 3 {
		t.Fatalf("%d entries, the limit is 3", len(c.entries))
	}
}

// The cache is used by all sessions at once.
func TestHostCacheConcurrent(t *testing.T) {
	c := &hostCache{max: 50, ttl: time.Millisecond, failureTTL: time.Millisecond}

	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 2000 {
				ip := "10.0." + strconv.Itoa(g) + "." + strconv.Itoa(i%100)
				c.put(ip, "h", i%2 == 0)
				c.get(ip)
			}
		}()
	}
	wg.Wait()

	if len(c.entries) > 50 {
		t.Fatalf("%d entries, the limit is 50", len(c.entries))
	}
}
