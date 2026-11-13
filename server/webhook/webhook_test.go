package webhook

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

func TestSendDeliversEveryRequestReusingConnections(t *testing.T) {
	var requests, conns atomic.Int64

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(5 * time.Millisecond)
		requests.Add(1)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	defer srv.Close()

	config.WebhookURL = srv.URL
	RateLimit = 0
	defer func() { config.WebhookURL = "" }()

	n := 2000
	for i := range n {
		Send(map[string]int{"i": i})
	}

	deadline := time.Now().Add(20 * time.Second)
	for requests.Load() < int64(n) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := requests.Load(); got != int64(n) {
		t.Fatalf("received %d webhook requests, want %d", got, n)
	}
	// connections are returned to the idle pool asynchronously, so allow some overshoot
	if got := conns.Load(); got > 2*maxConcurrent {
		t.Fatalf("opened %d connections, want at most %d", got, 2*maxConcurrent)
	}
	t.Logf("%d requests over %d connections", requests.Load(), conns.Load())
}
