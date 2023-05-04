package storage

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/jhillyerd/enmime"
	"github.com/leporo/sqlf"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func dataMigrations() {
	updateSortByCreatedTask()
	assignMessageIDsTask()
}

// Update Sort column using Created datetime <= v1.6.5
// Migration task implemented 05/2023 - can be removed end 2023
func updateSortByCreatedTask() {
	q := sqlf.From("mailbox").
		Select("ID").
		Select(`json_extract(Metadata, '$.Created') as Created`).
		Where("Created < ?", 1155000600)

	toUpdate := make(map[string]int64)
	p := message.NewPrinter(language.English)

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var id string
		var ts sql.NullString
		if err := row.Scan(&id, &ts); err != nil {
			logger.Log().Error("[migration]", err)
			return
		}

		if !ts.Valid {
			logger.Log().Errorf("[migration] cannot get Created timestamp from %s", id)
			return
		}

		t, _ := time.Parse(time.RFC3339Nano, ts.String)
		toUpdate[id] = t.UnixMilli()
	}); err != nil {
		logger.Log().Error("[migration]", err)
		return
	}

	total := len(toUpdate)

	if total == 0 {
		return
	}

	logger.Log().Infof("[migration] updating timestamp for %s messages", p.Sprintf("%d", len(toUpdate)))

	// begin a transaction
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Log().Error("[migration]", err)
		return
	}

	// roll back if it fails
	defer tx.Rollback()

	var blockTime = time.Now()

	count := 0
	for id, ts := range toUpdate {
		count++
		_, err := tx.Exec(`UPDATE mailbox SET Created = ? WHERE ID = ?`, ts, id)
		if err != nil {
			logger.Log().Error("[migration]", err)
		}

		if count%1000 == 0 {
			percent := (100 * count) / total
			logger.Log().Infof("[migration] updated timestamp for 1,000 messages [%d%%] in %s", percent, time.Since(blockTime))
			blockTime = time.Now()
		}
	}

	logger.Log().Infof("[migration] commit %s changes", p.Sprintf("%d", count))

	if err := tx.Commit(); err != nil {
		logger.Log().Error("[migration]", err)
		return
	}

	logger.Log().Infof("[migration] complete")
}

// Find any messages without a stored Message-ID and update it <= v1.6.5
// Migration task implemented 05/2023 - can be removed end 2023
func assignMessageIDsTask() {
	if !config.IgnoreDuplicateIDs {
		return
	}

	q := sqlf.From("mailbox").
		Select("ID").
		Where("MessageID = ''")

	missingIDS := make(map[string]string)

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var id string
		if err := row.Scan(&id); err != nil {
			logger.Log().Error("[migration]", err)
			return
		}
		missingIDS[id] = ""
	}); err != nil {
		logger.Log().Error("[migration]", err)
	}

	if len(missingIDS) == 0 {
		return
	}

	var count int
	var blockTime = time.Now()
	p := message.NewPrinter(language.English)

	total := len(missingIDS)

	logger.Log().Infof("[migration] extracting Message-IDs for %s messages", p.Sprintf("%d", total))

	for id := range missingIDS {
		raw, err := GetMessageRaw(id)
		if err != nil {
			logger.Log().Error("[migration]", err)
			continue
		}

		r := bytes.NewReader(raw)

		env, err := enmime.ReadEnvelope(r)
		if err != nil {
			logger.Log().Error("[migration]", err)
			continue
		}

		messageID := strings.Trim(env.GetHeader("Message-ID"), "<>")

		missingIDS[id] = messageID

		count++

		if count%1000 == 0 {
			percent := (100 * count) / total
			logger.Log().Infof("[migration] extracted 1,000 Message-IDs [%d%%] in %s", percent, time.Since(blockTime))
			blockTime = time.Now()
		}
	}

	// begin a transaction
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Log().Error("[migration]", err)
		return
	}

	// roll back if it fails
	defer tx.Rollback()

	count = 0

	for id, mid := range missingIDS {
		_, err = tx.Exec(`UPDATE mailbox SET MessageID = ? WHERE ID = ?`, mid, id)
		if err != nil {
			logger.Log().Error("[migration]", err)
		}

		count++

		if count%1000 == 0 {
			percent := (100 * count) / total
			logger.Log().Infof("[migration] stored 1,000 Message-IDs [%d%%] in %s", percent, time.Since(blockTime))
			blockTime = time.Now()
		}
	}

	logger.Log().Infof("[migration] commit %s changes", p.Sprintf("%d", count))

	if err := tx.Commit(); err != nil {
		logger.Log().Error("[migration]", err)
		return
	}

	logger.Log().Infof("[migration] complete")
}
