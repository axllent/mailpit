package storage

import (
	"context"
	"database/sql"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/leporo/sqlf"
)

// TagFilter struct
type TagFilter struct {
	Match string
	SQL   *sqlf.Stmt
	Tags  []string
}

var tagFilters = []TagFilter{}

// LoadTagFilters loads tag filters from the config and pre-generates the SQL query
func LoadTagFilters() {
	tagFilters = []TagFilter{}

	for _, t := range config.TagFilters {
		match := strings.TrimSpace(t.Match)
		if match == "" {
			logger.Log().Warnf("[tags] ignoring tag item with missing 'match'")
			continue
		}
		if t.Tags == nil || len(t.Tags) == 0 {
			logger.Log().Warnf("[tags] ignoring tag items with missing 'tags' array")
			continue
		}

		validTags := []string{}
		for _, tag := range t.Tags {
			tagName := tools.CleanTag(tag)
			if !config.ValidTagRegexp.MatchString(tagName) || len(tagName) == 0 {
				logger.Log().Warnf("[tags] invalid tag (%s) - can only contain spaces, letters, numbers, - & _", tagName)
				continue
			}
			validTags = append(validTags, tagName)
		}

		if len(validTags) == 0 {
			continue
		}

		tagFilters = append(tagFilters, TagFilter{Match: match, Tags: validTags, SQL: searchQueryBuilder(match, "")})
	}
}

// TagFilterMatches returns a slice of matching tags from a message
func tagFilterMatches(id string) []string {
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
