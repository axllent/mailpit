package webhook

import (
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

func TestSendDeliversEveryRequestReusingConnections(t *testing.T) {
	var requests, conns atomic.Int64

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Millisecond)
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

	// a fixed number of senders, each sending in a loop, like messages arriving
	// over several concurrent SMTP or API connections
	senders, perSender := 10, 200
	n := senders * perSender
	var wg sync.WaitGroup
	for s := range senders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range perSender {
				Send(map[string]int{"sender": s, "i": i})
				time.Sleep(2 * time.Millisecond)
			}
		}()
	}
	wg.Wait()

	deadline := time.Now().Add(20 * time.Second)
	for requests.Load() < int64(n) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got := requests.Load(); got != int64(n) {
		t.Fatalf("received %d webhook requests, want %d", got, n)
	}
	// without connection reuse every request opens a new connection
	if got := conns.Load(); got > int64(n/10) {
		t.Fatalf("opened %d connections for %d requests, want at most %d", got, n, n/10)
	}
	t.Logf("%d requests over %d connections", requests.Load(), conns.Load())
}
