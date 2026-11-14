package linkcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
)

// allowLocalTestServer lets the checker request the local test server, and restores the
// settings (and a clean cache and registry) afterwards.
func allowLocalTestServer(t *testing.T, rateLimit bool) {
	t.Helper()

	origAllow, origDisable := config.AllowInternalHTTPRequests, config.DisableLinkCheckRateLimit
	config.AllowInternalHTTPRequests = true
	config.DisableLinkCheckRateLimit = !rateLimit
	linkCache = newResultCache()
	domainRegistry = newRegistry()

	t.Cleanup(func() {
		config.AllowInternalHTTPRequests, config.DisableLinkCheckRateLimit = origAllow, origDisable
		linkCache = newResultCache()
		domainRegistry = newRegistry()
	})
}

func TestDoHeadStatus(t *testing.T) {
	allowLocalTestServer(t, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(http.StatusOK)
		case "/missing":
			w.WriteHeader(http.StatusNotFound)
		case "/moved":
			http.Redirect(w, r, "/ok", http.StatusMovedPermanently)
		}
	}))
	defer srv.Close()

	tests := []struct {
		path   string
		follow bool
		want   int
	}{
		{"/ok", false, 200},
		{"/missing", false, 404},
		{"/moved", false, 301},
		{"/moved", true, 200},
	}

	for _, tt := range tests {
		got, err := doHead(context.Background(), srv.URL+tt.path, tt.follow)
		if err != nil || got != tt.want {
			t.Errorf("%s follow=%v: got %d, %v, want %d", tt.path, tt.follow, got, err, tt.want)
		}
	}
}

// A server that keeps connections open must not be able to hold on to a file descriptor
// and goroutines for every link checked. Without closing them, 100 checks left over 200
// goroutines behind.
func TestDoHeadReleasesConnections(t *testing.T) {
	allowLocalTestServer(t, false)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // keep-alive is the default, the server never closes the connection
	}))
	defer srv.Close()

	// warm up so the server's own goroutines are not counted
	if _, err := doHead(context.Background(), srv.URL+"/warmup", false); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	before := runtime.NumGoroutine()

	for i := range 100 {
		if code, err := doHead(context.Background(), fmt.Sprintf("%s/%d", srv.URL, i), false); err != nil || code != 200 {
			t.Fatalf("check %d: %d, %v", i, code, err)
		}
	}

	// closed connections take a moment to wind down, so poll rather than sleep
	var after int
	for range 50 {
		runtime.GC()
		if after = runtime.NumGoroutine(); after <= before+10 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("goroutines grew from %d to %d after 100 checks", before, after)
}

// A check that was cancelled says nothing about the link, so it must not be cached.
func TestCancelledCheckIsNotCached(t *testing.T) {
	for _, tt := range []struct {
		name        string
		cancelFirst bool
		wantHits    int64 // requests received by the server over two consecutive checks
	}{
		// control: a completed check is cached, so the second check does not request again
		{"completed check is cached", false, 1},
		// exploit: a cancelled check is not cached, so the second check requests again and succeeds
		{"cancelled check is not cached", true, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			allowLocalTestServer(t, true)

			var hits atomic.Int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// the first request hangs until the client gives up (when it is going to), later ones answer
				if hits.Add(1) == 1 && tt.cancelFirst {
					<-r.Context().Done()
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			ctx := context.Background()
			if tt.cancelFirst {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 150*time.Millisecond)
				defer cancel()
			}

			first := getHTTPStatuses(ctx, []string{srv.URL}, false)
			if tt.cancelFirst && first[0].StatusCode != 0 {
				t.Fatalf("expected the cancelled check to fail, got %+v", first[0])
			}

			second := getHTTPStatuses(context.Background(), []string{srv.URL}, false)
			if second[0].StatusCode != http.StatusOK {
				t.Fatalf("second check = %+v, want 200 (a cancelled result was cached)", second[0])
			}

			if got := hits.Load(); got != tt.wantHits {
				t.Fatalf("server received %d requests, want %d", got, tt.wantHits)
			}
		})
	}
}

