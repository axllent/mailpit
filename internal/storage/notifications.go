package storage

import (
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/websockets"
)

var bcStatsDelay atomic.Bool

// BroadcastMailboxStats broadcasts the total number of messages
// displayed to the web UI, as well as the total unread messages.
// The lookup is very fast (< 10ms / 100k messages under load).
// Rate limited to 4x per second.
func BroadcastMailboxStats() {
	// only one broadcast may be pending at a time
	if !bcStatsDelay.CompareAndSwap(false, true) {
		return
	}

	go func() {
		time.Sleep(250 * time.Millisecond)
		bcStatsDelay.Store(false)
		b := struct {
			Total   uint64
			Unread  uint64
			Version string
		}{
			Total:   CountTotal(),
			Unread:  CountUnread(),
			Version: config.Version,
		}

		websockets.Broadcast("stats", b)
	}()
}
