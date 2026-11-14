package postmark

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const goodResponse = `{"success":true,"score":"0.7","rules":[{"score":"0.7","description":"Message has a DKIM"}],` +
	`"report":"Spam detection software\n pts rule name              description\n---- ---------------------- ----------\n 0.7 DKIM_SIGNED            Message has a DKIM\n"}`

func use(t *testing.T, h http.Handler) {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	orig := endpoint
	endpoint = srv.URL
	t.Cleanup(func() { endpoint = orig })
}

func respond(status int, body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
}

func TestCheck(t *testing.T) {
	use(t, respond(200, goodResponse))

	r, err := Check([]byte("Subject: x\r\n\r\nbody"), 5)
	if err != nil {
		t.Fatal(err)
	}
	if r.Score != "0.7" || len(r.Rules) != 1 || r.Rules[0].Name != "DKIM_SIGNED" {
		t.Fatalf("%+v", r)
	}
}

// A redirect is not followed, so the message is never sent anywhere else.
func TestRedirectIsNotFollowed(t *testing.T) {
	var elsewhere atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elsewhere.Add(1)
		_, _ = w.Write([]byte(goodResponse))
	}))
	t.Cleanup(other.Close)

	use(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	}))

	if _, err := Check([]byte("x"), 5); err == nil {
		t.Error("a redirect was accepted as a result")
	}
	if elsewhere.Load() != 0 {
		t.Fatal("the message was sent to the redirect target")
	}
}

func TestErrorStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		ok     bool
	}{
		{"control: 200", 200, goodResponse, true},
		{"control: an error message from Postmark", 500, `{"success":false,"message":"Internal error"}`, true},
		{"exploit: 500 with no message", 500, `{"score":"0.0"}`, false},
		{"exploit: 502 page", 502, `<html>Bad gateway</html>`, false},
		{"exploit: 200 not JSON", 200, `<html></html>`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			use(t, respond(tt.status, tt.body))
			if _, err := Check([]byte("x"), 5); (err == nil) != tt.ok {
				t.Fatalf("error: %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestResponseSizeLimit(t *testing.T) {
	orig := maxResponseBytes
	maxResponseBytes = 64 << 10
	t.Cleanup(func() { maxResponseBytes = orig })

	body := func(n int) string {
		return `{"success":true,"score":"1.0","report":"` + strings.Repeat("a", n) + `"}`
	}

	// control: within the limit
	use(t, respond(200, body(1000)))
	if _, err := Check([]byte("x"), 5); err != nil {
		t.Fatal(err)
	}

	// exploit: over the limit, which cuts the JSON off
	use(t, respond(200, body(200<<10)))
	if _, err := Check([]byte("x"), 5); err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("error: %v", err)
	}
}

func TestNameFromReport(t *testing.T) {
	report := " 0.7 DKIM_SIGNED            Message has a DKIM\n-1.5 OTHER_RULE (x) [y]    Odd (description) [chars]"

	tests := []struct {
		name, score, desc, want string
	}{
		{"control", "0.7", "Message has a DKIM", "DKIM_SIGNED"},
		{"no match", "0.8", "Message has a DKIM", ""},
		// the values are quoted, so they can not change the pattern
		{"regexp characters", "-1.5", "Odd (description) [chars]", ""},
		{"pattern injection", ".*", ".*", ""},
		// these used to panic (invalid UTF-8) or compile a pattern as big as the description
		{"invalid UTF-8", "0.7", "\xba", ""},
		{"huge description", "0.7", strings.Repeat("x", 8<<20), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			if got := nameFromReport(tt.score, tt.desc, report); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			// compiling the 8 MiB description took about 3 seconds
			if d := time.Since(start); d > time.Second {
				t.Errorf("took %v", d)
			}
		})
	}
}

func FuzzNameFromReport(f *testing.F) {
	f.Add("0.7", "Message has a DKIM", " 0.7 FOO_RULE  Message has a DKIM")
	f.Add("(", "[", "\n\n")
	f.Fuzz(func(t *testing.T, score, desc, report string) {
		if n := nameFromReport(score, desc, report); strings.ContainsAny(n, " \t\r\n") {
			t.Fatalf("name %q", n)
		}
	})
}
