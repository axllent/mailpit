package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// cssServer serves the stylesheet at /s.css, with the content type given.
func cssServer(t *testing.T, contentType string, css func(w http.ResponseWriter)) string {
	t.Helper()

	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		css(w)
	}))
	t.Cleanup(srv.Close)

	return srv.URL + "/s.css"
}

func uniqueURLs(n int) string {
	var sb strings.Builder
	for i := range n {
		fmt.Fprintf(&sb, "a{background:url(/i%d.png)}\n", i)
	}

	return sb.String()
}

// proxied is how many urls of the stylesheet were replaced with the proxy address.
func proxied(body string) int {
	return strings.Count(body, "proxy?data=")
}

func withCSSLimits(t *testing.T, assets int) {
	t.Helper()

	orig := maxCSSAssets
	maxCSSAssets = assets
	t.Cleanup(func() { maxCSSAssets = orig })
}

// A stylesheet which is too large is refused, where a smaller one (the control) is rewritten. It used to be read and
// rewritten up to 50 MiB, which took seconds and more than a gigabyte for one request.
func TestProxyRefusesLargeCSS(t *testing.T) {
	withCSSLimits(t, 1000000)

	large := uniqueURLs(maxProxyCSSSize/20 + 100) // over the limit
	small := uniqueURLs(1000)
	if len(large) <= maxProxyCSSSize || len(small) >= maxProxyCSSSize {
		t.Fatalf("test data: large %d, small %d, limit %d", len(large), len(small), maxProxyCSSSize)
	}

	for name, tc := range map[string]struct {
		css  string
		code int
	}{
		"control: within the limit": {small, http.StatusOK},
		"over the limit":            {large, http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			uri := cssServer(t, "text/css", func(w http.ResponseWriter) { _, _ = w.Write([]byte(tc.css)) })
			const id = "large-css"
			seedAssets(id)
			recordAsset(id, uri)

			w := proxyRequest(id, uri)
			if w.Code != tc.code {
				t.Fatalf("status %d, want %d", w.Code, tc.code)
			}
			if tc.code == http.StatusOK && proxied(w.Body.String()) != 1000 {
				t.Fatalf("%d urls were rewritten, want 1000", proxied(w.Body.String()))
			}
		})
	}
}

// A stylesheet which never ends is not read for ever (or into memory): it is refused once it is over the limit.
func TestProxyStopsReadingEndlessCSS(t *testing.T) {
	uri := cssServer(t, "text/css", func(w http.ResponseWriter) {
		chunk := []byte(strings.Repeat("a{color:red}\n", 1000))
		for range 100000 { // 1.3 GB, if it was read to the end
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	})
	const id = "endless-css"
	seedAssets(id)
	recordAsset(id, uri)

	runtime.GC()
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	start := time.Now()
	w := proxyRequest(id, uri)
	runtime.ReadMemStats(&m1)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want the stylesheet to be refused", w.Code)
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Fatalf("took %v", took)
	}
	if alloc := (m1.TotalAlloc - m0.TotalAlloc) >> 20; alloc > 64 {
		t.Fatalf("%d MiB were allocated for a refused stylesheet", alloc)
	}
}

// A stylesheet can only add so many urls to a message: the rest are left without url(), so they are not fetched, and
// the assets of the message do not grow with the stylesheet. With room (the control) all are added.
func TestCSSAssetsAreLimited(t *testing.T) {
	css := uniqueURLs(5000)

	for name, tc := range map[string]struct{ limit, want int }{
		"limited":          {100, 99}, // the stylesheet itself was recorded first, and counts as one
		"control: no need": {10000, 5000},
	} {
		t.Run(name, func(t *testing.T) {
			withCSSLimits(t, tc.limit)

			uri := cssServer(t, "text/css", func(w http.ResponseWriter) { _, _ = w.Write([]byte(css)) })
			const id = "limited-css"
			seedAssets(id)
			recordAsset(id, uri)

			w := proxyRequest(id, uri)
			if w.Code != http.StatusOK {
				t.Fatalf("status %d", w.Code)
			}

			body := w.Body.String()
			if got := proxied(body); got != tc.want {
				t.Fatalf("%d urls were rewritten, want %d", got, tc.want)
			}

			// those which were not are not left as a working url()
			if got := strings.Count(body, "url("); got != tc.want {
				t.Fatalf("%d url() are left, want %d", got, tc.want)
			}

			if got := len(snapshotAssets(id)); got != min(tc.want+1, tc.limit) {
				t.Fatalf("the message has %d assets, want %d", got, min(tc.want+1, tc.limit))
			}
		})
	}
}

// The same url again does not use up the limit, and one which is already known (the control for the limit) is still
// rewritten when the limit has been reached.
func TestCSSAssetsLimitIgnoresKnownURLs(t *testing.T) {
	withCSSLimits(t, 3)

	const id = "known-css"
	seedAssets(id)

	for i := range 3 {
		if !recordAsset(id, fmt.Sprintf("https://example.test/%d", i)) {
			t.Fatalf("url %d was not recorded within the limit", i)
		}
	}
	if recordAsset(id, "https://example.test/new") {
		t.Fatal("a new url was recorded over the limit")
	}
	if !recordAsset(id, "https://EXAMPLE.test/1") {
		t.Fatal("a url which is known was refused at the limit")
	}
	if got := len(snapshotAssets(id)); got != 3 {
		t.Fatalf("%d assets, want 3", got)
	}
}

// Only so many stylesheets are rewritten at once, and one which waits for its turn stops when the client goes.
func TestCSSRewritesAreLimited(t *testing.T) {
	orig := cssSlots
	cssSlots = make(chan struct{}, 1)
	t.Cleanup(func() { cssSlots = orig })

	css := uniqueURLs(10)
	uri := cssServer(t, "text/css", func(w http.ResponseWriter) { _, _ = w.Write([]byte(css)) })
	const id = "slots-css"
	seedAssets(id)
	recordAsset(id, uri)

	hold, err := acquireCSSSlot(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// the slot is taken: the request waits, and gives up (without a body) when its client is gone
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest("GET", "/proxy?data="+proxyData(id, uri), nil).WithContext(ctx)
	w := httptest.NewRecorder()
	start := time.Now()
	ProxyHandler(w, req)
	if took := time.Since(start); took < 250*time.Millisecond || w.Body.Len() != 0 {
		t.Fatalf("took %v with a body of %d bytes: it should have waited for a slot, and sent nothing", took, w.Body.Len())
	}

	// control: once it is free the stylesheet is rewritten
	hold()
	if w := proxyRequest(id, uri); w.Code != http.StatusOK || proxied(w.Body.String()) != 10 {
		t.Fatalf("status %d, %d rewritten", w.Code, proxied(w.Body.String()))
	}
}

// The content type is not case sensitive, so a stylesheet with "Text/CSS" is rewritten too (it was not).
func TestProxyRewritesCSSWhateverTheCase(t *testing.T) {
	css := uniqueURLs(5)
	uri := cssServer(t, "Text/CSS; charset=utf-8", func(w http.ResponseWriter) { _, _ = w.Write([]byte(css)) })
	const id = "case-css"
	seedAssets(id)
	recordAsset(id, uri)

	if w := proxyRequest(id, uri); w.Code != http.StatusOK || proxied(w.Body.String()) != 5 {
		t.Fatalf("status %d, %d rewritten", w.Code, proxied(w.Body.String()))
	}
}
