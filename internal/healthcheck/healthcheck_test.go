package healthcheck

import (
	"net"
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

// silentServer accepts connections and never answers.
func silentServer(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			t.Cleanup(func() { _ = c.Close() })
		}
	}()

	return "http://" + ln.Addr().String() + "/readyz"
}

// A server which accepts the connection and never answers is given up on. The probe had no time limit, so Check (and
// Wait, whatever its timeout) waited for ever. The tests above, of servers which answer, are the controls.
func TestCheckOfAServerWhichDoesNotAnswer(t *testing.T) {
	oldTimeout := CheckTimeout
	CheckTimeout = 200 * time.Millisecond
	t.Cleanup(func() { CheckTimeout = oldTimeout })

	uri := silentServer(t)

	done := make(chan error, 1)
	go func() { done <- Check(NewClient(), uri) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Check() error = nil for a server which does not answer")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Check() is still waiting, the probe time is 200ms")
	}
}

// Wait does not run past its timeout for a server which does not answer, even when that is shorter than the time a
// single probe may take.
func TestWaitForAServerWhichDoesNotAnswer(t *testing.T) {
	oldPoll := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = oldPoll })

	uri := silentServer(t)

	start := time.Now()
	done := make(chan error, 1)
	go func() { done <- Wait(NewClient(), uri, 300*time.Millisecond) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Wait() error = nil for a server which does not answer")
		}
		if took := time.Since(start); took > 2*time.Second {
			t.Fatalf("Wait() took %v, the timeout is 300ms", took)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Wait() is still waiting, the timeout is 300ms")
	}
}
