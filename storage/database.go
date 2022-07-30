package storage

import (
	"bytes"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/data"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/jhillyerd/enmime"
	"github.com/ostafen/clover/v2"
)

var (
	db *clover.DB

	// DefaultMailbox allowing for potential exampnsion in the future
	DefaultMailbox = "catchall"

	count       int
	per100start = time.Now()
)

// CloverStore struct
type CloverStore struct {
	Created     time.Time
	Read        bool
	From        *mail.Address
	To          []*mail.Address
	Cc          []*mail.Address
	Bcc         []*mail.Address
	Subject     string
	Size        int
	Inline      int
	Attachments int
	SearchText  string
}

// InitDB will initialise the database.
// If config.DataDir is empty then it will be in memory.
func InitDB() error {
	var err error
	if config.DataDir != "" {
		logger.Log().Infof("[db] initialising data storage: %s", config.DataDir)
		db, err = clover.Open(config.DataDir)
		if err != nil {
			return err
		}

		sigs := make(chan os.Signal, 1)
		// catch all signals since not explicitly listing
		// Program that will listen to the SIGINT and SIGTERM
		// SIGINT will listen to CTRL-C.
		// SIGTERM will be caught if kill command executed
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		// method invoked upon seeing signal
		go func() {
			s := <-sigs
			logger.Log().Infof("[db] got %s signal, saving persistant data & shutting down", s)
			if err := db.Close(); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
			}

			os.Exit(0)
		}()

	} else {
		logger.Log().Debug("[db] initialising memory data storage")
		db, err = clover.Open("", clover.InMemoryMode(true))
		if err != nil {
			return err
		}
	}

	// auto-prune
	if config.MaxMessages > 0 {
		go pruneCron()
	}

	// create catch-all collection
	return CreateMailbox(DefaultMailbox)
}

// ListMailboxes returns a slice of mailboxes (collections)
func ListMailboxes() ([]data.MailboxSummary, error) {
	mailboxes, err := db.ListCollections()
	if err != nil {
		return nil, err
	}

	results := []data.MailboxSummary{}

	for _, m := range mailboxes {
		// ignore *_data collections
		if strings.HasSuffix(m, "_data") {
			continue
		}

		stats := StatsGet(m)

		mb := data.MailboxSummary{}
		mb.Name = m
		mb.Slug = m
		mb.Total = stats.Total
		mb.Unread = stats.Unread

		if mb.Total > 0 {
			q, err := db.FindFirst(
				clover.NewQuery(m).Sort(clover.SortOption{Field: "Created", Direction: -1}),
			)
			if err != nil {
				return nil, err
			}
			mb.LastMessage = q.Get("Created").(time.Time)
		}

		results = append(results, mb)
	}

	return results, nil
}

// MailboxExists is used to return whether a collection (aka: mailbox) exists
func MailboxExists(name string) bool {
	ok, err := db.HasCollection(name)
	if err != nil {
		return false
	}

	return ok
}

// CreateMailbox will create a collection if it does not exist
func CreateMailbox(name string) error {
	if !MailboxExists(name) {
		logger.Log().Infof("[db] creating mailbox: %s", name)

		if err := db.CreateCollection(name); err != nil {
			return err
		}

		// create Created index
		if err := db.CreateIndex(name, "Created"); err != nil {
			return err
		}

		// create Read index
		if err := db.CreateIndex(name, "Read"); err != nil {
			return err
		}

		// create separate collection for data
		if err := db.CreateCollection(name + "_data"); err != nil {
			return err
		}

		// create Created index
		if err := db.CreateIndex(name+"_data", "Created"); err != nil {
			return err
		}
	}

	return statsRefresh(name)
}

