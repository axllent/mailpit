package apiv1

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/axllent/mailpit/internal/storage"
)

func storeSimpleEmail(t *testing.T, subject string) string {
	t.Helper()
	raw := []byte("From: a@example.test\r\nTo: b@example.test\r\nSubject: " + subject + "\r\n\r\nbody\r\n")
	id, err := storage.Store(&raw, nil)
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	return id
}

func resolveRequest(t *testing.T, id, rawQuery string) (string, bool, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/message/"+id+"?"+rawQuery, nil)
	rr := httptest.NewRecorder()
	got, ok := resolveMessageID(rr, req, id)
	return got, ok, rr
}

// TestResolveMessageID verifies that "latest" resolves to the newest message while any other ID is passed through.
func TestResolveMessageID(t *testing.T) {
	initTestDB(t)
	oldID := storeSimpleEmail(t, "old")
	time.Sleep(10 * time.Millisecond) // Created is stored in milliseconds, keep the order unambiguous
	newID := storeSimpleEmail(t, "new")

	// control: a literal ID is returned untouched, even when it is not the newest
	if got, ok, _ := resolveRequest(t, oldID, ""); !ok || got != oldID {
		t.Fatalf("literal ID: got %q, ok=%v, want %q, true", got, ok, oldID)
	}

	if got, ok, _ := resolveRequest(t, "latest", ""); !ok || got != newID {
		t.Fatalf("latest: got %q, ok=%v, want %q, true", got, ok, newID)
	}

	// `latest` honours the search query
	q := url.Values{"query": {"subject:old"}}.Encode()
	if got, ok, _ := resolveRequest(t, "latest", q); !ok || got != oldID {
		t.Fatalf("latest with query: got %q, ok=%v, want %q, true", got, ok, oldID)
	}
}

// TestResolveMessageIDEmpty verifies the plain text 404 when "latest" cannot be resolved.
func TestResolveMessageIDEmpty(t *testing.T) {
	initTestDB(t)

	// control: a literal ID is not looked up, so an empty database is not an error here
	if got, ok, rr := resolveRequest(t, "abc", ""); !ok || got != "abc" || rr.Code != http.StatusOK {
		t.Fatalf("literal ID: got %q, ok=%v, status=%d", got, ok, rr.Code)
	}

	got, ok, rr := resolveRequest(t, "latest", "")
	if ok || got != "" {
		t.Fatalf("latest: got %q, ok=%v, want \"\", false", got, ok)
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type=%q, want text/plain", ct)
	}
}
