package stats

import (
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
)

func TestMain(m *testing.M) {
	logger.NoLogging = true
	m.Run()
}

// resetVersionCache clears the cache and replaces the lookup for one test.
func resetVersionCache(t *testing.T, fetch func() (string, error)) {
	t.Helper()

	origFetch := fetchLatest
	fetchLatest = fetch

	versionMu.Lock()
	vCache = versionCache{}
	versionMu.Unlock()

	t.Cleanup(func() {
		fetchLatest = origFetch
		versionMu.Lock()
		vCache = versionCache{}
		versionMu.Unlock()
	})
}

func TestGetBackoff(t *testing.T) {
	want := map[int]time.Duration{
		-1: time.Minute, 0: time.Minute, 1: 2 * time.Minute, 2: 4 * time.Minute, 3: 8 * time.Minute,
		4: 16 * time.Minute, 5: maxBackoff, 6: maxBackoff,
	}
	for n, d := range want {
		if got := getBackoff(n); got != d {
			t.Errorf("getBackoff(%d) = %v, want %v", n, got, d)
		}
	}

	// exploit: it used to overflow from 28 errors (negative), and was 0 from 53
	for _, n := range []int{27, 28, 29, 31, 40, 52, 53, 63, 64, 100, 1 << 30} {
		if got := getBackoff(n); got != maxBackoff {
			t.Errorf("getBackoff(%d) = %v, want %v", n, got, maxBackoff)
		}
	}
}

// After many failures a lookup must still not be retried by every request.
func TestManyFailuresStillBackOff(t *testing.T) {
	var calls atomic.Int32
	resetVersionCache(t, func() (string, error) {
		calls.Add(1)
		return "", errors.New("offline")
	})

	for range 60 {
		// the cache expires, as it would after the backoff
		versionMu.Lock()
		vCache.expiry = time.Time{}
		versionMu.Unlock()

		latestVersion()
	}

	// control: now the backoff applies, and further requests do not look it up again
	before := calls.Load()
	for range 20 {
		if v := latestVersion(); v != "" {
			t.Fatalf("version %q after failures", v)
		}
	}
	if got := calls.Load() - before; got != 0 {
		t.Fatalf("%d lookups within the backoff after 60 failures", got)
	}
}

func TestVersionCache(t *testing.T) {
	var calls atomic.Int32
	fail := atomic.Bool{}
	resetVersionCache(t, func() (string, error) {
		calls.Add(1)
		if fail.Load() {
			return "", errors.New("offline")
		}
		return "v9.9.9", nil
	})

	// a success is cached
	for range 3 {
		if v := latestVersion(); v != "v9.9.9" {
			t.Fatalf("version %q", v)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("%d lookups, want 1", calls.Load())
	}

	// a failure gives no version, is not retried within the backoff, and the count is reset by a success
	fail.Store(true)
	versionMu.Lock()
	vCache.expiry = time.Time{}
	versionMu.Unlock()
	if v := latestVersion(); v != "" {
		t.Fatalf("version %q after a failure", v)
	}
	if latestVersion(); calls.Load() != 2 {
		t.Fatalf("%d lookups, a failure was retried within the backoff", calls.Load())
	}

	fail.Store(false)
	versionMu.Lock()
	vCache.expiry = time.Time{}
	versionMu.Unlock()
	if v := latestVersion(); v != "v9.9.9" {
		t.Fatalf("version %q after recovery", v)
	}
	versionMu.Lock()
	errs := vCache.errCount
	versionMu.Unlock()
	if errs != 0 {
		t.Fatalf("error count %d after a success", errs)
	}
}

// The lookup used to be made while holding the lock that counting an accepted message needs, so a
// lookup that hung (10 seconds) delayed the SMTP response.
func TestSlowLookupDoesNotBlockCounters(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var once sync.Once
	var calls atomic.Int32
	resetVersionCache(t, func() (string, error) {
		calls.Add(1)
		once.Do(func() { close(started) })
		<-release
		return "v1.0.0", nil
	})

	done := make(chan string)
	go func() { done <- latestVersion() }()
	<-started

	// exploit: another request while the lookup is in progress must not wait for it, or start a
	// second lookup. It used to hold the lock the SMTP counters needed.
	got := make(chan string, 1)
	go func() {
		LogSMTPAccepted(10)
		got <- latestVersion()
	}()

	select {
	case other := <-got:
		if other != "" || calls.Load() != 1 {
			t.Errorf("second request got %q, %d lookups", other, calls.Load())
		}
	case <-time.After(2 * time.Second):
		t.Error("a request waited for the lookup in progress")
	}

	// control: once it finishes the value is available to everyone
	close(release)
	if v := <-done; v != "v1.0.0" {
		t.Fatalf("version %q", v)
	}
	if v := latestVersion(); v != "v1.0.0" {
		t.Fatalf("cached version %q", v)
	}
}

func TestCountersAreCounted(t *testing.T) {
	a, s, r, i := smtpAccepted.Load(), smtpAcceptedSize.Load(), smtpRejected.Load(), smtpIgnored.Load()

	LogSMTPAccepted(100)
	LogSMTPAccepted(-5) // never counted as a huge number
	LogSMTPRejected()
	LogSMTPIgnored()

	if got := smtpAccepted.Load() - a; got != 2 {
		t.Errorf("accepted %d", got)
	}
	if got := smtpAcceptedSize.Load() - s; got != 100 {
		t.Errorf("size %d", got)
	}
	if smtpRejected.Load()-r != 1 || smtpIgnored.Load()-i != 1 {
		t.Error("rejected or ignored")
	}
}

// Load reads the counters while the SMTP server updates them, which -race reports if they are not
// synchronised.
func TestLoadWhileCounting(t *testing.T) {
	origDB, origCheck := config.Database, config.DisableVersionCheck
	t.Cleanup(func() { config.Database, config.DisableVersionCheck = origDB, origCheck })

	config.Database = filepath.Join(t.TempDir(), "mailpit.db")
	config.DisableVersionCheck = true
	if err := storage.InitDB(); err != nil {
		t.Fatal(err)
	}
	// registered after TempDir, so it runs first: Windows cannot delete a database that is still open
	t.Cleanup(storage.Close)
	Track()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				LogSMTPAccepted(10)
				LogSMTPRejected()
				LogSMTPIgnored()
			}
		}
	}()

	var last uint64
	for range 100 {
		info := Load(true)
		if info.RuntimeStats.SMTPAccepted < last {
			t.Fatal("the accepted count went backwards")
		}
		last = info.RuntimeStats.SMTPAccepted
		if info.LatestVersion != "disabled" {
			t.Fatalf("latest version %q with the check disabled", info.LatestVersion)
		}
	}
	close(stop)
	wg.Wait()
}
