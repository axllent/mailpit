package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/server/apiv1"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/gorilla/websocket"
)

const testSegmentDomain = "mail.example.com"

var testSegmentHosts = map[string]string{
	"":      "localhost:8925",
	"alpha": "alpha.mail.example.com:8925",
	"beta":  "beta.mail.example.com:8925",
}

// segmentRequest performs a request against the API routes using a specific Host header
func segmentRequest(t *testing.T, handler http.Handler, method, host, path, body string) (int, []byte) {
	t.Helper()

	req := httptest.NewRequest(method, "http://"+host+path, strings.NewReader(body))
	req.Host = host
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	b, _ := io.ReadAll(rec.Result().Body)

	return rec.Code, b
}

func segmentMessages(t *testing.T, handler http.Handler, host, path string) apiv1.MessagesSummary {
	t.Helper()

	code, b := segmentRequest(t, handler, "GET", host, path, "")
	if code != http.StatusOK {
		t.Fatalf("GET %s on %s returned %d: %s", path, host, code, string(b))
	}

	m := apiv1.MessagesSummary{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}

	return m
}

func storeSegmentMessage(t *testing.T, seg, subject string) string {
	t.Helper()

	b := []byte("From: sender@example.com\r\nTo: recipient@example.com\r\nSubject: " + subject + "\r\n\r\nHello " + seg + "\r\n")
	id, err := storage.Store(seg, &b, nil)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func enableTestSegments(t *testing.T) {
	t.Helper()

	config.SegmentDomain = testSegmentDomain
	t.Cleanup(func() { config.SegmentDomain = "" })
}

func TestSegmentsAPI(t *testing.T) {
	setup()
	defer storage.Close()
	enableTestSegments(t)

	ids := map[string]string{}
	for seg := range testSegmentHosts {
		ids[seg] = storeSegmentMessage(t, seg, "Message for "+seg)
	}

	r := apiRoutes()
	r.HandleFunc("GET /view/", middleWareFunc(viewHandler))

	for seg, host := range testSegmentHosts {
		other := "beta"
		if seg == "beta" {
			other = "alpha"
		}

		m := segmentMessages(t, r, host, "/api/v1/messages")
		assertEqual(t, uint64(1), m.Total, "wrong total for "+host)
		assertEqual(t, 1, len(m.Messages), "wrong number of messages for "+host)
		assertEqual(t, ids[seg], m.Messages[0].ID, "wrong message for "+host)

		s := segmentMessages(t, r, host, "/api/v1/search?query=subject:message")
		assertEqual(t, uint64(1), s.MessagesCount, "wrong search count for "+host)
		assertEqual(t, ids[seg], s.Messages[0].ID, "wrong search result for "+host)

		// the own message is accessible
		for _, path := range []string{"/api/v1/message/" + ids[seg], "/api/v1/message/" + ids[seg] + "/raw", "/api/v1/message/" + ids[seg] + "/headers", "/view/" + ids[seg] + ".txt"} {
			if code, b := segmentRequest(t, r, "GET", host, path, ""); code != http.StatusOK {
				t.Fatalf("GET %s on %s returned %d: %s", path, host, code, string(b))
			}
		}

		// the message of another segment is not
		for _, path := range []string{"/api/v1/message/" + ids[other], "/api/v1/message/" + ids[other] + "/raw", "/api/v1/message/" + ids[other] + "/headers", "/api/v1/message/" + ids[other] + "/part/1", "/view/" + ids[other] + ".html", "/view/" + ids[other] + ".txt"} {
			if code, _ := segmentRequest(t, r, "GET", host, path, ""); code == http.StatusOK {
				t.Fatalf("GET %s on %s returned a message of segment %q", path, host, other)
			}
		}

		// latest only resolves within the segment
		code, b := segmentRequest(t, r, "GET", host, "/api/v1/message/latest", "")
		if code != http.StatusOK || !strings.Contains(string(b), ids[seg]) {
			t.Fatalf("latest on %s did not return the own message", host)
		}
	}

	// unknown subdomains and other hosts are served the default segment
	for _, host := range []string{"gamma.other.example.com", "a.b.mail.example.com", "mail.example.com", "127.0.0.1:8925"} {
		m := segmentMessages(t, r, host, "/api/v1/messages")
		assertEqual(t, 1, len(m.Messages), "wrong number of messages for "+host)
		assertEqual(t, ids[""], m.Messages[0].ID, "wrong message for "+host)
	}

	// a known subdomain without messages is empty
	m := segmentMessages(t, r, "gamma.mail.example.com", "/api/v1/messages")
	assertEqual(t, 0, len(m.Messages), "segment gamma should be empty")

	// marking & deleting messages of another segment has no effect
	alpha := testSegmentHosts["alpha"]
	beta := testSegmentHosts["beta"]

	// viewing the message above marked it read, so reset first
	segmentRequest(t, r, "PUT", beta, "/api/v1/messages", `{"IDs":["`+ids["beta"]+`"],"Read":false}`)
	assertEqual(t, uint64(1), segmentMessages(t, r, beta, "/api/v1/messages").Unread, "beta could not mark its message unread")
	segmentRequest(t, r, "PUT", alpha, "/api/v1/messages", `{"IDs":["`+ids["beta"]+`"],"Read":true}`)
	assertEqual(t, uint64(1), segmentMessages(t, r, beta, "/api/v1/messages").Unread, "alpha marked a beta message read")

	segmentRequest(t, r, "PUT", alpha, "/api/v1/tags", `{"IDs":["`+ids["beta"]+`"],"Tags":["Alpha"]}`)
	assertEqual(t, 0, len(segmentMessages(t, r, beta, "/api/v1/messages").Tags), "alpha tagged a beta message")

	segmentRequest(t, r, "DELETE", alpha, "/api/v1/messages", `{"IDs":["`+ids["beta"]+`"]}`)
	assertEqual(t, uint64(1), segmentMessages(t, r, beta, "/api/v1/messages").Total, "alpha deleted a beta message")

	// delete all only deletes the own segment
	segmentRequest(t, r, "DELETE", alpha, "/api/v1/messages", "")
	assertEqual(t, uint64(0), segmentMessages(t, r, alpha, "/api/v1/messages").Total, "alpha delete all failed")
	assertEqual(t, uint64(1), segmentMessages(t, r, beta, "/api/v1/messages").Total, "alpha delete all deleted beta messages")
	assertEqual(t, uint64(1), segmentMessages(t, r, testSegmentHosts[""], "/api/v1/messages").Total, "alpha delete all deleted default messages")

	// the send API stores in the segment of the host
	code, b := segmentRequest(t, r, "POST", alpha, "/api/v1/send", `{"From":{"Email":"test@example.com"},"To":[{"Email":"recipient@example.com"}],"Subject":"Sent to alpha","Text":"Hello"}`)
	if code != http.StatusOK {
		t.Fatalf("send API returned %d: %s", code, string(b))
	}
	assertEqual(t, uint64(1), segmentMessages(t, r, alpha, "/api/v1/messages").Total, "send API did not store in alpha")
	assertEqual(t, uint64(1), segmentMessages(t, r, beta, "/api/v1/messages").Total, "send API stored in beta")
}

func TestSegmentsAllowedHosts(t *testing.T) {
	setup()
	defer storage.Close()

	AllowedHosts = "mailpit.example.com"
	setAllowedHosts()
	defer func() {
		AllowedHosts = ""
		setAllowedHosts()
	}()

	r := apiRoutes()

	if code, _ := segmentRequest(t, r, "GET", "alpha.mail.example.com", "/api/v1/messages", ""); code != http.StatusForbidden {
		t.Fatalf("segment host should be blocked when segments are disabled, got %d", code)
	}

	enableTestSegments(t)

	for _, host := range []string{"alpha.mail.example.com", "alpha.mail.example.com:8925", "mail.example.com"} {
		if code, _ := segmentRequest(t, r, "GET", host, "/api/v1/messages", ""); code != http.StatusOK {
			t.Fatalf("segment host %s should be allowed, got %d", host, code)
		}
	}

	if code, _ := segmentRequest(t, r, "GET", "alpha.evil.example.com", "/api/v1/messages", ""); code != http.StatusForbidden {
		t.Fatalf("other hosts should still be blocked, got %d", code)
	}
}

func TestSegmentsWebsocket(t *testing.T) {
	setup()
	defer storage.Close()
	enableTestSegments(t)

	websockets.MessageHub = websockets.NewHub()
	websockets.SetCheckOriginFunc(corsOriginAccessControl)
	go websockets.MessageHub.Run()
	defer func() { websockets.MessageHub = nil }()

	ts := httptest.NewServer(apiRoutes())
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/events"

	connect := func(host string) *websocket.Conn {
		h := http.Header{}
		h.Set("Host", host)
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, h)
		if err != nil {
			t.Fatalf("websocket dial for %s: %s", host, err.Error())
		}
		_ = resp.Body.Close()
		return conn
	}

	alphaConn := connect(testSegmentHosts["alpha"])
	defer func() { _ = alphaConn.Close() }()
	betaConn := connect(testSegmentHosts["beta"])
	defer func() { _ = betaConn.Close() }()

	// give the hub time to register both clients
	time.Sleep(300 * time.Millisecond)

	id := storeSegmentMessage(t, "alpha", "Live alpha message")

	// read events until the deadline, returning the IDs of all "new" events
	newEvents := func(conn *websocket.Conn) []string {
		ids := []string{}
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		for {
			_, b, err := conn.ReadMessage()
			if err != nil {
				return ids
			}
			n := struct {
				Type string
				Data struct{ ID string }
			}{}
			if err := json.Unmarshal(b, &n); err == nil && n.Type == "new" {
				ids = append(ids, n.Data.ID)
			}
		}
	}

	alphaIDs := newEvents(alphaConn)
	if len(alphaIDs) != 1 || alphaIDs[0] != id {
		t.Fatalf("alpha client did not receive its new message event, got %v", alphaIDs)
	}

	if betaIDs := newEvents(betaConn); len(betaIDs) != 0 {
		t.Fatalf("beta client received new message events of alpha: %v", betaIDs)
	}
}

