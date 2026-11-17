package postmark

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"
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

// capture is a handler which keeps the request it was sent.
type capture struct {
	body          []byte
	contentLength int64
	chunked       bool
	contentType   string
	method        string
}

func (c *capture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.body, _ = io.ReadAll(r.Body)
	c.contentLength = r.ContentLength
	c.chunked = len(r.TransferEncoding) > 0
	c.contentType = r.Header.Get("Content-Type")
	c.method = r.Method
	_, _ = w.Write([]byte(goodResponse))
}

// wantEmail is what the message is in JSON as encoding/json makes it, which is what was sent
// before it was streamed: the same string, with U+FFFD for each byte which is not valid UTF-8.
func wantEmail(t testing.TB, email []byte) string {
	t.Helper()

	b, err := json.Marshal(string(email))
	if err != nil {
		t.Fatal(err)
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}

	return s
}

// readAll reads from r, at most max bytes each time, so that the reader is used in small steps.
func readAll(t testing.TB, r io.Reader, max int) []byte {
	t.Helper()

	var out []byte
	buf := make([]byte, max)
	for {
		n, err := r.Read(buf)
		out = append(out, buf[:n]...)
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func emailBodies() map[string][]byte {
	around := func(prefix int, insert string) []byte {
		return []byte(strings.Repeat("a", prefix) + insert + strings.Repeat("b", 50))
	}

	return map[string][]byte{
		"empty":                      {},
		"plain":                      []byte("Subject: x\r\n\r\nhello"),
		"quotes and backslashes":     []byte(`say "hi" \ and \\ and \" "`),
		"control characters":         []byte("a\x00b\x01c\x07d\x08e\tf\ng\x0bh\x0ci\rj\x1fk\x7fl"),
		"html characters":            []byte("<b>a & b</b> <script>"),
		"two byte characters":        []byte("café über ñ"),
		"three byte characters":      []byte("日本語のテキスト"),
		"four byte characters":       []byte("emoji 😀 and 𝄞"),
		"line and paragraph breaks":  []byte("a\u2028b\u2029c\u00a0d"),
		"invalid bytes":              []byte("a\xffb\xfe\xfdc"),
		"lone continuation bytes":    []byte("a\x80\x80\x80b\xbf"),
		"truncated sequences":        []byte("a\xe3\x81 b\xf0\x9f\x98 c\xc3"),
		"a truncated one at the end": []byte("abc\xe3\x81"),
		"CRLF message":               []byte("From: a@b.test\r\nSubject: x\r\n\r\nline one\r\nline two\r\n"),

		// a character which is across the point where the message is cut into chunks
		"2 bytes across the chunk": around(escapeChunk-1, "é"),
		"3 bytes across the chunk": around(escapeChunk-1, "日"),
		"3 bytes, one byte in":     around(escapeChunk-2, "日"),
		"4 bytes across the chunk": around(escapeChunk-1, "😀"),
		"4 bytes, two bytes in":    around(escapeChunk-2, "😀"),
		"4 bytes, one byte in":     around(escapeChunk-3, "😀"),
		"exactly one chunk":        []byte(strings.Repeat("a", escapeChunk)),
		"one more than a chunk":    []byte(strings.Repeat("a", escapeChunk+1)),
		"escapes across a chunk":   around(escapeChunk-1, `"\`+"\n"),
		"every chunk is escapes":   []byte(strings.Repeat("\"\\\n\x00", 6000)),
		"many invalid bytes":       []byte(strings.Repeat("\xff\x80", 5000)),
		"many characters":          []byte(strings.Repeat("é日😀", 3000)),
	}
}

// The message is sent in JSON as it was with json.Marshal, with the length known (not chunked).
func TestRequestBodyMatchesJSONMarshal(t *testing.T) {
	for name, email := range emailBodies() {
		t.Run(name, func(t *testing.T) {
			c := &capture{}
			use(t, c)

			if _, err := Check(email, 10); err != nil {
				t.Fatal(err)
			}

			var got struct {
				Email   string `json:"email"`
				Options string `json:"options"`
			}
			if err := json.Unmarshal(c.body, &got); err != nil {
				t.Fatalf("the request is not JSON: %v: %.200q", err, c.body)
			}
			if got.Email != wantEmail(t, email) || got.Options != "long" {
				t.Errorf("the message or options sent are not what they should be (%d bytes sent for %d)", len(c.body), len(email))
			}
			// the request is valid UTF-8 even when the message is not, as JSON has to be (Go's decoder
			// accepts anything, but not everything which reads it does)
			if !utf8.Valid(c.body) {
				t.Errorf("the request is not valid UTF-8")
			}
			// the separators which not everything reading JSON accepts raw are escaped
			if bytes.Contains(c.body, []byte("\u2028")) || bytes.Contains(c.body, []byte("\u2029")) {
				t.Errorf("a line or paragraph separator is sent as it is")
			}
			if c.contentLength != int64(len(c.body)) || c.chunked {
				t.Errorf("content length %d for a body of %d bytes, chunked %v", c.contentLength, len(c.body), c.chunked)
			}
			if c.method != http.MethodPost || c.contentType != "application/json" {
				t.Errorf("method %q, content type %q", c.method, c.contentType)
			}
		})
	}
}

// The reader gives the same whatever size it is read in.
func TestEscapedReaderReadSizes(t *testing.T) {
	for name, email := range emailBodies() {
		want := readAll(t, &escapedReader{src: email}, 1<<20)

		for _, size := range []int{1, 2, 3, 7, 100, 4096, 5000} {
			if got := readAll(t, &escapedReader{src: email}, size); !bytes.Equal(got, want) {
				t.Errorf("%s: reading %d bytes at a time gives something different", name, size)
			}
		}

		var s string
		if err := json.Unmarshal(append(append([]byte(`"`), want...), '"'), &s); err != nil || s != wantEmail(t, email) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func FuzzRequestBody(f *testing.F) {
	for _, b := range emailBodies() {
		f.Add(b)
	}
	f.Add([]byte("\xf0\x9f"))

	f.Fuzz(func(t *testing.T, email []byte) {
		got := readAll(t, &escapedReader{src: email}, 4096)

		if !utf8.Valid(got) {
			t.Fatalf("%q is sent as bytes which are not valid UTF-8: %q", email, got)
		}

		var s string
		if err := json.Unmarshal(append(append([]byte(`"`), got...), '"'), &s); err != nil {
			t.Fatalf("not a JSON string: %v: %q", err, got)
		}
		if want := wantEmail(t, email); s != want {
			t.Fatalf("%q is sent as %q, want %q", email, s, want)
		}

		// the length which is worked out first is the length which is sent
		req, err := newRequest(email)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(req.Body)
		if req.ContentLength != int64(len(body)) {
			t.Fatalf("content length %d, body %d", req.ContentLength, len(body))
		}
	})
}

// The message used to be copied about 11 times (570 MB for a message of 50 MB), where it is now
// sent as it is escaped, with only a chunk of it kept.
func TestRequestDoesNotCopyTheMessage(t *testing.T) {
	use(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_, _ = w.Write([]byte(goodResponse))
	}))

	email := []byte("Subject: x\r\n\r\n" + strings.Repeat("hello \"world\" \xff <b>é</b>\r\n", (30<<20)/24))

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	if _, err := Check(email, 60); err != nil {
		t.Fatal(err)
	}

	runtime.ReadMemStats(&after)
	if used := after.TotalAlloc - before.TotalAlloc; used > 8<<20 {
		t.Errorf("sending a %d MiB message allocated %d MiB", len(email)>>20, used>>20)
	}
}
