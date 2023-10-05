package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/leporo/sqlf"
)

// Search will search a mailbox for search terms.
// The search is broken up by segments (exact phrases can be quoted), and interprets specific terms such as:
// is:read, is:unread, has:attachment, to:<term>, from:<term> & subject:<term>
// Negative searches also also included by prefixing the search term with a `-` or `!`
func Search(search string, start, limit int) ([]MessageSummary, int, error) {
	results := []MessageSummary{}
	allResults := []MessageSummary{}
	tsStart := time.Now()
	nrResults := 0
	if limit < 0 {
		limit = 50
	}

	q := searchQueryBuilder(search)
	var err error

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var created int64
		var id string
		var messageID string
		var subject string
		var metadata string
		var size int
		var attachments int
		var tags string
		var snippet string
		var read int
		var ignore string
		em := MessageSummary{}

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &tags, &snippet, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(metadata), &em); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(tags), &em.Tags); err != nil {
			logger.Log().Error(err)
			return
		}

		em.Created = time.UnixMilli(created)
		em.ID = id
		em.MessageID = messageID
		em.Subject = subject
		em.Size = size
		em.Attachments = attachments
		em.Read = read == 1
		em.Snippet = snippet

		allResults = append(allResults, em)
	}); err != nil {
		return results, nrResults, err
	}

	dbLastAction = time.Now()

	nrResults = len(allResults)

	if nrResults > start {
		end := nrResults
		if nrResults >= start+limit {
			end = start + limit
		}

		results = allResults[start:end]
	}

	elapsed := time.Since(tsStart)

	logger.Log().Debugf("[db] search for \"%s\" in %s", search, elapsed)

	return results, nrResults, err
}

// DeleteSearch will delete all messages for search terms.
// The search is broken up by segments (exact phrases can be quoted), and interprets specific terms such as:
// is:read, is:unread, has:attachment, to:<term>, from:<term> & subject:<term>
// Negative searches also also included by prefixing the search term with a `-` or `!`
func DeleteSearch(search string) error {
	q := searchQueryBuilder(search)

	ids := []string{}

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var created int64
		var id string
		var messageID string
		var subject string
		var metadata string
		var size int
		var attachments int
		var tags string
		var read int
		var snippet string
		var ignore string

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &tags, &snippet, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Error(err)
			return
		}

		ids = append(ids, id)
	}); err != nil {
		return err
	}

	if len(ids) > 0 {
		total := len(ids)

		// split ids into chunks of 1000 ids
		var chunks [][]string
		if total > 1000 {
			chunkSize := 1000
			chunks = make([][]string, 0, (len(ids)+chunkSize-1)/chunkSize)
			for chunkSize < len(ids) {
				ids, chunks = ids[chunkSize:], append(chunks, ids[0:chunkSize:chunkSize])
			}
			if len(ids) > 0 {
				// add remaining ids <= 1000
				chunks = append(chunks, ids)
			}
		} else {
			chunks = append(chunks, ids)
		}

		// begin a transaction to ensure both the message
		// and data are deleted successfully
		tx, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		// roll back if it fails
		defer tx.Rollback()

		for _, ids := range chunks {
			delIDs := make([]interface{}, len(ids))
			for i, id := range ids {
				delIDs[i] = id
			}

			sqlDelete1 := `DELETE FROM mailbox WHERE ID IN (?` + strings.Repeat(",?", len(ids)-1) + `)`

			_, err = tx.Exec(sqlDelete1, delIDs...)
			if err != nil {
				return err
			}

			sqlDelete2 := `DELETE FROM mailbox_data WHERE ID IN (?` + strings.Repeat(",?", len(ids)-1) + `)`

			_, err = tx.Exec(sqlDelete2, delIDs...)
			if err != nil {
				return err
			}
		}

		err = tx.Commit()

		if err == nil {
			logger.Log().Debugf("[db] deleted %d messages matching %s", total, search)
		}

		dbLastAction = time.Now()
		dbDataDeleted = true

		BroadcastMailboxStats()
	}

	return nil
}

// SearchParser returns the SQL syntax for the database search based on the search arguments
func searchQueryBuilder(searchString string) *sqlf.Stmt {
	searchString = strings.ToLower(searchString)
	// group strings with quotes as a single argument and remove quotes
	args := tools.ArgsParser(searchString)

	q := sqlf.From("mailbox").
		Select(`Created, ID, MessageID, Subject, Metadata, Size, Attachments, Read, Tags, Snippet,
			IFNULL(json_extract(Metadata, '$.To'), '{}') as ToJSON,
			IFNULL(json_extract(Metadata, '$.From'), '{}') as FromJSON,
			IFNULL(json_extract(Metadata, '$.Cc'), '{}') as CcJSON,
			IFNULL(json_extract(Metadata, '$.Bcc'), '{}') as BccJSON
		`).OrderBy("Created DESC")

	for _, w := range args {
		if cleanString(w) == "" {
			continue
		}

		exclude := false
		// search terms starting with a `-` or `!` imply an exclude
		if len(w) > 1 && (strings.HasPrefix(w, "-") || strings.HasPrefix(w, "!")) {
			exclude = true
			w = w[1:]
		}

		re := regexp.MustCompile(`[a-zA-Z0-9]+`)
		if !re.MatchString(w) {
			continue
		}

		if strings.HasPrefix(w, "to:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("ToJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("ToJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "from:") {
			w = cleanString(w[5:])
			if w != "" {
				if exclude {
					q.Where("FromJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("FromJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "cc:") {
			w = cleanString(w[3:])
			if w != "" {
				if exclude {
					q.Where("CcJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("CcJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "bcc:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where("BccJSON NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("BccJSON LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "subject:") {
			w = w[8:]
			if w != "" {
				if exclude {
					q.Where("Subject NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("Subject LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "message-id:") {
			w = cleanString(w[11:])
			if w != "" {
				if exclude {
					q.Where("MessageID NOT LIKE ?", "%"+escPercentChar(w)+"%")
				} else {
					q.Where("MessageID LIKE ?", "%"+escPercentChar(w)+"%")
				}
			}
		} else if strings.HasPrefix(w, "tag:") {
			w = cleanString(w[4:])
			if w != "" {
				if exclude {
					q.Where("Tags NOT LIKE ?", "%\""+escPercentChar(w)+"\"%")
				} else {
					q.Where("Tags LIKE ?", "%\""+escPercentChar(w)+"\"%")
				}
			}
		} else if w == "is:read" {
			if exclude {
				q.Where("Read = 0")
			} else {
				q.Where("Read = 1")
			}
		} else if w == "is:unread" {
			if exclude {
				q.Where("Read = 1")
			} else {
				q.Where("Read = 0")
			}
		} else if w == "is:tagged" {
			if exclude {
				q.Where("Tags = ?", "[]")
			} else {
				q.Where("Tags != ?", "[]")
			}
		} else if w == "has:attachment" || w == "has:attachments" {
			if exclude {
				q.Where("Attachments = 0")
			} else {
				q.Where("Attachments > 0")
			}
		} else {
			// search text
			if exclude {
				q.Where("SearchText NOT LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			} else {
				q.Where("SearchText LIKE ?", "%"+cleanString(escPercentChar(w))+"%")
			}
		}
	}

	return q
}
