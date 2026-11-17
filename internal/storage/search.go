package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/leporo/sqlf"
)

// MaxSearchTerms is the maximum number of terms (words, or quoted phrases) in a search.
const MaxSearchTerms = 100

// Search will search a mailbox for search terms.
// The search is broken up by segments (exact phrases can be quoted), and interprets specific terms such as:
// is:read, is:unread, has:attachment, to:<term>, from:<term> & subject:<term>
// Negative searches also also included by prefixing the search term with a `-` or `!`
func Search(search, timezone string, start int, beforeTS int64, limit int) ([]MessageSummary, int, error) {
	results := []MessageSummary{}
	tsStart := time.Now()
	nrResults := 0
	if limit < 0 {
		limit = 50
	}
	if start < 0 {
		start = 0
	}

	q, err := searchQueryBuilder(search, timezone)
	if err != nil {
		return results, nrResults, err
	}

	if beforeTS > 0 {
		q = q.Where(`m.Created < ?`, beforeTS)
	}

	// Every match is counted, but only the messages of the page are kept: a search which matches most of a
	// large mailbox would otherwise hold all of them (with their metadata) in memory for every request.
	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var created float64 // use float64 for rqlite compatibility
		var id string
		var messageID string
		var subject string
		var metadata string
		var size float64 // use float64 for rqlite compatibility
		var attachments int
		var snippet string
		var read int
		var ignore string
		em := MessageSummary{}

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &snippet, &ignore, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

		position := nrResults
		nrResults++

		if position < start || position-start >= limit {
			return
		}

		if err := json.Unmarshal([]byte(metadata), &em); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

		em.Created = time.UnixMilli(int64(created))
		em.ID = id
		em.MessageID = messageID
		em.Subject = subject
		em.Size = uint64(size)
		em.Attachments = attachments
		em.Read = read == 1
		em.Snippet = snippet

		results = append(results, em)
	}); err != nil {
		return []MessageSummary{}, 0, err
	}

	dbLastAction.touch()

	// set tags for listed messages only
	if len(results) > 0 {
		ids := make([]string, len(results))
		for i, m := range results {
			ids[i] = m.ID
		}
		tagMap := getTagsForIDs(ids)
		for i, m := range results {
			if tags, ok := tagMap[m.ID]; ok {
				results[i].Tags = tags
			} else {
				results[i].Tags = []string{}
			}
		}
	}

	elapsed := time.Since(tsStart)

	logger.Log().Debugf("[db] search for \"%s\" in %s", search, elapsed)

	return results, nrResults, err
}

// SearchUnreadCount returns the number of unread messages matching a search.
// This is run one at a time to allow connected browsers to be updated.
func SearchUnreadCount(search, timezone string, beforeTS int64) (int64, error) {
	tsStart := time.Now()

	q, err := searchQueryBuilder(search, timezone)
	if err != nil {
		return 0, err
	}

	if beforeTS > 0 {
		q = q.Where(`m.Created < ?`, beforeTS)
	}

	var unread float64 // use float64 for rqlite compatibility

	q = q.Where("m.Read = 0").Select(`COUNT(*)`)

	err = q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var ignore sql.NullString
		if err := row.Scan(&ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &unread); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

	})

	dbLastAction.touch()

	elapsed := time.Since(tsStart)

	logger.Log().Debugf("[db] counted %d unread for \"%s\" in %s", int64(unread), search, elapsed)

	return int64(unread), err
}

