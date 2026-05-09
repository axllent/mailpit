package storage

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/leporo/sqlf"
)

var (
	addressPlusRe = regexp.MustCompile(`(?U)^(.*){1,}\+(.*)@`)
)

// SetMessageTags will set the tags for a given database ID, removing any not in the array
func SetMessageTags(id string, tags []string) ([]string, error) {
	// Clean and deduplicate incoming tags (case-insensitive)
	seen := make(map[string]struct{})
	applyTags := []string{}
	for _, t := range tags {
		t = tools.CleanTag(t)
		if t == "" || !config.ValidTagRegexp.MatchString(t) {
			continue
		}
		lc := strings.ToLower(t)
		if _, exists := seen[lc]; exists {
			continue
		}
		seen[lc] = struct{}{}
		applyTags = append(applyTags, t)
	}

	// Fetch existing tags once and index by lowercase name for O(1) lookup
	currentTags := getMessageTags(id)
	currentSet := make(map[string]struct{}, len(currentTags))
	for _, t := range currentTags {
		currentSet[strings.ToLower(t)] = struct{}{}
	}

	// Build apply set for O(1) lookup when computing deletions
	applySet := make(map[string]struct{}, len(applyTags))
	for _, t := range applyTags {
		applySet[strings.ToLower(t)] = struct{}{}
	}

	// Add tags not already on the message
	tagNames := []string{}
	for _, t := range applyTags {
		if _, exists := currentSet[strings.ToLower(t)]; exists {
			continue
		}
		name, err := addMessageTag(id, t)
		if err != nil {
			return []string{}, err
		}
		tagNames = append(tagNames, name)
	}

	// Delete tags removed from the message in a single batch query
	toDelete := []string{}
	for _, t := range currentTags {
		if _, exists := applySet[strings.ToLower(t)]; !exists {
			toDelete = append(toDelete, t)
		}
	}
	if len(toDelete) > 0 {
		if err := deleteMessageTags(id, toDelete); err != nil {
			return []string{}, err
		}
	}

	d := struct {
		ID   string
		Tags []string
	}{ID: id, Tags: applyTags}

	websockets.Broadcast("update", d)

	return tagNames, nil
}

// AddMessageTag adds a tag to a message
func addMessageTag(id, name string) (string, error) {
	// Ensure the tag row exists; the UNIQUE index on Name makes concurrent inserts safe
	if _, err := db.Exec(fmt.Sprintf(`INSERT OR IGNORE INTO %s (Name) VALUES (?)`, tenant("tags")), name); err != nil { // #nosec
		return name, err
	}

	var tagID int
	var foundName string

	if err := sqlf.From(tenant("tags")).
		Select("ID").To(&tagID).
		Select("Name").To(&foundName).
		Where("Name = ?", name).
		QueryRowAndClose(context.TODO(), db); err != nil {
		return name, err
	}

	// Check message does not already have this tag
	var exists int
	if err := sqlf.From(tenant("message_tags")).
		Select("COUNT(ID)").To(&exists).
		Where("ID = ?", id).
		Where("TagID = ?", tagID).
		QueryRowAndClose(context.Background(), db); err != nil {
		return "", err
	}
	if exists > 0 {
		return foundName, nil
	}

	logger.Log().Debugf("[tags] adding tag \"%s\" to %s", name, id)

	_, err := sqlf.InsertInto(tenant("message_tags")).
		Set("ID", id).
		Set("TagID", tagID).
		ExecAndClose(context.TODO(), db)

	return foundName, err
}

// deleteMessageTags deletes multiple tags from a message in a single query
func deleteMessageTags(id string, names []string) error {
	args := make([]any, 1+len(names))
	args[0] = id
	for i, n := range names {
		args[i+1] = n
	}

	query := fmt.Sprintf(
		`DELETE FROM %s WHERE ID = ? AND TagID IN (SELECT ID FROM %s WHERE Name IN (?%s))`,
		tenant("message_tags"), tenant("tags"), strings.Repeat(",?", len(names)-1),
	) // #nosec

	if _, err := db.Exec(query, args...); err != nil {
		return err
	}

	return pruneUnusedTags()
}

// DeleteMessageTag deletes a tag from a message
func deleteMessageTag(id, name string) error {
	if _, err := sqlf.DeleteFrom(tenant("message_tags")).
		Where(tenant("message_tags.ID")+" = ?", id).
		Where(tenant("message_tags.Key")+` IN (SELECT Key FROM `+tenant("message_tags")+` LEFT JOIN `+tenant("tags")+` ON TagID=`+tenant("tags.ID")+` WHERE Name = ?)`, name).
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
		QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
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
	var total float64 // use float64 for rqlite compatibility

	if err := sqlf.
		Select(`Name`).To(&name).
		Select(`COUNT(`+tenant("message_tags.TagID")+`) as total`).To(&total).
		From(tenant("tags")).
		LeftJoin(tenant("message_tags"), tenant("tags.ID")+" = "+tenant("message_tags.TagID")).
		GroupBy(tenant("message_tags.TagID")).
		OrderBy("Name").
		QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
			tags[name] = int64(total)
		}); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return tags
}

