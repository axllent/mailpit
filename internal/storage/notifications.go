package storage

import (
	"time"

	"github.com/axllent/mailpit/server/websockets"
)

var bcStatsDelay = false

// BroadcastMailboxStats broadcasts the total number of messages
// displayed to the web UI, as well as the total unread messages.
// The lookup is very fast (< 10ms / 100k messages under load).
// Rate limited to 4x per second.
func BroadcastMailboxStats() {
	if bcStatsDelay {
		return
	}

	bcStatsDelay = true

	go func() {
		time.Sleep(250 * time.Millisecond)
		bcStatsDelay = false
		b := struct {
			Total  int
			Unread int
		}{
			Total:  CountTotal(),
			Unread: CountUnread(),
		}

		websockets.Broadcast("stats", b)
	}()
}
