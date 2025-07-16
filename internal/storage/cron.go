package storage

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/leporo/sqlf"
)

// Database cron runs every minute
func dbCron() {
	for {
		time.Sleep(60 * time.Second)

		currentTime := time.Now()
		sinceLastDbAction := currentTime.Sub(dbLastAction)

		// only run the database has been idle for 5 minutes
		if math.Floor(sinceLastDbAction.Minutes()) == 5 {
			deletedSize := getDeletedSize()

			if deletedSize > 0 {
				total := totalMessagesSize()
				var deletedPercent float64
				if total == 0 {
					deletedPercent = 100
				} else {
					deletedPercent = float64(deletedSize * 100 / total)
				}
				// only vacuum the DB if at least 1% of mail storage size has been deleted
				if deletedPercent >= 1 {
					logger.Log().Debugf("[db] deleted messages is %f%% of total size, reclaim space", deletedPercent)
					vacuumDb()
				}
			}
		}

		pruneMessages()
	}
}

// PruneMessages will auto-delete the oldest messages if messages > config.MaxMessages.
// Set config.MaxMessages to 0 to disable.
func pruneMessages() {
	if config.MaxMessages < 1 && config.MaxAgeInHours == 0 {
		return
	}

	start := time.Now()

	ids := []string{}
	var prunedSize uint64
	var size float64 // use float64 for rqlite compatibility

	// prune using `--max` if set
	if config.MaxMessages > 0 && CountTotal() > uint64(config.MaxMessages) {
		offset := config.MaxMessages
		if config.DemoMode {
			offset = 500
		}
		q := sqlf.Select("ID, Size").
			From(tenant("mailbox")).
			OrderBy("Created DESC").
			Limit(5000).
			Offset(offset)

		if err := q.QueryAndClose(
			context.TODO(), db, func(row *sql.Rows) {
				var id string

				if err := row.Scan(&id, &size); err != nil {
					logger.Log().Errorf("[db] %s", err.Error())
					return
				}
				ids = append(ids, id)
				prunedSize = prunedSize + uint64(size)

			},
		); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}
	}

	// prune using `--max-age` if set
	if config.MaxAgeInHours > 0 {
		// now() minus the number of hours
		ts := time.Now().Add(time.Duration(-config.MaxAgeInHours) * time.Hour).UnixMilli()

		q := sqlf.Select("ID, Size").
			From(tenant("mailbox")).
			Where("Created < ?", ts).
			Limit(5000)

		if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			var id string

			if err := row.Scan(&id, &size); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				return
			}

			if !tools.InArray(id, ids) {
				ids = append(ids, id)
				prunedSize = prunedSize + uint64(size)
			}

		}); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}
	}

	if len(ids) == 0 {
		return
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	_, err = tx.Exec(`DELETE FROM `+tenant("mailbox_data")+` WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	_, err = tx.Exec(`DELETE FROM `+tenant("message_tags")+` WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	_, err = tx.Exec(`DELETE FROM `+tenant("mailbox")+` WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	err = tx.Commit()

	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		if err := tx.Rollback(); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
		}
	}

	if err := pruneUnusedTags(); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	addDeletedSize(prunedSize)
	dbLastAction = time.Now()

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] auto-pruned %d messages in %s", len(ids), elapsed)

	logMessagesDeleted(len(ids))

	if config.DemoMode {
		vacuumDb()
	}

	websockets.Broadcast("prune", nil)
}

// Vacuum the database to reclaim space from deleted messages
func vacuumDb() {
	if sqlDriver == "rqlite" {
		// let rqlite handle vacuuming
		return
	}

	start := time.Now()

	// set WAL file checkpoint
	if _, err := db.Exec("PRAGMA wal_checkpoint"); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	// vacuum database
	if _, err := db.Exec("VACUUM"); err != nil {
		logger.Log().Errorf("[db] VACUUM: %s", err.Error())
		return
	}

	// truncate WAL file
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return
	}

	if err := SettingPut("DeletedSize", "0"); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] vacuum completed in %s", elapsed)
}
