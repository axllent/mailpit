package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/mail"
	"os"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/jhillyerd/enmime/v2"
	"github.com/leporo/sqlf"
)

// ReindexAll will regenerate the search text and snippet for a message
// and update the database.
func ReindexAll() {
	ids := []string{}
	var i string
	chunkSize := 1000

	finished := 0

	err := sqlf.Select("ID").To(&i).
		From(tenant("mailbox")).
		OrderBy("Created DESC").
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			ids = append(ids, i)
		})

	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		os.Exit(1)
	}

	total := len(ids)

	chunks := chunkBy(ids, chunkSize)

	logger.Log().Infof("reindexing %d messages", total)

	type updateStruct struct {
		// ID in database
		ID string
		// SearchText for searching
		SearchText string
		// Snippet for UI
		Snippet string
		// Metadata info
		Metadata string
	}

	parser := enmime.NewParser(enmime.DisableCharacterDetection(true))

	for _, ids := range chunks {
		updates := []updateStruct{}

		for _, id := range ids {
			raw, err := GetMessageRaw(id)
			if err != nil {
				logger.Log().Error(err)
				continue
			}

			r := bytes.NewReader(raw)

			env, err := parser.ReadEnvelope(r)
			if err != nil {
				logger.Log().Errorf("[message] %s", err.Error())
				continue
			}

			meta, _ := GetMetadata(id)

			fromJSON := addressToSlice(env, "From")
			if len(fromJSON) > 0 {
				meta.From = fromJSON[0]
			} else if env.GetHeader("From") != "" {
				meta.From = &mail.Address{Name: env.GetHeader("From")}
			} else {
				meta.From = nil
			}
			meta.To = addressToSlice(env, "To")
			meta.Cc = addressToSlice(env, "Cc")
			meta.Bcc = addressToSlice(env, "Bcc")
			meta.ReplyTo = addressToSlice(env, "Reply-To")

			MetadataJSON, err := json.Marshal(meta)
			if err != nil {
				logger.Log().Errorf("[message] %s", err.Error())
				continue
			}

			searchText := createSearchText(env)
			snippet := tools.CreateSnippet(env.Text, env.HTML)

			u := updateStruct{}
			u.ID = id
			u.SearchText = searchText
			u.Snippet = snippet
			u.Metadata = string(MetadataJSON)

			updates = append(updates, u)
		}

		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			continue
		}

		// roll back if it fails
		defer func() { _ = tx.Rollback() }()

		// insert mail summary data
		for _, u := range updates {
			_, err = tx.Exec(fmt.Sprintf(`UPDATE %s SET SearchText = ?, Snippet = ?, Metadata = ? WHERE ID = ?`, tenant("mailbox")), u.SearchText, u.Snippet, u.Metadata, u.ID)
			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			continue
		}

		finished += len(updates)

		logger.Log().Printf("reindexed: %d / %d (%d%%)", finished, total, finished*100/total)
	}
}

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}
