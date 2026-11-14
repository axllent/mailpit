// Package stats stores and returns Mailpit statistics
package stats

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/tools"
)

// Stores cached version  along with its expiry time and error count.
// Used to minimize repeated version lookups and track consecutive errors.
type versionCache struct {
	// github version string
	value string
	// time to expire the cache
	expiry time.Time
	// count of consecutive errors
	errCount int
	// whether a lookup is in progress
	refreshing bool
}

// maxBackoff is the longest time between version lookups after errors.
const maxBackoff = 30 * time.Minute

// fetchLatest looks up the latest release version. It is a variable so that the tests can replace it.
var fetchLatest = func() (string, error) {
	latest, err := config.GHRUConfig.Latest()
	if err != nil {
		return "", err
	}

	return latest.Tag, nil
}

var (
	// Version cache storing the latest GitHub version
	vCache versionCache

	// StartedAt is set to the current ime when Mailpit starts
	startedAt time.Time

	// versionMu protects vCache. It is never held while looking up the version.
	versionMu sync.Mutex

	// the SMTP counters, which are updated by the SMTP server while being read here
	smtpAccepted     atomic.Uint64
	smtpAcceptedSize atomic.Uint64
	smtpRejected     atomic.Uint64
	smtpIgnored      atomic.Uint64
)

// AppInformation struct
// swagger:model AppInformation
type AppInformation struct {
	// Current Mailpit version
	Version string
	// Latest Mailpit version
	LatestVersion string
	// Database path
	Database string
	// Database size in bytes
	DatabaseSize uint64
	// Total number of messages in the database
	Messages uint64
	// Total number of messages in the database
	Unread uint64
	// Tags and message totals per tag
	Tags map[string]int64
	// Runtime statistics
	RuntimeStats struct {
		// Mailpit server uptime in seconds
		Uptime uint64
		// Current memory usage in bytes
		Memory uint64
		// Database runtime messages deleted
		MessagesDeleted uint64
		// Accepted runtime SMTP messages
		SMTPAccepted uint64
		// Total runtime accepted messages size in bytes
		SMTPAcceptedSize uint64
		// Rejected runtime SMTP messages
		SMTPRejected uint64
		// Ignored runtime SMTP messages (when using --ignore-duplicate-ids)
		SMTPIgnored uint64
	}
}

// Calculates exponential backoff duration based on the error count, from 2 minutes up to maxBackoff.
func getBackoff(errCount int) time.Duration {
	// the count is limited before it is shifted, as a large one overflows (and goes negative)
	shift := min(max(errCount, 0), 5)

	return min(time.Duration(1<<shift)*time.Minute, maxBackoff)
}

// latestVersion returns the cached latest version, looking it up when the cache has expired. The
// lookup is a network request which can take as long as its timeout, so it is made without holding
// the lock, and a request that arrives during it gets the current cached value instead of waiting.
func latestVersion() string {
	versionMu.Lock()
	if time.Now().Before(vCache.expiry) || vCache.refreshing {
		value := vCache.value
		versionMu.Unlock()

		return value
	}
	vCache.refreshing = true
	versionMu.Unlock()

	tag, err := fetchLatest()

	versionMu.Lock()
	defer versionMu.Unlock()

	vCache.refreshing = false

	if err == nil {
		vCache.value = tag
		vCache.errCount = 0
		vCache.expiry = time.Now().Add(15 * time.Minute)

		return tag
	}

	logger.Log().Errorf("Failed to fetch latest version: %v", err)
	vCache.errCount++
	vCache.value = ""
	vCache.expiry = time.Now().Add(getBackoff(vCache.errCount))

	return ""
}

// Load the current statistics
func Load(detectLatestVersion bool) AppInformation {
	info := AppInformation{}
	info.Version = config.Version

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info.RuntimeStats.Memory = m.Sys - m.HeapReleased
	info.RuntimeStats.Uptime = uint64(time.Since(startedAt).Seconds())
	info.RuntimeStats.MessagesDeleted = storage.StatsDeleted.Load()
	info.RuntimeStats.SMTPAccepted = smtpAccepted.Load()
	info.RuntimeStats.SMTPAcceptedSize = smtpAcceptedSize.Load()
	info.RuntimeStats.SMTPRejected = smtpRejected.Load()
	info.RuntimeStats.SMTPIgnored = smtpIgnored.Load()

	if config.DisableVersionCheck {
		info.LatestVersion = "disabled"
	} else if detectLatestVersion {
		info.LatestVersion = latestVersion()
	}

	info.Database = config.Database
	info.DatabaseSize = storage.DbSize()
	info.Messages = storage.CountTotal()
	info.Unread = storage.CountUnread()
	info.Tags = storage.GetAllTagsCount()

	return info
}

// Track will start the statistics logging in memory
func Track() {
	startedAt = time.Now()
}

// LogSMTPAccepted logs a successful SMTP transaction
func LogSMTPAccepted(size int) {
	smtpAccepted.Add(1)
	smtpAcceptedSize.Add(tools.SafeUint64(size))
}

// LogSMTPRejected logs a rejected SMTP transaction
func LogSMTPRejected() {
	smtpRejected.Add(1)
}

// LogSMTPIgnored logs an ignored SMTP transaction
func LogSMTPIgnored() {
	smtpIgnored.Add(1)
}
