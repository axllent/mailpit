package rspamd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func fakeServer(t *testing.T, wantPassword, body string, status int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ping" {
			_, _ = w.Write([]byte("pong\r\n"))
			return
		}
		if r.URL.Path != "/checkv2" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Password") != wantPassword {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"Unauthorized"}`))
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

const spamBody = `{"score":7.5,"required_score":15,"action":"add header",
"thresholds":{"reject":15,"add header":6,"greylist":4},
"symbols":{"A":{"name":"A","score":0.5,"description":"minor"},"B":{"name":"B","score":7,"description":"major"}}}`

const hamBody = `{"score":1.234,"required_score":15,"action":"no action",
"thresholds":{"reject":15,"add header":6},"symbols":{}}`

func TestCheck(t *testing.T) {
	srv := fakeServer(t, "secret", spamBody, http.StatusOK)
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	u.User = url.UserPassword("", "secret")
	shown, err := SetService(u.String() + "/")
	if err != nil || strings.Contains(shown, "secret") {
		t.Fatalf("password must be stripped from %q (%v)", shown, err)
	}
	if err := Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	res, err := Check([]byte("Subject: test\r\n\r\nbody"))
	if err != nil || res.Error != "" {
		t.Fatalf("unexpected error: %v %q", err, res.Error)
	}
	if !res.IsSpam || res.Score != 7.5 || res.Threshold != 6 {
		t.Errorf("unexpected spam result: %+v", res)
	}
	if len(res.Rules) != 2 || res.Rules[0].Name != "B" {
		t.Errorf("rules not sorted by score: %+v", res.Rules)
	}

	// control: same shape, benign message
	srv2 := fakeServer(t, "secret", hamBody, http.StatusOK)
	defer srv2.Close()
	_, _ = SetService("http://secret@" + strings.TrimPrefix(srv2.URL, "http://"))
	res, _ = Check([]byte("x"))
	if res.IsSpam || res.Score != 1.23 || res.Error != "" {
		t.Errorf("unexpected ham result: %+v", res)
	}
}

func TestSetServiceInvalid(t *testing.T) {
	for _, in := range []string{"http://", "https://:secret@%zz"} {
		if shown, err := SetService(in); err == nil || strings.Contains(err.Error(), "secret") || shown != "" {
			t.Errorf("%q: expected redacted error, got %q %v", in, shown, err)
		}
	}
}

func TestCheckErrors(t *testing.T) {
	srv := fakeServer(t, "secret", spamBody, http.StatusOK)
	defer srv.Close()

	// wrong password is surfaced as a result error, not a hard failure
	_, _ = SetService("http://:wrong@" + strings.TrimPrefix(srv.URL, "http://"))
	res, err := Check([]byte("x"))
	if err != nil || res.Error != "Unauthorized" {
		t.Errorf("expected Unauthorized, got %v %+v", err, res)
	}

	// unreachable server
	_, _ = SetService("127.0.0.1:1")
	if res, _ = Check([]byte("x")); res.Error == "" {
		t.Error("expected connection error")
	}
	if Ping() == nil {
		t.Error("expected ping error")
	}
}