// DeleteSearch will delete all messages for search terms.
// The search is broken up by segments (exact phrases can be quoted), and interprets specific terms such as:
// is:read, is:unread, has:attachment, to:<term>, from:<term> & subject:<term>
// Negative searches also also included by prefixing the search term with a `-` or `!`
func DeleteSearch(search, timezone string) error {
	q, err := searchQueryBuilder(search, timezone)
	if err != nil {
		return err
	}

	ids := []string{}
	deleteSize := uint64(0)

	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var created float64 // use float64 for rqlite compatibility
		var id string
		var messageID string
		var subject string
		var metadata string
		var size float64 // use float64 for rqlite compatibility
		var attachments int
		var read int
		var snippet string
		var ignore string

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &snippet, &ignore, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

		ids = append(ids, id)
		deleteSize = deleteSize + uint64(size)
	}); err != nil {
		return err
	}

	if len(ids) > 0 {
		total := len(ids)

		// split ids into chunks, as a statement can only have so many placeholders (this must not change
		// ids, which are used again below)
		chunks := chunkBy(ids, maxSQLVariables)

		// begin a transaction to ensure both the message
		// and data are deleted successfully
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		// roll back if it fails
		defer func() { _ = tx.Rollback() }()

		for _, ids := range chunks {
			delIDs := make([]any, len(ids))
			for i, id := range ids {
				delIDs[i] = id
			}

			sqlDelete1 := `DELETE FROM ` + tenant("mailbox") + ` WHERE ID IN (?` + strings.Repeat(",?", len(ids)-1) + `)` // #nosec

			_, err = tx.Exec(sqlDelete1, delIDs...)
			if err != nil {
				return err
			}

			sqlDelete2 := `DELETE FROM ` + tenant("mailbox_data") + ` WHERE ID IN (?` + strings.Repeat(",?", len(ids)-1) + `)` // #nosec

			_, err = tx.Exec(sqlDelete2, delIDs...)
			if err != nil {
				return err
			}

			sqlDelete3 := `DELETE FROM ` + tenant("message_tags") + ` WHERE ID IN (?` + strings.Repeat(",?", len(ids)-1) + `)` // #nosec

			_, err = tx.Exec(sqlDelete3, delIDs...)
			if err != nil {
				return err
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		if err := pruneUnusedTags(); err != nil {
			return err
		}

		logger.Log().Debugf("[db] deleted %d messages matching %s", total, search)

		dbLastAction.touch()

		// broadcast changes
		if total > 200 {
			websockets.Broadcast("prune", nil)
		} else {
			for _, id := range ids {
				d := struct {
					ID string
				}{ID: id}
				websockets.Broadcast("delete", d)
			}
		}

		for _, id := range ids {
			invalidateEnvelopeCache(id)
		}

		addDeletedSize(deleteSize)

		logMessagesDeleted(total)

		BroadcastMailboxStats()
	}

	return nil
}

// SetSearchReadStatus marks all messages matching the search as read or unread
func SetSearchReadStatus(search, timezone string, read bool) error {
	q, err := searchQueryBuilder(search, timezone)
	if err != nil {
		return err
	}

	q = q.Where("m.Read = ?", !read)

	ids := []string{}

	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var created float64 // use float64 for rqlite compatibility
		var id string
		var messageID string
		var subject string
		var metadata string
		var size float64 // use float64 for rqlite compatibility
		var attachments int
		var read int
		var snippet string
		var ignore string

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &snippet, &ignore, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

		ids = append(ids, id)
	}); err != nil {
		return err
	}

	if read {
		if err := MarkRead(ids); err != nil {
			return err
		}
	} else {
		if err := MarkUnread(ids); err != nil {
			return err
		}
	}

	return nil
}

// MetaSearchTextField returns the SQL expression for a searchable address field (eg: To). It uses
// the case-folded copy in MetaSearchText, as search terms are case-folded and SQLite's LIKE only
// folds ASCII case. Messages without MetaSearchText (see backfillMetaSearchText) use the
// original metadata.
func metaSearchTextField(field string) string {
	return fmt.Sprintf(`IFNULL(json_extract(m.MetaSearchText, '$.%[1]s'), IFNULL(json_extract(m.Metadata, '$.%[1]s'), '{}'))`, field)
}

