package cmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckReady(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := checkReady(srv.Client(), srv.URL); err != nil {
		t.Fatalf("checkReady() error = %v", err)
	}
}

func TestWaitForReadyRetriesUntilSuccess(t *testing.T) {
	oldPoll := readyzPollEvery
	readyzPollEvery = time.Millisecond
	t.Cleanup(func() { readyzPollEvery = oldPoll })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	if err := waitForReady(srv.Client(), srv.URL, 100*time.Millisecond); err != nil {
		t.Fatalf("waitForReady() error = %v", err)
	}

	if calls < 2 {
		t.Fatalf("waitForReady() calls = %d, want at least 2", calls)
	}
}

func TestWaitForReadyTimesOut(t *testing.T) {
	oldPoll := readyzPollEvery
	readyzPollEvery = time.Millisecond
	t.Cleanup(func() { readyzPollEvery = oldPoll })

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	if err := waitForReady(srv.Client(), srv.URL, 5*time.Millisecond); err == nil {
		t.Fatal("waitForReady() error = nil, want timeout")
	}

	if calls == 0 {
		t.Fatal("waitForReady() did not call the endpoint")
	}
}
