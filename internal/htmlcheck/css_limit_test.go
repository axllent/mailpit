package htmlcheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/axllent/mailpit/config"
)

func cssLinksHTML(base string, n int) string {
	var sb strings.Builder
	sb.WriteString("<html><head>")
	for i := range n {
		fmt.Fprintf(&sb, `<link rel="stylesheet" href="%s/s%d.css">`, base, i)
	}
	sb.WriteString("</head><body>hi</body></html>")

	return sb.String()
}

func TestRemoteCSSLimits(t *testing.T) {
	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	body := "/*" + strings.Repeat("A", 1000) + "*/"
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	tests := []struct {
		name      string
		links     int
		maxFiles  int
		maxBytes  int64
		wantHits  int64
		wantStyle int
	}{
		// control: under both limits, every stylesheet is fetched and inlined
		{"under limits", 5, 10, 1 << 20, 5, 5},
		// exploit: file count limit stops the fetching
		{"count limit", 50, 10, 1 << 20, 10, 10},
		// exploit: byte budget stops the fetching (2.5 files of 1002 bytes)
		{"byte limit", 50, 10, 2500, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hits.Store(0)
			out, err := inlineRemoteCSSWithLimits(context.Background(), cssLinksHTML(srv.URL, tt.links), tt.maxFiles, tt.maxBytes)
			if err != nil {
				t.Fatal(err)
			}
			if got := hits.Load(); got != tt.wantHits {
				t.Errorf("fetched %d stylesheets, want %d", got, tt.wantHits)
			}
			if got := strings.Count(out, "<style>"); got != tt.wantStyle {
				t.Errorf("inlined %d style blocks, want %d", got, tt.wantStyle)
			}
		})
	}

	t.Run("duplicate hrefs fetched once", func(t *testing.T) {
		hits.Store(0)
		html := "<html><head>" + strings.Repeat(fmt.Sprintf(`<link rel="stylesheet" href="%s/same.css">`, srv.URL), 30) + "</head><body></body></html>"
		if _, err := inlineRemoteCSS(context.Background(), html); err != nil {
			t.Fatal(err)
		}
		if got := hits.Load(); got != 1 {
			t.Errorf("fetched %d times, want 1", got)
		}
	})
}

func TestRemoteCSSCancelled(t *testing.T) {
	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "text/css")
		_, _ = w.Write([]byte("body{}"))
	}))
	defer srv.Close()

	// control: a live context fetches the stylesheet
	if _, err := inlineRemoteCSS(context.Background(), cssLinksHTML(srv.URL, 3)); err != nil {
		t.Fatal(err)
	}
	if got := hits.Load(); got != 3 {
		t.Fatalf("control fetched %d stylesheets, want 3", got)
	}

	// exploit: a cancelled context (client disconnected) fetches nothing and returns an error
	hits.Store(0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inlineRemoteCSS(ctx, cssLinksHTML(srv.URL, 3)); err == nil {
		t.Error("expected an error for a cancelled context")
	}
	if got := hits.Load(); got != 0 {
		t.Errorf("cancelled context fetched %d stylesheets, want 0", got)
	}

	// the whole check aborts too, instead of carrying on with the local CSS tests
	if _, err := RunTests(ctx, cssLinksHTML(srv.URL, 3)); err == nil {
		t.Error("expected RunTests to return an error for a cancelled context")
	}
}
