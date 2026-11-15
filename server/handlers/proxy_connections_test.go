package handlers

import (
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// assetServer serves a small image (/a.png) and a redirect to it (/r), and counts the connections which are open.
func assetServer(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()

	var open atomic.Int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/r" {
			http.Redirect(w, r, "/a.png", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			open.Add(1)
		case http.StateClosed, http.StateHijacked:
			open.Add(-1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	return srv, &open
}

func proxyRequest(id, uri string) *httptest.ResponseRecorder {
	data := base64.StdEncoding.EncodeToString([]byte(id + ":" + uri))
	w := httptest.NewRecorder()
	ProxyHandler(w, httptest.NewRequest("GET", "/proxy?data="+data, nil))

	return w
}

// waitOpen waits for the number of open connections to be at most max.
func waitOpen(open *atomic.Int64, max int64) int64 {
	deadline := time.Now().Add(3 * time.Second)
	for open.Load() > max && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	return open.Load()
}

// A request does not leave a connection open on the server. Each one used to, with three goroutines, for as long as
// the server kept it (an attacker's server could keep it for ever), which could use up all file descriptors.
func TestProxyDoesNotKeepConnectionsOpen(t *testing.T) {
	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	srv, open := assetServer(t)

	const id = "connections"
	seedAssets(id)

	// the second is a redirect, which is another request on the same transport
	for _, path := range []string{"/a.png", "/r"} {
		uri := srv.URL + path
		recordAsset(id, uri)

		for range 30 {
			w := proxyRequest(id, uri)
			if w.Code != http.StatusOK || w.Body.String() != "png" || w.Header().Get("Content-Type") != "image/png" {
				t.Fatalf("%s: status %d, type %q, body %q", path, w.Code, w.Header().Get("Content-Type"), w.Body.String())
			}
		}
	}

	if n := waitOpen(open, 0); n != 0 {
		t.Fatalf("%d connections are still open on the server after 60 requests", n)
	}
}

// The same when the response is refused after it has started (here, for a type which is not supported).
func TestProxyClosesConnectionsOfRefusedResponses(t *testing.T) {
	config.AllowInternalHTTPRequests = true
	t.Cleanup(func() { config.AllowInternalHTTPRequests = false })

	var open atomic.Int64
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>"))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		switch s {
		case http.StateNew:
			open.Add(1)
		case http.StateClosed, http.StateHijacked:
			open.Add(-1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	const id = "refused"
	seedAssets(id)
	uri := srv.URL + "/page"
	recordAsset(id, uri)

	for range 30 {
		if w := proxyRequest(id, uri); w.Code != http.StatusBadRequest {
			t.Fatalf("status %d, want the unsupported type to be refused", w.Code)
		}
	}

	if n := waitOpen(&open, 0); n != 0 {
		t.Fatalf("%d connections are still open on the server after 30 refused responses", n)
	}
}
