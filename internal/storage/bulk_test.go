package storage

import (
	"fmt"
	"testing"
	"time"

	"github.com/axllent/mailpit/config"
)

// seedMessages inserts n minimal messages straight into the tables (the tests which need more messages than
// a statement can have placeholders for would take far too long going through Store).
func seedMessages(t *testing.T, n int, read bool) []string {
	t.Helper()

	r := 0
	if read {
		r = 1
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()

	m, err := tx.Prepare(fmt.Sprintf(`INSERT INTO %s (Created, ID, MessageID, Subject, Metadata, Size, Inline, Attachments, SearchText, MetaSearchText, Read, Snippet) VALUES(?,?,?,?,?,?,0,0,?,?,?,?)`, tenant("mailbox")))
	if err != nil {
		t.Fatal(err)
	}
	d, err := tx.Prepare(fmt.Sprintf(`INSERT INTO %s (ID, Email, Compressed) VALUES(?, ?, 0)`, tenant("mailbox_data")))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UnixMilli()
	ids := make([]string, n)

	for i := range n {
		id := fmt.Sprintf("bulk%07d", i)
		ids[i] = id

		if _, err := m.Exec(now-int64(i), id, "m"+id, "subject", "{}", 100, "common "+id, "{}", r, "snippet"); err != nil {
			t.Fatal(err)
		}
		if _, err := d.Exec(id, "x"); err != nil {
			t.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	return ids
}

// More IDs than SQLite allows placeholders (32,766) used to fail with "too many SQL variables", and nothing
// was changed. This uses SQLite's real limit, so it is slow with the race detector, and the other operations
// are tested with a short chunk of IDs.
func TestBulkOperationsBeyondTheSQLiteLimit(t *testing.T) {
	if raceEnabled {
		t.Skip("inserting 33,000 messages takes over a minute with the race detector")
	}

	setup("")

	const many = 33000

	ids := seedMessages(t, many, false)

	if err := MarkRead(ids); err != nil {
		t.Fatal(err)
	}
	if got := CountUnread(); got != 0 {
		t.Errorf("%d are still unread", got)
	}

	if err := MarkUnread(ids); err != nil {
		t.Fatal(err)
	}
	if got := CountUnread(); got != many {
		t.Errorf("%d are unread, want %d", got, many)
	}

	if err := DeleteMessages(ids); err != nil {
		t.Fatal(err)
	}
	if got := CountTotal(); got != 0 {
		t.Errorf("%d messages are left", got)
	}
}

// Every operation which takes a list of IDs works however long the list is. A chunk of a few IDs is used
// so that it is longer than a chunk with a small number of messages.
func TestBulkOperationsInChunks(t *testing.T) {
	setup("")

	orig := maxSQLVariables
	maxSQLVariables = 7
	t.Cleanup(func() { maxSQLVariables = orig })

	const many = 60 // not a multiple of the chunk

	ids := seedMessages(t, many, false)

	t.Run("mark read and unread", func(t *testing.T) {
		if err := MarkRead(ids); err != nil {
			t.Fatal(err)
		}
		if got := CountUnread(); got != 0 {
			t.Errorf("%d are still unread", got)
		}

		if err := MarkUnread(ids); err != nil {
			t.Fatal(err)
		}
		if got := CountUnread(); got != many {
			t.Errorf("%d are unread, want %d", got, many)
		}
	})

	t.Run("mark all the results of a search", func(t *testing.T) {
		if err := SetSearchReadStatus("is:unread", "", true); err != nil {
			t.Fatal(err)
		}
		if got := CountUnread(); got != 0 {
			t.Errorf("%d are still unread", got)
		}
		if err := SetSearchReadStatus("is:read", "", false); err != nil {
			t.Fatal(err)
		}
		if got := CountUnread(); got != many {
			t.Errorf("%d are unread, want %d", got, many)
		}
	})

	t.Run("tags of many messages", func(t *testing.T) {
		if _, err := SetMessageTags(ids[many-1], []string{"last"}); err != nil {
			t.Fatal(err)
		}
		tags := getTagsForIDs(ids)
		if len(tags[ids[many-1]]) != 1 || tags[ids[many-1]][0] != "last" {
			t.Errorf("the tags of the last message are %v", tags[ids[many-1]])
		}
	})

	t.Run("delete", func(t *testing.T) {
		// a message which does not exist, and one that is repeated, are not counted
		all := append(append([]string{"does-not-exist", ids[0]}, ids...), ids[1])
		if err := DeleteMessages(all); err != nil {
			t.Fatal(err)
		}
		if got := CountTotal(); got != 0 {
			t.Errorf("%d messages are left", got)
		}
		if tags := GetAllTags(); len(tags) != 0 {
			t.Errorf("unused tags are left: %v", tags)
		}
	})

	// control: a short list works as it did
	t.Run("control: a few IDs", func(t *testing.T) {
		few := seedMessages(t, 5, false)
		if err := MarkRead(few[:2]); err != nil {
			t.Fatal(err)
		}
		if got := CountUnread(); got != 3 {
			t.Errorf("%d are unread, want 3", got)
		}
		if err := DeleteMessages(few[:3]); err != nil {
			t.Fatal(err)
		}
		if got := CountTotal(); got != 2 {
			t.Errorf("%d messages, want 2", got)
		}
	})
}

// Pruning deletes up to 5,000 messages at once, which is more than the IDs of one chunk.
func TestPruneDeletesInParts(t *testing.T) {
	setup("")
	t.Cleanup(func() { config.MaxMessages = 0 })

	orig := maxSQLVariables
	maxSQLVariables = 100
	t.Cleanup(func() { maxSQLVariables = orig })

	seedMessages(t, 2501, false)

	config.MaxMessages = 1
	pruneMessages()

	if got := CountTotal(); got != 1 {
		t.Errorf("%d messages are left, want 1", got)
	}
}