// SearchParser returns the SQL syntax for the database search based on the search arguments
func searchQueryBuilder(searchString, timezone string) (*sqlf.Stmt, error) {
	// group strings with quotes as a single argument and remove quotes
	args := tools.ArgsParser(searchString)

	// Every term is a condition of the query, which costs memory to build and time to run (and SQLite
	// refuses more than a thousand levels of them anyway). This is an error rather than a limit on the
	// terms which are used, because dropping terms from a search which deletes messages would delete more
	// than was asked for.
	if len(args) > MaxSearchTerms {
		return nil, fmt.Errorf("a search can have at most %d terms, %d were given", MaxSearchTerms, len(args))
	}

	loc := time.Local
	if timezone != "" {
		if l, err := time.LoadLocation(timezone); err != nil {
			logger.Log().Warnf("ignoring invalid timezone:\"%s\"", timezone)
		} else {
			loc = l
		}
	}

	q := sqlf.From(tenant("mailbox") + " m").
		Select(`m.Created, m.ID, m.MessageID, m.Subject, m.Metadata, m.Size, m.Attachments, m.Read,
			m.Snippet,
			` + metaSearchTextField("To") + ` as ToJSON,
			` + metaSearchTextField("From") + ` as FromJSON,
			` + metaSearchTextField("Cc") + ` as CcJSON,
			` + metaSearchTextField("Bcc") + ` as BccJSON,
			` + metaSearchTextField("ReplyTo") + ` as ReplyToJSON
		`).
		OrderBy("m.Created DESC")

	for _, w := range args {
		if cleanString(w) == "" {
			continue
		}

		// lowercase search to try match search prefixes
		lw := strings.ToLower(w)

		exclude := false
		// search terms starting with a `-` or `!` imply an exclude
		if len(w) > 1 && (strings.HasPrefix(w, "-") || strings.HasPrefix(w, "!")) {
			exclude = true
			w = w[1:]
			lw = lw[1:]
		}

		// ignore blank searches
		if len(w) == 0 {
			continue
		}

		if strings.HasPrefix(lw, "to:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("ToJSON NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				} else {
					q.Where("ToJSON LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				}
			}
		} else if strings.HasPrefix(lw, "from:") {
			w = cleanString(w[5:])
			if w != "" {
				if exclude {
					q.Where("FromJSON NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				} else {
					q.Where("FromJSON LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				}
			}
		} else if strings.HasPrefix(lw, "cc:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("CcJSON NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				} else {
					q.Where("CcJSON LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				}
			}
		} else if strings.HasPrefix(lw, "bcc:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where("BccJSON NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				} else {
					q.Where("BccJSON LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				}
			}
		} else if strings.HasPrefix(lw, "reply-to:") {
			w = cleanString(w[9:])
			if w != "" {
				if exclude {
					q.Where("ReplyToJSON NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				} else {
					q.Where("ReplyToJSON LIKE ? ESCAPE '\\'", "%"+escLikeChars(escJSONChars(w))+"%")
				}
			}
		} else if strings.HasPrefix(lw, "addressed:") {
			w = cleanString(w[10:])
			arg := "%" + escLikeChars(escJSONChars(w)) + "%"
			if w != "" {
				if exclude {
					q.Where("(ToJSON NOT LIKE ? ESCAPE '\\' AND FromJSON NOT LIKE ? ESCAPE '\\' AND CcJSON NOT LIKE ? ESCAPE '\\' AND BccJSON NOT LIKE ? ESCAPE '\\' AND ReplyToJSON NOT LIKE ? ESCAPE '\\')", arg, arg, arg, arg, arg)
				} else {
					q.Where("(ToJSON LIKE ? ESCAPE '\\' OR FromJSON LIKE ? ESCAPE '\\' OR CcJSON LIKE ? ESCAPE '\\' OR BccJSON LIKE ? ESCAPE '\\' OR ReplyToJSON LIKE ? ESCAPE '\\')", arg, arg, arg, arg, arg)
				}
			}
		} else if strings.HasPrefix(lw, "subject:") {
			w = foldCase(w[8:])
			if w != "" {
				// messages without MetaSearchText (see backfillMetaSearchText) use the original subject
				if exclude {
					q.Where("IFNULL(json_extract(m.MetaSearchText, '$.Subject'), m.Subject) NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(w)+"%")
				} else {
					q.Where("IFNULL(json_extract(m.MetaSearchText, '$.Subject'), m.Subject) LIKE ? ESCAPE '\\'", "%"+escLikeChars(w)+"%")
				}
			}
		} else if strings.HasPrefix(lw, "message-id:") {
			w = cleanString(w[11:])
			if w != "" {
				if exclude {
					q.Where("m.MessageID NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(w)+"%")
				} else {
					q.Where("m.MessageID LIKE ? ESCAPE '\\'", "%"+escLikeChars(w)+"%")
				}
			}
		} else if strings.HasPrefix(lw, "tag:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where(`m.ID NOT IN (SELECT mt.ID FROM `+tenant("message_tags")+` mt JOIN `+tenant("tags")+` t ON mt.TagID = t.ID WHERE t.Name = ?)`, w)
				} else {
					q.Where(`m.ID IN (SELECT mt.ID FROM `+tenant("message_tags")+` mt JOIN `+tenant("tags")+` t ON mt.TagID = t.ID WHERE t.Name = ?)`, w)
				}
			}
		} else if lw == "is:read" {
			if exclude {
				q.Where("m.Read = 0")
			} else {
				q.Where("m.Read = 1")
			}
		} else if lw == "is:unread" {
			if exclude {
				q.Where("m.Read = 1")
			} else {
				q.Where("m.Read = 0")
			}
		} else if lw == "is:tagged" {
			if exclude {
				q.Where(`m.ID NOT IN (SELECT DISTINCT mt.ID FROM ` + tenant("message_tags") + ` mt JOIN ` + tenant("tags") + ` t ON mt.TagID = t.ID)`)
			} else {
				q.Where(`m.ID IN (SELECT DISTINCT mt.ID FROM ` + tenant("message_tags") + ` mt JOIN ` + tenant("tags") + ` t ON mt.TagID = t.ID)`)
			}
		} else if lw == "has:inline" || lw == "has:inlines" {
			if exclude {
				q.Where("m.Inline = 0")
			} else {
				q.Where("m.Inline > 0")
			}
		} else if lw == "has:attachment" || lw == "has:attachments" {
			if exclude {
				q.Where("m.Attachments = 0")
			} else {
				q.Where("m.Attachments > 0")
			}
		} else if strings.HasPrefix(lw, "after:") {
			w = strings.ToUpper(cleanString(w[6:]))
			if w != "" {
				t, err := dateparse.ParseIn(w, loc)
				if err != nil {
					logger.Log().Warnf("ignoring invalid after: date \"%s\"", w)
				} else {
					timestamp := t.UnixMilli()
					if exclude {
						q.Where(`m.Created < ?`, timestamp)
					} else {
						q.Where(`m.Created >= ?`, timestamp)
					}
				}
			}
		} else if strings.HasPrefix(lw, "before:") {
			w = strings.ToUpper(cleanString(w[7:]))
			if w != "" {
				t, err := dateparse.ParseIn(w, loc)
				if err != nil {
					logger.Log().Warnf("ignoring invalid before: date \"%s\"", w)
				} else {
					timestamp := t.UnixMilli()
					if exclude {
						q.Where(`m.Created >= ?`, timestamp)
					} else {
						q.Where(`m.Created < ?`, timestamp)
					}
				}
			}
		} else if strings.HasPrefix(lw, "larger:") && sizeToBytes(cleanString(w[7:])) > 0 {
			w = cleanString(w[7:])
			size := sizeToBytes(w)
			if exclude {
				q.Where("m.Size < ?", size)
			} else {
				q.Where("m.Size >= ?", size)
			}
		} else if strings.HasPrefix(lw, "smaller:") && sizeToBytes(cleanString(w[8:])) > 0 {
			w = cleanString(w[8:])
			size := sizeToBytes(w)
			if exclude {
				q.Where("m.Size >= ?", size)
			} else {
				q.Where("m.Size < ?", size)
			}
		} else {
			// search text
			if exclude {
				q.Where("m.SearchText NOT LIKE ? ESCAPE '\\'", "%"+escLikeChars(cleanString(w))+"%")
			} else {
				q.Where("m.SearchText LIKE ? ESCAPE '\\'", "%"+escLikeChars(cleanString(w))+"%")
			}
		}
	}

	return q, nil
}

// sizeRe matches a size such as 2kb, 4MB or 1.5m
var sizeRe = regexp.MustCompile(`^(\d+)(\.\d+)?\s?([a-z]{1,2})?$`)

// Simple function to return a size in bytes, eg 2kb, 4MB or 1.5m.
//
// K, k, Kb, KB, kB and kb are treated as Kilobytes.
// M, m, Mb, MB and mb are treated as Megabytes.
//
// A size is at most math.MaxInt64, which is more than any message can be and the most a database can
// compare with (a larger number made the search fail).
func sizeToBytes(v string) uint64 {
	v = strings.ToLower(v)

	m := sizeRe.FindAllStringSubmatch(v, -1)
	if len(m) == 0 {
		return 0
	}

	val := fmt.Sprintf("%s%s", m[0][1], m[0][2])
	unit := m[0][3]

	i, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0
	}

	switch unit {
	case "":
	case "k", "kb":
		i *= 1024
	case "m", "mb":
		i *= 1024 * 1024
	default:
		return 0
	}

	// the float is converted only when it fits, as the result of converting one which does not is undefined
	if i >= math.MaxInt64 {
		return math.MaxInt64
	}

	return uint64(i)
}
