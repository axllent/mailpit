package storage

import (
	"sync"

	"github.com/axllent/mailpit/data"
	"github.com/axllent/mailpit/logger"
	"github.com/ostafen/clover/v2"
)

var (
	mailboxStats = map[string]data.MailboxStats{}
	statsLock    = sync.RWMutex{}
)

// StatsGet returns the total/unread statistics for a mailbox
func StatsGet(mailbox string) data.MailboxStats {
	statsLock.Lock()
	defer statsLock.Unlock()
	s, ok := mailboxStats[mailbox]
	if !ok {
		return data.MailboxStats{
			Total:  0,
			Unread: 0,
		}
	}
	return s
}

// Refresh will completely refresh the existing stats for a given mailbox
func statsRefresh(mailbox string) error {
	logger.Log().Debugf("[stats] refreshing stats for %s", mailbox)

	total, err := db.Count(clover.NewQuery(mailbox))
	if err != nil {
		return err
	}

	unread, err := db.Count(clover.NewQuery(mailbox).Where(clover.Field("Read").IsFalse()))
	if err != nil {
		return nil
	}

	statsLock.Lock()
	mailboxStats[mailbox] = data.MailboxStats{
		Total:  total,
		Unread: unread,
	}
	statsLock.Unlock()

	return nil
}

func statsAddNewMessage(mailbox string) {
	statsLock.Lock()
	s, ok := mailboxStats[mailbox]
	if ok {
		mailboxStats[mailbox] = data.MailboxStats{
			Total:  s.Total + 1,
			Unread: s.Unread + 1,
		}
	}
	statsLock.Unlock()
}

// Delete one message from the totals. If the message was unread,
// then it will also deduct one from the Unread status.
func statsDeleteOneMessage(mailbox string, unread bool) {
	statsLock.Lock()
	s, ok := mailboxStats[mailbox]
	if ok {
		// deduct from the totals
		if s.Total > 0 {
			s.Total = s.Total - 1
		}
		// only deduct if the original was unread
		if unread && s.Unread > 0 {
			s.Unread = s.Unread - 1
		}

		mailboxStats[mailbox] = data.MailboxStats{
			Total:  s.Total,
			Unread: s.Unread,
		}
	}
	statsLock.Unlock()
}

// Mark one message as read
func statsReadOneMessage(mailbox string) {
	statsLock.Lock()
	s, ok := mailboxStats[mailbox]
	if ok {
		mailboxStats[mailbox] = data.MailboxStats{
			Total:  s.Total,
			Unread: s.Unread - 1,
		}
	}
	statsLock.Unlock()
}

// Mark one message as unread
func statsUnreadOneMessage(mailbox string) {
	statsLock.Lock()
	s, ok := mailboxStats[mailbox]
	if ok {
		mailboxStats[mailbox] = data.MailboxStats{
			Total:  s.Total,
			Unread: s.Unread + 1,
		}
	}
	statsLock.Unlock()
}
