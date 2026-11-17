package storage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/websocket"
	"github.com/jhillyerd/enmime/v2"
)

var testHubOnce sync.Once

// websocketEvents connects a client to a hub for the test, and returns a function which returns how many
// events of each type it was sent.
func websocketEvents(t *testing.T) func() map[string]int {
	t.Helper()

	// The hub is shared by every test, and made once: the goroutines which broadcast read it, so it is only
	// set when none are running, and its own goroutine reads the logging setting, which setup() has set already.
	testHubOnce.Do(func() {
		stopBackground()

		websockets.SetCheckOriginFunc(func(*http.Request) bool { return true })
		websockets.MessageHub = websockets.NewHub()
		go websockets.MessageHub.Run()
	})
	hub := websockets.MessageHub

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { websockets.ServeWs(hub, w, r) }))

	c, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}

	var (
		mu     sync.Mutex
		counts = map[string]int{}
	)

	go func() {
		for {
			_, b, err := c.ReadMessage()
			if err != nil {
				return
			}
			var m struct{ Type string }
			if json.Unmarshal(b, &m) == nil {
				mu.Lock()
				counts[m.Type]++
				mu.Unlock()
			}
		}
	}()

	t.Cleanup(func() {
		_ = c.Close()
		srv.Close()
	})

	// the client is registered with the hub
	time.Sleep(200 * time.Millisecond)

	return func() map[string]int {
		// the events are sent from goroutines
		time.Sleep(600 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		out := map[string]int{}
		for k, v := range counts {
			out[k] = v
		}

		return out
	}
}

// The web UI is told which messages a search deleted. With more matches than one chunk of IDs only the IDs
// of the last chunk were looked at, so 1,001 deleted messages were announced as one, and the other 1,000
// stayed in the list.
func TestDeleteSearchTellsTheWebUI(t *testing.T) {
	setup("")

	for _, tt := range []struct {
		name         string
		matches      int
		deleteEvents int
		pruneEvents  int
	}{
		{"control: a few are announced one by one", 150, 150, 0},
		{"exploit: one more than a chunk of IDs", 1001, 0, 1},
		{"exploit: two chunks with a short last one", 2100, 0, 1},
		{"control: more than a chunk and a long last one", 1500, 0, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_ = DeleteAllMessages()
			seedMessages(t, tt.matches, true)

			events := websocketEvents(t)

			if err := DeleteSearch("common", ""); err != nil {
				t.Fatal(err)
			}

			got := events()
			if got["delete"] != tt.deleteEvents || got["prune"] != tt.pruneEvents {
				t.Errorf("%d matches: %d delete and %d prune events, want %d and %d", tt.matches, got["delete"], got["prune"], tt.deleteEvents, tt.pruneEvents)
			}
			if CountTotal() != 0 {
				t.Errorf("%d messages are left", CountTotal())
			}
		})
	}
}

// A message which is deleted must not be served from the cache of parsed messages.
func TestDeleteSearchClearsTheEnvelopeCache(t *testing.T) {
	setup("")
	_ = DeleteAllMessages()
	ids := seedMessages(t, 2, false)

	cache := func(id string) {
		envelopeCache.Lock()
		storeEnvelopeLocked(id, &enmime.Envelope{Attachments: []*enmime.Part{{ContentType: "image/png"}}}, 10)
		envelopeCache.Unlock()
	}
	cached := func(id string) bool {
		envelopeCache.Lock()
		defer envelopeCache.Unlock()
		_, ok := envelopeCache.entries[id]

		return ok
	}

	cache(ids[0])
	cache(ids[1])

	// only the first message matches the search
	if err := DeleteSearch("common "+ids[0], ""); err != nil {
		t.Fatal(err)
	}

	if cached(ids[0]) {
		t.Error("a deleted message is still cached")
	}
	if !cached(ids[1]) {
		t.Error("control: a message which was not deleted was removed from the cache")
	}

	invalidateAllEnvelopeCache()
}
