package storage

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/websockets"
)

// bcStatsDelay holds an *atomic.Bool per segment
var bcStatsDelay sync.Map

// BroadcastMailboxStats broadcasts the total number of messages
// displayed to the web UI, as well as the total unread messages.
// The lookup is very fast (< 10ms / 100k messages under load).
// Rate limited to 4x per second per segment.
func BroadcastMailboxStats(seg string) {
	v, _ := bcStatsDelay.LoadOrStore(seg, &atomic.Bool{})
	delay := v.(*atomic.Bool)

	// only one broadcast may be pending at a time
	if !delay.CompareAndSwap(false, true) {
		return
	}

	go func() {
		time.Sleep(250 * time.Millisecond)
		delay.Store(false)
		b := struct {
			Total   uint64
			Unread  uint64
			Version string
		}{
			Total:   CountTotal(seg),
			Unread:  CountUnread(seg),
			Version: config.Version,
		}

		websockets.BroadcastSegment(seg, "stats", b)
	}()
}
