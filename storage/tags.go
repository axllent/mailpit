package storage

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/axllent/mailpit/utils/tools"
	"github.com/leporo/sqlf"
)

// SetTags will set the tags for a given database ID, used via API
func SetTags(id string, tags []string) error {
	applyTags := []string{}
	for _, t := range tags {
		t = tools.CleanTag(t)
		if t != "" && config.ValidTagRegexp.MatchString(t) && !inArray(t, applyTags) {
			applyTags = append(applyTags, t)
		}
	}

	sort.Strings(applyTags)

	tagJSON, err := json.Marshal(applyTags)
	if err != nil {
		logger.Log().Errorf("[db] setting tags for message %s", id)
		return err
	}

	_, err = sqlf.Update("mailbox").
		Set("Tags", string(tagJSON)).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	if err == nil {
		logger.Log().Debugf("[db] set tags %s for message %s", string(tagJSON), id)
	}

	return err
}

// Find tags set via --tags in raw message.
// Returns a comma-separated string.
func findTagsInRawMessage(message *[]byte) string {
	tagStr := ""
	if len(config.SMTPTags) == 0 {
		return tagStr
	}

	str := strings.ToLower(string(*message))
	for _, t := range config.SMTPTags {
		if strings.Contains(str, t.Match) {
			tagStr += "," + t.Tag
		}
	}

	return tagStr
}

// Get message tags from the database for a given database ID
// Used when parsing a raw email.
func getMessageTags(id string) []string {
	tags := []string{}
	var data string

	q := sqlf.From("mailbox").
		Select(`Tags`).To(&data).
		Where(`ID = ?`, id)

	err := q.QueryRowAndClose(context.Background(), db)
	if err != nil {
		logger.Log().Error(err)
		return tags
	}

	if err := json.Unmarshal([]byte(data), &tags); err != nil {
		logger.Log().Error(err)
		return tags
	}

	return tags
}

// UniqueTagsFromString will split a string with commas, and extract a unique slice of formatted tags
func uniqueTagsFromString(s string) []string {
	tags := []string{}

	if s == "" {
		return tags
	}

	parts := strings.Split(s, ",")
	for _, p := range parts {
		w := tools.CleanTag(p)
		if w == "" {
			continue
		}
		if config.ValidTagRegexp.MatchString(w) {
			if !inArray(w, tags) {
				tags = append(tags, w)
			}
		} else {
			logger.Log().Debugf("[db] ignoring invalid tag: %s", w)
		}
	}

	sort.Strings(tags)

	return tags
}
