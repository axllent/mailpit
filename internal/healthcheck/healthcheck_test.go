package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := Check(srv.Client(), srv.URL); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestWaitRetriesUntilSuccess(t *testing.T) {
	oldPoll := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = oldPoll })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := Wait(srv.Client(), srv.URL, 100*time.Millisecond); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if calls < 2 {
		t.Fatalf("Wait() calls = %d, want at least 2", calls)
	}
}

func TestWaitTimesOut(t *testing.T) {
	oldPoll := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = oldPoll })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	if err := Wait(srv.Client(), srv.URL, 5*time.Millisecond); err == nil {
		t.Fatal("Wait() error = nil, want timeout")
	}

	if calls == 0 {
		t.Fatal("Wait() did not call the endpoint")
	}
}

func TestURI(t *testing.T) {
	tests := []struct {
		name    string
		listen  string
		webroot string
		https   bool
		want    string
	}{
		{"plain", "127.0.0.1:8025", "", false, "http://127.0.0.1:8025/readyz"},
		{"https", "127.0.0.1:8025", "", true, "https://127.0.0.1:8025/readyz"},
		{"webroot", "127.0.0.1:8025", "/mailpit", false, "http://127.0.0.1:8025/mailpit/readyz"},
		{"webroot trailing slash", "127.0.0.1:8025", "/mailpit/", false, "http://127.0.0.1:8025/mailpit/readyz"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := URI(tc.listen, tc.webroot, tc.https); got != tc.want {
				t.Errorf("URI() = %q, want %q", got, tc.want)
			}
		})
	}
}
