package websockets

import (
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var hubOnce sync.Once

// testServer serves the websocket of a hub (which is started once, as the hub is global) and returns its address.
func testServer(t *testing.T) string {
	t.Helper()

	hubOnce.Do(func() {
		MessageHub = NewHub()
		go MessageHub.Run()
	})

	origCheck := checkOriginFunc
	SetCheckOriginFunc(func(*http.Request) bool { return true })
	t.Cleanup(func() { checkOriginFunc = origCheck })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { ServeWs(MessageHub, w, r) }))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dial(t *testing.T, addr string, compression bool) *websocket.Conn {
	t.Helper()

	d := websocket.Dialer{EnableCompression: compression}
	c, _, err := d.Dial(addr, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })

	if compression {
		c.EnableWriteCompression(true)
		_ = c.SetCompressionLevel(1)
	}

	return c
}

// waitClients waits for the number of clients of the hub to be n.
func waitClients(t *testing.T, n int64) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for MessageHub.clientCount.Load() != n && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := MessageHub.clientCount.Load(); got != n {
		t.Fatalf("%d clients, want %d", got, n)
	}
}

// receive reads what the server sends until it is a notification of the type, or the connection closes.
func receive(c *websocket.Conn, wait time.Duration) (string, error) {
	_ = c.SetReadDeadline(time.Now().Add(wait))
	_, msg, err := c.ReadMessage()

	return string(msg), err
}

// A client which sends more than it should is closed (with the code for a message which is too large), however the
// message is sent. The message was read, and thrown away, whatever the size: 512 MiB of a compressed message took
// about as long to inflate as it did to make. (A message is counted as it is sent, so a compressed one by its
// compressed size.)
func TestClientMessagesAreLimited(t *testing.T) {
	addr := testServer(t)
	waitClients(t, 0)

	random := func(n int) []byte {
		b := make([]byte, n)
		_, _ = rand.Read(b)

		return b
	}

	for name, tc := range map[string]struct {
		message     []byte
		compression bool
	}{
		"just over the limit":                  {make([]byte, maxClientMessageSize+1), false},
		"a megabyte":                           {make([]byte, 1<<20), false},
		"compressed, just over (random data)":  {random(maxClientMessageSize + 1), true},
		"compressed, large (512 MiB of zeros)": {make([]byte, 512<<20), true},
	} {
		t.Run(name, func(t *testing.T) {
			c := dial(t, addr, tc.compression)
			waitClients(t, 1)

			go func() { _ = c.WriteMessage(websocket.TextMessage, tc.message) }()

			_, err := receive(c, 10*time.Second)
			if !websocket.IsCloseError(err, websocket.CloseMessageTooBig) {
				t.Fatalf("a message of %d bytes: %v, want the connection to be closed as too large", len(tc.message), err)
			}

			// and the hub has lost it
			waitClients(t, 0)
		})
	}
}

// Control: a client which sends little (or nothing, as the web UI does) is not closed, including a message of exactly
// the limit, and it is sent what is broadcast.
func TestClientsWhichSendLittleAreNotClosed(t *testing.T) {
	addr := testServer(t)
	waitClients(t, 0)

	for _, compression := range []bool{false, true} {
		c := dial(t, addr, compression)
		waitClients(t, 1)

		for _, size := range []int{0, 1, 100, maxClientMessageSize} {
			if err := c.WriteMessage(websocket.TextMessage, make([]byte, size)); err != nil {
				t.Fatalf("a message of %d bytes: %v", size, err)
			}
		}
		if err := c.WriteControl(websocket.PingMessage, []byte("hi"), time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}

		// it is still there: what is broadcast reaches it
		Broadcast("test", "still here")
		msg, err := receive(c, 5*time.Second)
		if err != nil || !strings.Contains(msg, "still here") {
			t.Fatalf("compression %v: %q, %v", compression, msg, err)
		}

		_ = c.Close()
		waitClients(t, 0)
	}
}