// RenameTag renames a tag
func RenameTag(from, to string) error {
	to = tools.CleanTag(to)
	if to == "" || !config.ValidTagRegexp.MatchString(to) {
		return fmt.Errorf("invalid tag name: %s", to)
	}

	if from == to {
		return nil // ignore
	}

	var id, existsID int

	q := sqlf.From(tenant("tags")).
		Select(`ID`).To(&id).
		Where(`Name = ?`, from).
		Limit(1)
	err := q.QueryRowAndClose(context.Background(), db)
	if err != nil {
		return fmt.Errorf("tag not found: %s", from)
	}

	// check if another tag by this name already exists
	q = sqlf.From(tenant("tags")).
		Select("ID").To(&existsID).
		Where(`Name = ?`, to).
		Where(`ID != ?`, id).
		Limit(1)
	err = q.QueryRowAndClose(context.Background(), db)
	if err == nil || existsID != 0 {
		return fmt.Errorf("tag already exists: %s", to)
	}

	q = sqlf.Update(tenant("tags")).
		Set("Name", to).
		Where("ID = ?", id)
	_, err = q.ExecAndClose(context.Background(), db)

	return err
}

// DeleteTag deleted a tag and removed all references to the tag
func DeleteTag(tag string) error {
	var id int

	q := sqlf.From(tenant("tags")).
		Select(`ID`).To(&id).
		Where(`Name = ?`, tag).
		Limit(1)
	err := q.QueryRowAndClose(context.Background(), db)
	if err != nil {
		return fmt.Errorf("tag not found: %s", tag)
	}

	// delete all references
	q = sqlf.DeleteFrom(tenant("message_tags")).
		Where(`TagID = ?`, id)
	_, err = q.ExecAndClose(context.Background(), db)
	if err != nil {
		return fmt.Errorf("error deleting tag references: %s", err.Error())
	}

	// delete tag
	q = sqlf.DeleteFrom(tenant("tags")).
		Where(`ID = ?`, id)
	_, err = q.ExecAndClose(context.Background(), db)
	if err != nil {
		return fmt.Errorf("error deleting tag: %s", err.Error())
	}

	return nil
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

// Find tags set via --tags in raw message, useful for matching all headers etc.
// This function is largely superseded by the database searching, however this
// includes literally everything and is kept for backwards compatibility.
// Returns a comma-separated string.
func findTagsInRawMessage(message *[]byte) []string {
	tags := []string{}
	if len(tagFilters) == 0 {
		return tags
	}

	str := bytes.ToLower(*message)
	for _, t := range tagFilters {
		if bytes.Contains(str, []byte(t.Match)) {
			tags = append(tags, t.Tags...)
		}
	}

	return tags
}

// Returns tags found in email plus addresses (eg: test+tagname@example.com)
func (d Metadata) tagsFromPlusAddresses() []string {
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

	return tools.SetTagCasing(tags)
}

// getTagsForIDs fetches tags for a set of message IDs in a single query,
// returning a map of message ID to tag names.
func getTagsForIDs(ids []string) map[string][]string {
	result := make(map[string][]string, len(ids))
	if len(ids) == 0 {
		return result
	}

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT mt.ID, t.Name FROM %s t JOIN %s mt ON t.ID = mt.TagID WHERE mt.ID IN (?%s) ORDER BY mt.ID, t.Name`,
		tenant("Tags"), tenant("message_tags"), strings.Repeat(",?", len(ids)-1),
	) // #nosec

	rows, err := db.Query(query, args...)
	if err != nil {
		logger.Log().Errorf("[tags] %s", err.Error())
		return result
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			logger.Log().Errorf("[tags] %s", err.Error())
			return result
		}
		result[id] = append(result[id], name)
	}

	return result
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
		QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
			tags = append(tags, name)
		}); err != nil {
		logger.Log().Errorf("[tags] %s", err.Error())
		return tags
	}

	return tags
}

// SortedUniqueTags will return a unique slice of normalised tags
func sortedUniqueTags(s []string) []string {
	tags := []string{}
	added := make(map[string]bool)

	if len(s) == 0 {
		return tags
	}

	for _, p := range s {
		w := tools.CleanTag(p)
		if w == "" {
			continue
		}
		lc := strings.ToLower(w)
		if _, exists := added[lc]; exists {
			continue
		}
		if config.ValidTagRegexp.MatchString(w) {
			added[lc] = true
			tags = append(tags, w)
		} else {
			logger.Log().Debugf("[tags] ignoring invalid tag: %s", w)
		}
	}

	sort.Strings(tags)

	return tags
}
