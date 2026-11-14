package dump

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

func resetState(url string) {
	URL = url
	base = ""
	dumpIDs = make(map[string]struct{ Timestamp time.Time })
}

func testIDs(from, n int) []map[string]any {
	out := make([]map[string]any, n)
	for i := range out {
		out[i] = map[string]any{"ID": fmt.Sprintf("id%08d", from+i), "Created": time.Now()}
	}
	return out
}

// newServer serves summary pages from pageFn and a fixed raw body, counting summary requests.
func newServer(hits *int64, total uint64, pageFn func(start, limit int) []map[string]any, raw func(w http.ResponseWriter, id string)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/messages" {
			// guard so a regression fails the test instead of hanging it
			if atomic.AddInt64(hits, 1) > 100 {
				http.Error(w, "too many requests", http.StatusInternalServerError)
				return
			}
			var s, l int
			_, _ = fmt.Sscan(r.URL.Query().Get("start"), &s)
			_, _ = fmt.Sscan(r.URL.Query().Get("limit"), &l)
			_ = json.NewEncoder(w).Encode(map[string]any{"total": total, "messages": pageFn(s, l)})
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/message/"), "/raw")
		if raw != nil {
			raw(w, id)
			return
		}
		_, _ = w.Write([]byte("Subject: x\r\n\r\nbody"))
	}))
}

// pager returns an honest pager over n messages, serving at most maxPage per request.
func pager(n, maxPage int) func(s, l int) []map[string]any {
	return func(s, l int) []map[string]any {
		if s >= n {
			return nil
		}
		c := n - s
		if c > l {
			c = l
		}
		if c > maxPage {
			c = maxPage
		}
		return testIDs(s, c)
	}
}

func countFiles(t *testing.T, dir string) int {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(ents)
}

func TestSyncRemotePaging(t *testing.T) {
	config.MaxMessageSize = 50

	tests := []struct {
		name      string
		total     uint64
		pageFn    func(s, l int) []map[string]any
		wantErr   bool
		wantFiles int
	}{
		{"control: honest server", 120, pager(120, 1<<30), false, 120},
		// regression: start must advance by the page actually returned, not pageSize
		{"proxy clamps page size to 50", 120, pager(120, 50), false, 120},
		{"exploit: same page repeated with huge total", 1 << 20, func(_, _ int) []map[string]any { return testIDs(0, 5) }, true, 0},
		{"exploit: total over the cap", maxRemoteMessages + 1, pager(10, 1<<30), true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int64
			srv := newServer(&hits, tt.total, tt.pageFn, nil)
			defer srv.Close()

			resetState(srv.URL)
			dir := t.TempDir()
			err := Sync(dir)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got := countFiles(t, dir); got != tt.wantFiles {
				t.Fatalf("files written = %d, want %d", got, tt.wantFiles)
			}
			if hits > 50 {
				t.Fatalf("made %d summary requests, expected the loop to stop early", hits)
			}
		})
	}
}

func TestSyncRemoteUniquePagesBounded(t *testing.T) {
	// A server advertising an in-range total but serving endless unique IDs must still stop at
	// the total (control), and a lying total beyond the cap is rejected up front (see above).
	config.MaxMessageSize = 50
	var hits int64
	srv := newServer(&hits, 25, func(s, _ int) []map[string]any { return testIDs(s, 10) }, nil)
	defer srv.Close()

	resetState(srv.URL)
	dir := t.TempDir()
	if err := Sync(dir); err != nil {
		t.Fatal(err)
	}
	if hits > 5 {
		t.Fatalf("made %d requests for 25 messages", hits)
	}
}

func TestSyncReportsFailedMessages(t *testing.T) {
	config.MaxMessageSize = 50

	tests := []struct {
		name    string
		raw     func(w http.ResponseWriter, id string)
		wantErr bool
	}{
		{"control: all messages fetched", nil, false},
		{"exploit: one message returns HTTP 500", func(w http.ResponseWriter, id string) {
			if id == "id00000001" {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			_, _ = w.Write([]byte("Subject: x\r\n\r\nbody"))
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int64
			srv := newServer(&hits, 3, pager(3, 1<<30), tt.raw)
			defer srv.Close()

			resetState(srv.URL)
			dir := t.TempDir()
			err := Sync(dir)

			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			// the failure must not stop the other messages being saved
			want := 3
			if tt.wantErr {
				want = 2
			}
			if got := countFiles(t, dir); got != want {
				t.Fatalf("files written = %d, want %d", got, want)
			}
		})
	}
}