// Store will store a message in the database and return the unique ID
func Store(mailbox string, b []byte) (string, error) {
	r := bytes.NewReader(b)
	// Parse message body with enmime.
	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return "", err
	}

	var from *mail.Address
	fromData := addressToSlice(env, "From")
	if len(fromData) > 0 {
		from = fromData[0]
	} else if env.GetHeader("From") != "" {
		from = &mail.Address{Name: env.GetHeader("From")}
	}

	obj := CloverStore{
		Created:     time.Now(),
		From:        from,
		To:          addressToSlice(env, "To"),
		Cc:          addressToSlice(env, "Cc"),
		Bcc:         addressToSlice(env, "Bcc"),
		Subject:     env.GetHeader("Subject"),
		Size:        len(b),
		Inline:      len(env.Inlines),
		Attachments: len(env.Attachments),
		SearchText:  createSearchText(env),
	}

	doc := clover.NewDocumentOf(obj)

	id, err := db.InsertOne(mailbox, doc)
	if err != nil {
		return "", err
	}

	// save the raw email in a separate collection
	raw := clover.NewDocument()
	raw.Set("_id", id)
	raw.Set("Created", time.Now())
	raw.Set("Data", string(b))
	_, err = db.InsertOne(mailbox+"_data", raw)
	if err != nil {
		// delete the summary because the data insert failed
		logger.Log().Debugf("[db] error inserting raw message, rolling back")
		_ = DeleteOneMessage(mailbox, id)
		return "", err
	}

	statsAddNewMessage(mailbox)

	count++
	if count%100 == 0 {
		logger.Log().Infof("100 messages added in %s", time.Since(per100start))

		per100start = time.Now()
	}

	d, err := db.FindById(DefaultMailbox, id)
	if err != nil {
		return "", err
	}

	c := &data.Summary{}
	if err := d.Unmarshal(c); err != nil {
		return "", err
	}

	c.ID = id

	websockets.Broadcast("new", c)

	return id, nil
}

// List returns a summary of messages.
// For pertformance reasons we manually paginate over queries of 100 results
// as clover's `Skip()` returns a subset of all results which is much slower.
// @see https://github.com/ostafen/clover/issues/73
func List(mailbox string, start, limit int) ([]data.Summary, error) {
	var lastDoc *clover.Document
	count := 0
	startAddingAt := start + 1
	adding := false
	results := []data.Summary{}

	for {
		var instant time.Time
		if lastDoc == nil {
			instant = time.Now()
		} else {
			instant = lastDoc.Get("Created").(time.Time)
		}

		all, err := db.FindAll(
			clover.NewQuery(mailbox).
				Where(clover.Field("Created").Lt(instant)).
				Sort(clover.SortOption{Field: "Created", Direction: -1}).
				Limit(100),
		)
		if err != nil {
			return nil, err
		}

		for _, d := range all {
			count++

			if count == startAddingAt {
				adding = true
			}

			resultsLen := len(results)

			if adding && resultsLen < limit {
				cs := &data.Summary{}
				if err := d.Unmarshal(cs); err != nil {
					return nil, err
				}
				cs.ID = d.ObjectId()
				results = append(results, *cs)
			}
		}

		// we have enough resuts
		if len(results) == limit {
			return results, nil
		}

		if len(all) > 0 {
			lastDoc = all[len(all)-1]
		} else {
			break
		}
	}

	return results, nil
}

// Search returns a summary of items mathing a search. It searched the SearchText field.
func Search(mailbox, search string, start, limit int) ([]data.Summary, error) {
	sq := fmt.Sprintf("(?i)%s", cleanString(regexp.QuoteMeta(search)))
	q, err := db.FindAll(clover.NewQuery(mailbox).
		Skip(start).
		Limit(limit).
		Sort(clover.SortOption{Field: "Created", Direction: -1}).
		Where(clover.Field("SearchText").Like(sq)))
	if err != nil {
		return nil, err
	}

	results := []data.Summary{}

	for _, d := range q {
		cs := &CloverStore{}
		if err := d.Unmarshal(cs); err != nil {
			return nil, err
		}

		results = append(results, cs.Summary(d.ObjectId()))
	}

	return results, nil
}

// Count returns the total number of messages in a mailbox
func Count(mailbox string) (int, error) {
	return db.Count(clover.NewQuery(mailbox))
}

// CountUnread returns the unread number of messages in a mailbox
func CountUnread(mailbox string) (int, error) {
	return db.Count(
		clover.NewQuery(mailbox).
			Where(clover.Field("Read").IsFalse()),
	)
}

// Summary generated a message summary. ID must be supplied
// as this is not stored within the CloverStore but rather the
// *clover.Document
func (c *CloverStore) Summary(id string) data.Summary {
	s := data.Summary{
		ID:          id,
		From:        c.From,
		To:          c.To,
		Cc:          c.Cc,
		Bcc:         c.Bcc,
		Subject:     c.Subject,
		Created:     c.Created,
		Size:        c.Size,
		Attachments: c.Attachments,
	}

	return s
}