func TestOverLongLinks(t *testing.T) {
	allowLocalTestServer(t, false)

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// pad a link on the test server to an exact total length
	linkOfLength := func(n int) string {
		prefix := srv.URL + "/"
		return prefix + strings.Repeat("a", n-len(prefix))
	}

	tests := []struct {
		name       string
		link       string
		wantStatus string
		wantHits   int64
	}{
		{"control: exactly at the limit is requested", linkOfLength(maxLinkLength), "OK", 1},
		{"exploit: one byte over the limit is not requested", linkOfLength(maxLinkLength + 1), "URL too long to check", 0},
		{"exploit: 5 MiB link is not requested", linkOfLength(5 << 20), "URL too long to check", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits.Store(0)

			// through the collector, as RunTests does
			c := &linkCollector{seen: make(map[string]bool)}
			c.add(tt.link)

			if len(c.links) != 1 || len(c.links[0]) > maxLinkLength+1 {
				t.Fatalf("collector kept %d links, first of length %d", len(c.links), len(c.links[0]))
			}

			res := getHTTPStatuses(context.Background(), c.links, false)

			if res[0].Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", res[0].Status, tt.wantStatus)
			}
			if len(res[0].URL) > maxLinkLength+3 {
				t.Fatalf("returned URL is %d bytes", len(res[0].URL))
			}
			if got := hits.Load(); got != tt.wantHits {
				t.Fatalf("server received %d requests, want %d", got, tt.wantHits)
			}
		})
	}
}

func TestDisplayLink(t *testing.T) {
	short := "http://example.com/" + strings.Repeat("a", 10)
	if got := displayLink(short); got != short {
		t.Fatalf("short link changed: %q", got)
	}

	// a multi-byte character straddling the cut must not be left half-cut
	long := "http://example.com/" + strings.Repeat("é", maxLinkLength)
	got := strings.TrimSuffix(displayLink(long), "...")
	if !strings.HasSuffix(displayLink(long), "...") || len(got) > maxLinkLength {
		t.Fatalf("unexpected result of length %d", len(got))
	}
	for _, r := range got {
		if r == '�' {
			t.Fatal("result contains a broken character")
		}
	}
}

// Invalid links are rejected before using up the shared rate limit bucket for unparseable hosts.
func TestInvalidLinksDoNotUseRateLimiter(t *testing.T) {
	allowLocalTestServer(t, true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()

	res := getHTTPStatuses(context.Background(), []string{"http://link with spaces", "http://exa mple.com/x"}, false)
	for _, r := range res {
		if r.StatusCode != 0 || r.Status != "Invalid URL" {
			t.Fatalf("got %+v", r)
		}
	}
	if _, ok := domainRegistry.entries[""]; ok {
		t.Fatal("invalid links used a rate limit bucket")
	}
	if _, ok := linkCache.get("http://link with spaces"); ok {
		t.Fatal("an invalid link was cached")
	}

	// control: a valid link does use (and is cached by) the limiter
	if r := getHTTPStatuses(context.Background(), []string{srv.URL}, false)[0]; r.StatusCode != 200 {
		t.Fatalf("got %+v", r)
	}
	if len(domainRegistry.entries) != 1 {
		t.Fatalf("expected the valid link to use one rate limit bucket, found %d", len(domainRegistry.entries))
	}
}

func TestScanLimit(t *testing.T) {
	orig := maxScanBytes
	maxScanBytes = 1000
	t.Cleanup(func() { maxScanBytes = orig })

	before := "http://before.example.com/a"
	after := "http://after.example.com/b"
	pad := strings.Repeat(" ", maxScanBytes)

	tests := []struct {
		name string
		msg  storage.Message
		want []string
	}{
		{"control: text links inside the limit", storage.Message{Text: before + pad + "x"}, []string{before}},
		{"exploit: text link after the limit is ignored", storage.Message{Text: pad + after}, nil},
		{"control: html links inside the limit", storage.Message{HTML: `<a href="` + before + `">x</a>` + pad}, []string{before}},
		{"exploit: html link after the limit is ignored", storage.Message{HTML: pad + `<a href="` + after + `">x</a>`}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &linkCollector{seen: make(map[string]bool)}
			extractHTMLLinks(&tt.msg, c)
			extractTextLinks(&tt.msg, c)

			if strings.Join(c.links, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("links = %v, want %v", c.links, tt.want)
			}
		})
	}
}

func TestHTTPErrorSummary(t *testing.T) {
	tests := []struct{ in, want string }{
		{`Head "http://x": dial tcp 1.2.3.4:80: connect: connection refused`, "connection refused"},
		{`no colon here`, "no colon here"},
		{`a: b`, "b"},
	}

	for _, tt := range tests {
		if got := httpErrorSummary(fmt.Errorf("%s", tt.in)); got != tt.want {
			t.Errorf("httpErrorSummary(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
