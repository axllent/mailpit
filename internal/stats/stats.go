// Package stats stores and returns Mailpit statistics
package stats

import (
	"runtime"
	"sync"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/storage"
	"github.com/axllent/mailpit/internal/updater"
)

var (
	// to prevent hammering Github for latest version
	latestVersionCache string

	// StartedAt is set to the current ime when Mailpit starts
	startedAt time.Time

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

// Load the current statistics
func Load() AppInformation {
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

	if latestVersionCache != "" {
		info.LatestVersion = latestVersionCache
	} else {
		latest, _, _, err := updater.GithubLatest(config.Repo, config.RepoBinaryName)
		if err == nil {
			info.LatestVersion = latest
			latestVersionCache = latest

			// clear latest version cache after 5 minutes
			go func() {
				time.Sleep(5 * time.Minute)
				latestVersionCache = ""
			}()
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
