// Package stats stores and returns Mailpit statistics
package stats

import (
	"runtime"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/storage"
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
}

var (
	// Version cache storing the latest GitHub version
	vCache versionCache

	// StartedAt is set to the current ime when Mailpit starts
	startedAt time.Time

	// sync mutex to prevent race condition with simultaneous requests
	mu sync.RWMutex

	smtpAccepted     uint64
	smtpAcceptedSize uint64
	smtpRejected     uint64
	smtpIgnored      uint64
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

// Calculates exponential backoff duration based on the error count.
func getBackoff(errCount int) time.Duration {
	backoff := min(time.Duration(1<<errCount)*time.Minute, 30*time.Minute)
	return backoff
}

// Load the current statistics
func Load(detectLatestVersion bool) AppInformation {
	info := AppInformation{}
	info.Version = config.Version

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info.RuntimeStats.Memory = m.Sys - m.HeapReleased
	info.RuntimeStats.Uptime = uint64(time.Since(startedAt).Seconds())
	info.RuntimeStats.MessagesDeleted = storage.StatsDeleted
	info.RuntimeStats.SMTPAccepted = smtpAccepted
	info.RuntimeStats.SMTPAcceptedSize = smtpAcceptedSize
	info.RuntimeStats.SMTPRejected = smtpRejected
	info.RuntimeStats.SMTPIgnored = smtpIgnored

	if config.DisableVersionCheck {
		info.LatestVersion = "disabled"
	} else if detectLatestVersion {
		mu.RLock()
		cacheValid := time.Now().Before(vCache.expiry)
		cacheValue := vCache.value
		mu.RUnlock()

		if cacheValid {
			info.LatestVersion = cacheValue
		} else {
			mu.Lock()
			// Re-check after acquiring write lock in case another goroutine refreshed it
			if time.Now().Before(vCache.expiry) {
				info.LatestVersion = vCache.value
			} else {
				latest, err := config.GHRUConfig.Latest()
				if err == nil {
					vCache = versionCache{value: latest.Tag, expiry: time.Now().Add(15 * time.Minute)}
					info.LatestVersion = latest.Tag
				} else {
					logger.Log().Errorf("Failed to fetch latest version: %v", err)
					vCache.errCount++
					vCache.value = ""
					vCache.expiry = time.Now().Add(getBackoff(vCache.errCount))
					info.LatestVersion = ""
				}
			}
			mu.Unlock()
		}
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
	mu.Lock()
	smtpAccepted = smtpAccepted + 1
	smtpAcceptedSize = smtpAcceptedSize + uint64(size)
	mu.Unlock()
}

// LogSMTPRejected logs a rejected SMTP transaction
func LogSMTPRejected() {
	mu.Lock()
	smtpRejected = smtpRejected + 1
	mu.Unlock()
}

// LogSMTPIgnored logs an ignored SMTP transaction
func LogSMTPIgnored() {
	mu.Lock()
	smtpIgnored = smtpIgnored + 1
	mu.Unlock()
}
