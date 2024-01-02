// Package stats stores and returns Mailpit statistics
package stats

import (
	"os"
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

	smtpReceived     int
	smtpReceivedSize int
	smtpErrors       int
	smtpIgnored      int
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
	DatabaseSize int64
	// Total number of messages in the database
	Messages int
	// Total number of messages in the database
	Unread int
	// Tags and message totals per tag
	Tags map[string]int64
	// Runtime statistics
	RuntimeStats struct {
		// Mailpit server uptime in seconds
		Uptime int
		// Current memory usage in bytes
		Memory uint64
		// Messages deleted
		MessagesDeleted int
		// SMTP messages received via since run
		SMTPReceived int
		// Total size in bytes of received messages since run
		SMTPReceivedSize int
		// SMTP errors since run
		SMTPErrors int
		// SMTP messages ignored since run (duplicate IDs)
		SMTPIgnored int
	}
}

// Load the current statistics
func Load() AppInformation {
	info := AppInformation{}
	info.Version = config.Version

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info.RuntimeStats.Memory = m.Sys - m.HeapReleased

	info.RuntimeStats.Uptime = int(time.Since(startedAt).Seconds())
	info.RuntimeStats.MessagesDeleted = storage.StatsDeleted
	info.RuntimeStats.SMTPReceived = smtpReceived
	info.RuntimeStats.SMTPReceivedSize = smtpReceivedSize
	info.RuntimeStats.SMTPErrors = smtpErrors
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

	info.Database = config.DataFile

	db, err := os.Stat(info.Database)
	if err == nil {
		info.DatabaseSize = db.Size()
	}

	info.Messages = storage.CountTotal()
	info.Unread = storage.CountUnread()

	info.Tags = storage.GetAllTagsCount()

	return info
}

// Track will start the statistics logging in memory
func Track() {
	startedAt = time.Now()
}

// LogSMTPReceived logs a successfully SMTP transaction
func LogSMTPReceived(size int) {
	mu.Lock()
	smtpReceived = smtpReceived + 1
	smtpReceivedSize = smtpReceivedSize + size
	mu.Unlock()
}

// LogSMTPError logs a failed SMTP transaction
func LogSMTPError() {
	mu.Lock()
	smtpErrors = smtpErrors + 1
	mu.Unlock()
}

// LogSMTPIgnored logs an ignored SMTP transaction
func LogSMTPIgnored() {
	mu.Lock()
	smtpIgnored = smtpIgnored + 1
	mu.Unlock()
}
