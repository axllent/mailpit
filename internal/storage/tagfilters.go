package storage

import (
	"context"
	"database/sql"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

// TagFilter struct
type TagFilter struct {
	Search string
	SQL    *sqlf.Stmt
	Tags   []string
}

var tagFilters = []TagFilter{}

// LoadTagFilters loads tag filters from the config and pre-generates the SQL query
func LoadTagFilters() {
	tagFilters = []TagFilter{}

	for _, t := range config.SMTPTags {
		tagFilters = append(tagFilters, TagFilter{Search: t.Match, Tags: []string{t.Tag}, SQL: searchQueryBuilder(t.Match, "")})
	}
}

// TagFilterMatches returns a slice of matching tags from a message
func TagFilterMatches(id string) []string {
	tags := []string{}

	if len(tagFilters) == 0 {
		return tags
	}

	for _, f := range tagFilters {
		var matchID string
		q := f.SQL.Clone().Where("ID = ?", id)
		if err := q.QueryAndClose(context.Background(), db, func(row *sql.Rows) {
			var ignore sql.NullString

			if err := row.Scan(&ignore, &matchID, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore, &ignore); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				return
			}
		}); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return tags
		}
		if matchID == id {
			tags = append(tags, f.Tags...)
		}
	}

	return tags
}
