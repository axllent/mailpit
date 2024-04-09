package storage

import (
	"context"
	"database/sql"
	"regexp"
	"sort"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/leporo/sqlf"
)

var (
	addressPlusRe = regexp.MustCompile(`(?U)^(.*){1,}\+(.*)@`)
)

// SetMessageTags will set the tags for a given database ID
func SetMessageTags(id string, tags []string) error {
	applyTags := []string{}
	for _, t := range tags {
		t = tools.CleanTag(t)
		if t != "" && config.ValidTagRegexp.MatchString(t) && !inArray(t, applyTags) {
			applyTags = append(applyTags, t)
		}
	}

	currentTags := getMessageTags(id)
	origTagCount := len(currentTags)

	for _, t := range applyTags {
		t = tools.CleanTag(t)
		if t == "" || !config.ValidTagRegexp.MatchString(t) || inArray(t, currentTags) {
			continue
		}

		if err := AddMessageTag(id, t); err != nil {
			return err
		}
	}

	if origTagCount > 0 {
		currentTags = getMessageTags(id)

		for _, t := range currentTags {
			if !inArray(t, applyTags) {
				if err := DeleteMessageTag(id, t); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// AddMessageTag adds a tag to a message
func AddMessageTag(id, name string) error {
	var tagID int

	q := sqlf.From(tenant("tags")).
		Select("ID").To(&tagID).
		Where("Name = ?", name)

	// tag exists - add tag to message
	if err := q.QueryRowAndClose(context.TODO(), db); err == nil {
		// check message does not already have this tag
		var count int
		if _, err := sqlf.From(tenant("message_tags")).
			Select("COUNT(ID)").To(&count).
			Where("ID = ?", id).
			Where("TagID = ?", tagID).
			ExecAndClose(context.TODO(), db); err != nil {
			return err
		}
		if count != 0 {
			// already exists
			return nil
		}

		logger.Log().Debugf("[tags] adding tag \"%s\" to %s", name, id)

		_, err := sqlf.InsertInto(tenant("message_tags")).
			Set("ID", id).
			Set("TagID", tagID).
			ExecAndClose(context.TODO(), db)
		return err
	}

	logger.Log().Debugf("[tags] adding tag \"%s\" to %s", name, id)

	// tag dos not exist, add new one
	if _, err := sqlf.InsertInto(tenant("tags")).
		Set("Name", name).
		ExecAndClose(context.TODO(), db); err != nil {
		return err
	}

	return AddMessageTag(id, name)
}

// DeleteMessageTag deleted a tag from a message
func DeleteMessageTag(id, name string) error {
	if _, err := sqlf.DeleteFrom(tenant("message_tags")).
		Where(tenant("message_tags.ID")+" = ?", id).
		Where(tenant("message_tags.Key")+` IN (SELECT Key FROM `+tenant("message_tags")+` LEFT JOIN tags ON `+tenant("TagID")+"="+tenant("tags.ID")+` WHERE Name = ?)`, name).
		ExecAndClose(context.TODO(), db); err != nil {
		return err
	}

	return pruneUnusedTags()
}

// DeleteAllMessageTags deleted all tags from a message
func DeleteAllMessageTags(id string) error {
	if _, err := sqlf.DeleteFrom(tenant("message_tags")).
		Where(tenant("message_tags.ID")+" = ?", id).
		ExecAndClose(context.TODO(), db); err != nil {
		return err
	}

	return pruneUnusedTags()
}

// GetAllTags returns all used tags
func GetAllTags() []string {
	var tags = []string{}
	var name string

	if err := sqlf.
		Select(`DISTINCT Name`).
		From(tenant("tags")).To(&name).
		OrderBy("Name").
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			tags = append(tags, name)
		}); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return tags
}

// GetAllTagsCount returns all used tags with their total messages
func GetAllTagsCount() map[string]int64 {
	var tags = make(map[string]int64)
	var name string
	var total int64

	if err := sqlf.
		Select(`Name`).To(&name).
		Select(`COUNT(`+tenant("message_tags.TagID")+`) as total`).To(&total).
		From(tenant("tags")).
		LeftJoin(tenant("message_tags"), tenant("tags.ID")+" = "+tenant("message_tags.TagID")).
		GroupBy(tenant("message_tags.TagID")).
		OrderBy("Name").
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			tags[name] = total
			// tags = append(tags, name)
		}); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return tags
}

// PruneUnusedTags will delete all unused tags from the database
func pruneUnusedTags() error {
	q := sqlf.From(tenant("tags")).
		Select(tenant("tags.ID")+", "+tenant("tags.Name")+", COUNT("+tenant("message_tags.ID")+") as COUNT").
		LeftJoin(tenant("message_tags"), tenant("tags.ID")+" = "+tenant("message_tags.TagID")).
		GroupBy(tenant("tags.ID"))

	toDel := []int{}

	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var n string
		var id int
		var c int

		if err := row.Scan(&id, &n, &c); err != nil {
			logger.Log().Errorf("[tags] %s", err.Error())
			return
		}

		if c == 0 {
			logger.Log().Debugf("[tags] deleting unused tag \"%s\"", n)
			toDel = append(toDel, id)
		}
	}); err != nil {
		return err
	}

	if len(toDel) > 0 {
		for _, id := range toDel {
			if _, err := sqlf.DeleteFrom(tenant("tags")).
				Where("ID = ?", id).
				ExecAndClose(context.TODO(), db); err != nil {
				return err
			}
		}
	}

	return nil
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

// Returns tags found in email plus addresses (eg: test+tagname@example.com)
func (d DBMailSummary) tagsFromPlusAddresses() string {
	tags := []string{}
	for _, c := range d.To {
		matches := addressPlusRe.FindAllStringSubmatch(c.Address, 1)
		if len(matches) == 1 {
			tags = append(tags, strings.Split(matches[0][2], "+")...)
		}
	}
	for _, c := range d.Cc {
		matches := addressPlusRe.FindAllStringSubmatch(c.Address, 1)
		if len(matches) == 1 {
			tags = append(tags, strings.Split(matches[0][2], "+")...)
		}
	}
	for _, c := range d.Bcc {
		matches := addressPlusRe.FindAllStringSubmatch(c.Address, 1)
		if len(matches) == 1 {
			tags = append(tags, strings.Split(matches[0][2], "+")...)
		}
	}
	matches := addressPlusRe.FindAllStringSubmatch(d.From.Address, 1)
	if len(matches) == 1 {
		tags = append(tags, strings.Split(matches[0][2], "+")...)
	}

	return strings.Join(tags, ",")
}

// Get message tags from the database for a given database ID
// Used when parsing a raw email.
func getMessageTags(id string) []string {
	tags := []string{}
	var name string

	if err := sqlf.
		Select(`Name`).To(&name).
		From(tenant("Tags")).
		LeftJoin(tenant("message_tags"), tenant("Tags.ID")+"="+tenant("message_tags.TagID")).
		Where(tenant("message_tags.ID")+` = ?`, id).
		OrderBy("Name").
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			tags = append(tags, name)
		}); err != nil {
		logger.Log().Errorf("[tags] %s", err.Error())
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
			logger.Log().Debugf("[tags] ignoring invalid tag: %s", w)
		}
	}

	sort.Strings(tags)

	return tags
}