// GetMessage returns a data.Message generated from the {mailbox}_data collection.
// ID must be supplied as this is not stored within the CloverStore but rather the
// *clover.Document
func GetMessage(mailbox, id string) (*data.Message, error) {
	q, err := db.FindById(mailbox+"_data", id)
	if err != nil {
		return nil, err
	}

	if q == nil {
		return nil, errors.New("message not found")
	}

	raw := q.Get("Data").(string)

	r := bytes.NewReader([]byte(raw))

	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}

	var from *mail.Address
	fromData := addressToSlice(env, "From")
	if len(fromData) > 0 {
		from = fromData[0]
	} else if env.GetHeader("From") != "" {
		from = &mail.Address{Name: env.GetHeader("From")}
	}

	date, err := env.Date()
	if err != nil {
		// date =
	}

	obj := data.Message{
		ID:      q.ObjectId(),
		Read:    true,
		Created: q.Get("Created").(time.Time),
		From:    from,
		Date:    date,
		To:      addressToSlice(env, "To"),
		Cc:      addressToSlice(env, "Cc"),
		Bcc:     addressToSlice(env, "Bcc"),
		Subject: env.GetHeader("Subject"),
		Size:    len(raw),
		Text:    env.Text,
	}

	html := env.HTML

	// strip base tags
	var re = regexp.MustCompile(`(?U)<base .*>`)
	html = re.ReplaceAllString(html, "")

	for _, i := range env.Inlines {
		if i.FileName != "" || i.ContentID != "" {
			obj.Inline = append(obj.Inline, data.AttachmentSummary(i))
		}
	}

	for _, i := range env.OtherParts {
		if i.FileName != "" || i.ContentID != "" {
			obj.Inline = append(obj.Inline, data.AttachmentSummary(i))
		}
	}

	for _, a := range env.Attachments {
		if a.FileName != "" || a.ContentID != "" {
			obj.Attachments = append(obj.Attachments, data.AttachmentSummary(a))
		}
	}

	obj.HTML = html

	msg, err := db.FindById(mailbox, id)
	if err == nil && !msg.Get("Read").(bool) {
		updates := make(map[string]interface{})
		updates["Read"] = true

		if err := db.UpdateById(mailbox, id, updates); err != nil {
			return nil, err
		}

		statsReadOneMessage(mailbox)
	}

	return &obj, nil
}

// GetAttachmentPart returns an *enmime.Part (attachment or inline) from a message
func GetAttachmentPart(mailbox, id, partID string) (*enmime.Part, error) {
	data, err := GetMessageRaw(mailbox, id)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(data)

	env, err := enmime.ReadEnvelope(r)
	if err != nil {
		return nil, err
	}

	for _, a := range env.Inlines {
		if a.PartID == partID {
			return a, nil
		}
	}

	for _, a := range env.OtherParts {
		if a.PartID == partID {
			return a, nil
		}
	}

	for _, a := range env.Attachments {
		if a.PartID == partID {
			return a, nil
		}
	}

	return nil, errors.New("attachment not found")
}

// GetMessageRaw returns an []byte of the full message
func GetMessageRaw(mailbox, id string) ([]byte, error) {
	q, err := db.FindById(mailbox+"_data", id)
	if err != nil {
		return nil, err
	}

	if q == nil {
		return nil, errors.New("message not found")
	}

	data := q.Get("Data").(string)

	return []byte(data), err
}

// UnreadMessage will delete all messages from a mailbox
func UnreadMessage(mailbox, id string) error {
	updates := make(map[string]interface{})
	updates["Read"] = false

	statsUnreadOneMessage(mailbox)

	return db.UpdateById(mailbox, id, updates)
}

// DeleteOneMessage will delete a single message from a mailbox
func DeleteOneMessage(mailbox, id string) error {
	if err := db.DeleteById(mailbox, id); err != nil {
		return err
	}

	statsDeleteOneMessage(mailbox)

	return db.DeleteById(mailbox+"_data", id)
}

// DeleteAllMessages will delete all messages from a mailbox
func DeleteAllMessages(mailbox string) error {

	totalStart := time.Now()

	totalMessages, err := db.Count(clover.NewQuery(mailbox))
	if err != nil {
		return err
	}

	for {
		toDelete, err := db.Count(clover.NewQuery(mailbox))
		if err != nil {
			return err
		}
		if toDelete == 0 {
			break
		}
		if err := db.Delete(clover.NewQuery(mailbox).Limit(2500)); err != nil {
			return err
		}
		if err := db.Delete(clover.NewQuery(mailbox + "_data").Limit(2500)); err != nil {
			return err
		}
	}

	// resets stats for mailbox
	statsRefresh(mailbox)

	elapsed := time.Since(totalStart)
	logger.Log().Infof("Deleted %d messages from %s in %s", totalMessages, mailbox, elapsed)

	return nil
}