func TestSegmentsPasswordAuth(t *testing.T) {
	setup()
	defer storage.Close()
	enableTestSegments(t)

	config.SegmentPassword = "shared"
	t.Cleanup(func() { config.SegmentPassword = "" })

	storeSegmentMessage(t, "alpha", "Alpha mail")

	handler := apiRoutes()

	get := func(host, user, pass string) int {
		req := httptest.NewRequest("GET", "http://"+host+"/api/v1/messages", nil)
		req.Host = host
		if user != "" || pass != "" {
			req.SetBasicAuth(user, pass)
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		return rec.Code
	}

	tests := []struct {
		host, user, pass string
		code             int
	}{
		{testSegmentHosts["alpha"], "alpha", "shared", http.StatusOK},
		{testSegmentHosts["beta"], "beta", "shared", http.StatusOK},
		{testSegmentHosts["beta"], "alpha", "shared", http.StatusUnauthorized},
		{testSegmentHosts["alpha"], "beta", "shared", http.StatusUnauthorized},
		{testSegmentHosts["alpha"], "alpha", "wrong", http.StatusUnauthorized},
		{testSegmentHosts["alpha"], "", "", http.StatusUnauthorized},
		{testSegmentHosts[""], "alpha", "shared", http.StatusUnauthorized},
	}

	for _, tc := range tests {
		if got := get(tc.host, tc.user, tc.pass); got != tc.code {
			t.Errorf("GET on %s as %q/%q returned %d, expected %d", tc.host, tc.user, tc.pass, got, tc.code)
		}
	}
}
