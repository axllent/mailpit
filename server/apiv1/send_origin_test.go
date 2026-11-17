package apiv1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/axllent/mailpit/internal/storage"
)

// The address of a request which came over TCP is its IP address, and a request which did not (a Unix socket has "@"
// or the path of the client, which has no port) is local.
func TestOriginAddr(t *testing.T) {
	for remote, want := range map[string]string{
		"192.0.2.1:5555":   "192.0.2.1", // control: TCP is unchanged
		"[2001:db8::1]:80": "2001:db8::1",
		"127.0.0.1:1":      "127.0.0.1",
		"@":                "127.0.0.1", // a Unix socket
		"":                 "127.0.0.1",
		"/tmp/client.sock": "127.0.0.1",
	} {
		if got := originAddr(remote).String(); got != want {
			t.Errorf("originAddr(%q) = %q, want %q", remote, got, want)
		}
	}
}

// A message can be sent through the API over a Unix socket. It was refused with "error parsing request RemoteAddr", as
// that address has no port. Over TCP (the control) it was, and is, accepted.
func TestSendFromUnixSocket(t *testing.T) {
	initTestDB(t)

	body := `{"From":{"Email":"a@b.test"},"To":[{"Email":"c@d.test"}],"Subject":"unix socket","Text":"t"}`

	for _, remote := range []string{"192.0.2.1:5555", "@", "", "/tmp/client.sock"} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/send", strings.NewReader(body))
		req.RemoteAddr = remote
		w := httptest.NewRecorder()
		SendMessageHandler(w, req)

		var res struct{ ID string }
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil || w.Code != http.StatusOK || res.ID == "" {
			t.Fatalf("RemoteAddr %q: status %d, body %q", remote, w.Code, w.Body.String())
		}

		if _, err := storage.GetMessageRaw(res.ID); err != nil {
			t.Fatalf("RemoteAddr %q: the message was not stored: %v", remote, err)
		}
	}
}
