package storage

// These functions are used to migrate data formats/structure on startup.

import (
	"database/sql"
	"encoding/json"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

func dataMigrations() {
	migrateTagsToManyMany()
}

// Migrate tags to ManyMany structure
// Migration task implemented 12/2023
// Can be removed end 06/2024 and Tags column & index dropped from mailbox
func migrateTagsToManyMany() {
	toConvert := make(map[string][]string)
	q := sqlf.
		Select("ID, Tags").
		From("mailbox").
		Where("Tags != ?", "[]").
		Where("Tags IS NOT NULL")

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var id string
		var jsonTags string
		if err := row.Scan(&id, &jsonTags); err != nil {
			logger.Log().Errorf("[migration] %s", err.Error())
			return
		}

		tags := []string{}

		if err := json.Unmarshal([]byte(jsonTags), &tags); err != nil {
			logger.Log().Error(err)
			return
		}

		toConvert[id] = tags
	}); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}

	if len(toConvert) > 0 {
		logger.Log().Infof("[migration] converting %d message tags", len(toConvert))
		for id, tags := range toConvert {
			if err := SetMessageTags(id, tags); err != nil {
				logger.Log().Errorf("[migration] %s", err.Error())
			} else {
				if _, err := sqlf.Update("mailbox").
					Set("Tags", nil).
					Where("ID = ?", id).
					ExecAndClose(nil, db); err != nil {
					logger.Log().Errorf("[migration] %s", err.Error())
				}
			}
		}

		logger.Log().Info("[migration] tags conversion complete")
	}

	// set all legacy `[]` tags to NULL
	if _, err := sqlf.Update("mailbox").
		Set("Tags", nil).
		Where("Tags = ?", "[]").
		ExecAndClose(nil, db); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}
}
