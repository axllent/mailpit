package storage

import (
	"bytes"
	"context"
	"database/sql"
	"os"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/jhillyerd/enmime"
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
		From("mailbox").
		OrderBy("Created DESC").
		QueryAndClose(nil, db, func(row *sql.Rows) {
			ids = append(ids, i)
		})

	if err != nil {
		logger.Log().Error(err)
		os.Exit(1)
	}

	total := len(ids)

	chunks := chunkBy(ids, chunkSize)

	logger.Log().Infof("Reindexing %d messages", total)

	// fmt.Println(len(ids), " = ", len(chunks), "chunks")

	type updateStruct struct {
		ID         string
		SearchText string
		Snippet    string
	}

	for _, ids := range chunks {
		updates := []updateStruct{}

		for _, id := range ids {
			raw, err := GetMessageRaw(id)
			if err != nil {
				logger.Log().Error(err)
				continue
			}

			r := bytes.NewReader(raw)

			env, err := enmime.ReadEnvelope(r)
			if err != nil {
				logger.Log().Error(err)
				continue
			}

			searchText := createSearchText(env)
			snippet := tools.CreateSnippet(env.Text, env.HTML)

			u := updateStruct{}
			u.ID = id
			u.SearchText = searchText
			u.Snippet = snippet

			updates = append(updates, u)
		}

		ctx := context.Background()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			logger.Log().Error(err)
			continue
		}

		// roll back if it fails
		defer tx.Rollback()

		// insert mail summary data
		for _, u := range updates {
			_, err = tx.Exec("UPDATE mailbox SET SearchText = ?, Snippet = ? WHERE ID = ?", u.SearchText, u.Snippet, u.ID)
			if err != nil {
				logger.Log().Error(err)
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			logger.Log().Error(err)
			continue
		}

		finished += len(updates)

		logger.Log().Printf("Reindexed: %d / %d (%d%%)", finished, total, finished*100/total)
	}
}

// Reindex will regenerate the search text and snippet for a message
// and update the database.
func Reindex(id string) error {
	// ids := []string{}
	// var i string
	// // chunkSize := 100

	// err := sqlf.Select("ID").To(&i).From("mailbox_data").QueryAndClose(nil, db, func(row *sql.Rows) {
	// 	ids = append(ids, id)
	// })

	// if err != nil {
	// 	return err
	// }

	// chunks := chunkBy(ids, 100)

	// fmt.Println(len(ids), " = ", len(chunks), "chunks")

	// return nil

	raw, err := GetMessageRaw(id)
	if err != nil {
		return err
	}

	r := bytes.NewReader(raw)

	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return err
	}

	searchText := createSearchText(env)
	snippet := tools.CreateSnippet(env.Text, env.HTML)

	// return nil

	// ctx := context.Background()
	// tx, err := db.BeginTx(ctx, nil)
	// if err != nil {
	// 	return err
	// }

	// // roll back if it fails
	// defer tx.Rollback()

	// // insert mail summary data
	// _, err = tx.Exec("UPDATE mailbox SET SearchText = ?, Snippet = ? WHERE ID = ?", searchText, snippet, id)
	// if err != nil {
	// 	return err
	// }

	// return tx.Commit()

	_, err = sqlf.Update("mailbox").
		Set("SearchText", searchText).
		Set("Snippet", snippet).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	return err
}

// ctx := context.Background()
// tx, err := db.BeginTx(ctx, nil)
// if err != nil {
// 	return "", err
// }

func chunkBy[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}
