package apiv1

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/axllent/mailpit/internal/rspamd"
	"github.com/axllent/mailpit/internal/storage"
)

func rspamdRequest(id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message/"+id+"/rspamd-check", nil)
	req.SetPathValue("id", id)
	rr := httptest.NewRecorder()
	RspamdCheck(rr, req)

	return rr
}

func TestRspamdCheckHandler(t *testing.T) {
	initTestDB(t)

	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(`{"score":7.5,"required_score":15,"action":"add header","thresholds":{"add header":6},"symbols":{"A":{"name":"A","score":7.5,"description":"d"}}}`))
	}))
	defer srv.Close()
	if _, err := rspamd.SetService(srv.URL); err != nil {
		t.Fatal(err)
	}

	raw := []byte("From: a@example.test\r\nTo: b@example.test\r\nSubject: spam\r\n\r\nbody\r\n")
	id, err := storage.Store(&raw, nil)
	if err != nil {
		t.Fatal(err)
	}

	// control: a message which exists is checked
	rr := rspamdRequest(id)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body %q", rr.Code, rr.Body.String())
	}
	var res rspamd.Result
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil || !res.IsSpam || res.Score != 7.5 || len(res.Rules) != 1 {
		t.Fatalf("unexpected result %+v (%v)", res, err)
	}

	// asking again does not send the message to Rspamd again
	for range 5 {
		if rr := rspamdRequest(id); rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
	}
	if got := hits.Load(); got != 1 {
		t.Fatalf("Rspamd received %d requests for one message, want 1", got)
	}

	// "latest" is resolved to the message, and so uses the same result
	if rr := rspamdRequest("latest"); rr.Code != http.StatusOK || hits.Load() != 1 {
		t.Fatalf("latest: status %d, %d requests", rr.Code, hits.Load())
	}

	// exploit: an ID which does not exist is still a 404, and is not sent to Rspamd
	rr = rspamdRequest("does-not-exist-0000")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("an unknown message gave status %d, want 404 (%q)", rr.Code, strings.TrimSpace(rr.Body.String()))
	}
	if hits.Load() != 1 {
		t.Fatal("an unknown message was sent to Rspamd")
	}
}
