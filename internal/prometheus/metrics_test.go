package prometheus

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/stats"
)

// TestMain silences the logger and shortens the server timeouts before any test runs, as they
// are read by goroutines that are still running during later tests.
func TestMain(m *testing.M) {
	logger.NoLogging = true
	serverIdleTimeout = 800 * time.Millisecond

	os.Exit(m.Run())
}

func tagCounts(n int) map[string]int64 {
	tags := make(map[string]int64, n)
	for i := range n {
		tags[fmt.Sprintf("tag-%05d", i)] = int64(i + 1)
	}

	return tags
}

func scrape() string {
	var buf bytes.Buffer
	writeMetrics(&buf)

	return buf.String()
}

func TestTopTags(t *testing.T) {
	t.Run("control: fewer than the limit are all returned", func(t *testing.T) {
		got := topTags(map[string]int64{"a": 3, "b": 1, "c": 2}, 5)
		if len(got) != 3 || got["a"] != 3 || got["b"] != 1 || got["c"] != 2 {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("control: exactly the limit", func(t *testing.T) {
		if got := topTags(tagCounts(500), 500); len(got) != 500 {
			t.Fatalf("got %d", len(got))
		}
	})

	t.Run("exploit: the most used tags are kept", func(t *testing.T) {
		got := topTags(tagCounts(2000), 500)
		if len(got) != 500 {
			t.Fatalf("got %d tags", len(got))
		}
		// tag-01999 has the most messages (2000), tag-01500 the 500th most (1501)
		if _, ok := got["tag-01999"]; !ok {
			t.Error("the most used tag was dropped")
		}
		if _, ok := got["tag-01500"]; !ok {
			t.Error("the 500th most used tag was dropped")
		}
		if _, ok := got["tag-01499"]; ok {
			t.Error("the 501st most used tag was kept")
		}
	})

	t.Run("ties are broken by name, so the choice is stable", func(t *testing.T) {
		tags := map[string]int64{}
		for _, n := range []string{"e", "b", "d", "a", "c"} {
			tags[n] = 7
		}
		for range 20 {
			got := topTags(tags, 3)
			if len(got) != 3 || got["a"] != 7 || got["b"] != 7 || got["c"] != 7 {
				t.Fatalf("got %v, want a, b and c", got)
			}
		}
	})

	t.Run("no tags", func(t *testing.T) {
		if got := topTags(nil, 500); len(got) != 0 {
			t.Fatalf("got %v", got)
		}
	})
}

func TestApplyMetricsLimitsTagSeries(t *testing.T) {
	initMetrics()

	// control: a few tags are all exported, and counted
	applyMetrics(stats.AppInformation{Tags: tagCounts(3)})
	out := scrape()
	if got := strings.Count(out, "mailpit_tag_messages{"); got != 3 {
		t.Fatalf("%d series for 3 tags", got)
	}
	if !strings.Contains(out, "\nmailpit_tags 3\n") {
		t.Fatalf("mailpit_tags is not 3:\n%s", out)
	}

	// exploit: tens of thousands of tags, as one message can create, are not
	applyMetrics(stats.AppInformation{Tags: tagCounts(60000)})
	out = scrape()
	if got := strings.Count(out, "mailpit_tag_messages{"); got != maxTagSeries {
		t.Fatalf("%d series for 60000 tags, want %d", got, maxTagSeries)
	}
	if !strings.Contains(out, "\nmailpit_tags 60000\n") {
		t.Fatal("mailpit_tags does not report the real number of tags")
	}
	if len(out) > 100<<10 {
		t.Fatalf("a scrape is %d KiB", len(out)>>10)
	}
}

// The tag series are replaced in one step. They used to be cleared and then refilled, so a scrape
// in between saw some of them, or none.
func TestTagSeriesAreReplacedAtomically(t *testing.T) {
	initMetrics()

	const n = 400
	setA, setB := map[string]int64{}, map[string]int64{}
	for i := range n {
		setA[fmt.Sprintf("a-%03d", i)] = 1
		setB[fmt.Sprintf("b-%03d", i)] = 1
	}
	applyMetrics(stats.AppInformation{Tags: setA})

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; ; i++ {
			select {
			case <-stop:
				return
			default:
				if i%2 == 0 {
					applyMetrics(stats.AppInformation{Tags: setB})
				} else {
					applyMetrics(stats.AppInformation{Tags: setA})
				}
			}
		}
	}()

	deadline := time.Now().Add(1500 * time.Millisecond)
	scrapes := 0
	for time.Now().Before(deadline) {
		out := scrape()
		scrapes++
		a, b := strings.Count(out, `mailpit_tag_messages{tag="a-`), strings.Count(out, `mailpit_tag_messages{tag="b-`)
		if !(a == n && b == 0) && !(a == 0 && b == n) {
			close(stop)
			wg.Wait()
			t.Fatalf("a scrape saw %d + %d series, want all of one set (%d) and none of the other", a, b, n)
		}
	}
	close(stop)
	wg.Wait()
	t.Logf("%d scrapes during continuous updates", scrapes)
}

func TestInitMetricsIsIdempotent(t *testing.T) {
	initMetrics()
	initMetrics()

	out := scrape()
	for _, name := range []string{"mailpit_messages", "mailpit_tags", "mailpit_tag_messages", "mailpit_uptime_seconds"} {
		if got := strings.Count(out, "# TYPE "+name+" "); got != 1 {
			t.Errorf("# TYPE %s appears %d times, want once", name, got)
		}
	}
}

func TestEscapeLabelValue(t *testing.T) {
	tests := []struct{ in, want string }{
		{"tag", "tag"},
		{"a b-c_d.e@f", "a b-c_d.e@f"},
		{`a"b`, `a\"b`},
		{"a\nb", `a\nb`},
		{`a\b`, `a\\b`},
		{"日本語", "日本語"},
		// a value which tries to end the label and start a new metric stays inside the label
		{"x\"} 1\n# TYPE evil gauge\nevil{a=\"", `x\"} 1\n# TYPE evil gauge\nevil{a=\"`},
	}

	for _, tt := range tests {
		if got := escapeLabelValue(tt.in); got != tt.want {
			t.Errorf("escapeLabelValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGetMode(t *testing.T) {
	orig := config.PrometheusListen
	t.Cleanup(func() { config.PrometheusListen = orig })

	tests := []struct{ in, want string }{
		{"", "disabled"},
		{"false", "disabled"},
		{" FALSE ", "disabled"},
		{"true", "integrated"},
		{" True ", "integrated"},
		{"0.0.0.0:9090", "separate"},
		{"127.0.0.1:9090", "separate"},
	}

	for _, tt := range tests {
		config.PrometheusListen = tt.in
		if got := GetMode(); got != tt.want {
			t.Errorf("GetMode() with %q = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func startTestServer(t *testing.T, maxConns int) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() { _ = serve(ln, maxConns) }()

	return ln.Addr().String()
}

// request sends a metrics request on the connection and reports whether a response came back.
func request(c net.Conn, wait time.Duration) bool {
	if _, err := io.WriteString(c, "GET /metrics HTTP/1.1\r\nHost: x\r\n\r\n"); err != nil {
		return false
	}
	_ = c.SetReadDeadline(time.Now().Add(wait))

	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func TestSeparateServerServesMetrics(t *testing.T) {
	initMetrics()
	addr := startTestServer(t, 10)

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if !request(c, 2*time.Second) {
		t.Fatal("no metrics response")
	}
}

// An idle connection is closed, where it used to be held open for ever.
func TestSeparateServerClosesIdleConnections(t *testing.T) {
	initMetrics()
	addr := startTestServer(t, 10)

	// control: a connection used again within the idle timeout is kept
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if !request(c, 2*time.Second) {
		t.Fatal("first request failed")
	}
	time.Sleep(serverIdleTimeout / 4)
	if !request(c, 2*time.Second) {
		t.Fatal("a connection used within the idle timeout was closed")
	}

	// exploit: one that sits idle for longer is closed
	time.Sleep(serverIdleTimeout + 700*time.Millisecond)
	if request(c, 2*time.Second) {
		t.Fatal("an idle connection was still open well after the idle timeout")
	}
}

// The number of connections is limited. Further connections wait until one closes.
func TestSeparateServerLimitsConnections(t *testing.T) {
	initMetrics()

	const limit = 3
	addr := startTestServer(t, limit)

	var held []net.Conn
	for range limit {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		if !request(c, 2*time.Second) {
			t.Fatal("a connection within the limit was not served")
		}
		held = append(held, c)
	}

	// exploit: one more than the limit is not served while the others are open
	extra, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer extra.Close()
	if request(extra, 400*time.Millisecond) {
		t.Fatal("a connection beyond the limit was served")
	}

	// control: once a slot is free it is
	_ = held[0].Close()
	served := false
	for range 20 {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		if request(c, time.Second) {
			served = true
			_ = c.Close()
			break
		}
		_ = c.Close()
	}
	if !served {
		t.Fatal("a slot was not reusable after a connection closed")
	}
}
