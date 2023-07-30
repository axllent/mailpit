// Package storage handles all database actions
package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/GuiaBolso/darwin"
	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/axllent/mailpit/utils/logger"
	"github.com/jhillyerd/enmime"
	"github.com/klauspost/compress/zstd"
	"github.com/leporo/sqlf"
	"github.com/mattn/go-shellwords"
	uuid "github.com/satori/go.uuid"

	// sqlite (native) - https://gitlab.com/cznic/sqlite
	_ "modernc.org/sqlite"
)

var (
	db            *sql.DB
	dbFile        string
	dbIsTemp      bool
	dbLastAction  time.Time
	dbIsIdle      bool
	dbDataDeleted bool

	// zstd compression encoder & decoder
	dbEncoder, _ = zstd.NewWriter(nil)
	dbDecoder, _ = zstd.NewReader(nil)

	dbMigrations = []darwin.Migration{
		{
			Version:     1.0,
			Description: "Creating tables",
			Script: `CREATE TABLE IF NOT EXISTS mailbox (
				Sort INTEGER PRIMARY KEY AUTOINCREMENT,
				ID TEXT NOT NULL,
				Data BLOB,
				Search TEXT,
				Read INTEGER
			);
			CREATE INDEX IF NOT EXISTS idx_sort ON mailbox (Sort);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_id ON mailbox (ID);
			CREATE INDEX IF NOT EXISTS idx_read ON mailbox (Read);
			
			CREATE TABLE IF NOT EXISTS mailbox_data (
				ID TEXT KEY NOT NULL,
				Email BLOB
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_data_id ON mailbox_data (ID);`,
		},
		{
			Version:     1.1,
			Description: "Create tags column",
			Script: `ALTER TABLE mailbox ADD COLUMN Tags Text  NOT NULL DEFAULT '[]';
			CREATE INDEX IF NOT EXISTS idx_tags ON mailbox (Tags);`,
		},
		{
			Version:     1.2,
			Description: "Creating new mailbox format",
			Script: `CREATE TABLE IF NOT EXISTS mailboxtmp (
				Created INTEGER NOT NULL,
				ID TEXT NOT NULL,
				MessageID TEXT NOT NULL,
				Subject TEXT NOT NULL,
				Metadata TEXT,
				Size INTEGER NOT NULL,
				Inline INTEGER NOT NULL,
				Attachments INTEGER NOT NULL,
				Read INTEGER,
				Tags TEXT,
				SearchText TEXT
			);
			INSERT INTO mailboxtmp 
				(Created, ID, MessageID, Subject, Metadata, Size, Inline, Attachments, SearchText, Read, Tags) 
			SELECT 
				Sort, ID, '', json_extract(Data, '$.Subject'),Data, 
				json_extract(Data, '$.Size'), json_extract(Data, '$.Inline'), json_extract(Data, '$.Attachments'), 
				Search, Read, Tags
			FROM mailbox;

			DROP TABLE IF EXISTS mailbox;
			ALTER TABLE mailboxtmp RENAME TO mailbox;
			CREATE INDEX IF NOT EXISTS idx_created ON mailbox (Created);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_id ON mailbox (ID);
			CREATE INDEX IF NOT EXISTS idx_message_id ON mailbox (MessageID);
			CREATE INDEX IF NOT EXISTS idx_subject ON mailbox (Subject);
			CREATE INDEX IF NOT EXISTS idx_size ON mailbox (Size);
			CREATE INDEX IF NOT EXISTS idx_inline ON mailbox (Inline);
			CREATE INDEX IF NOT EXISTS idx_attachments ON mailbox (Attachments);
			CREATE INDEX IF NOT EXISTS idx_read ON mailbox (Read);
			CREATE INDEX IF NOT EXISTS idx_tags ON mailbox (Tags);`,
		},
	}
)

// DBMailSummary struct for storing mail summary
type DBMailSummary struct {
	From *mail.Address
	To   []*mail.Address
	Cc   []*mail.Address
	Bcc  []*mail.Address
}

// InitDB will initialise the database
func InitDB() error {
	p := config.DataFile

	if p == "" {
		// when no path is provided then we create a temporary file
		// which will get deleted on Close(), SIGINT or SIGTERM
		p = fmt.Sprintf("%s-%d.db", path.Join(os.TempDir(), "mailpit"), time.Now().UnixNano())
		dbIsTemp = true
		logger.Log().Debugf("[db] using temporary database: %s", p)
	} else {
		p = filepath.Clean(p)
	}

	config.DataFile = p

	logger.Log().Debugf("[db] opening database %s", p)

	var err error

	dsn := fmt.Sprintf("file:%s?cache=shared", p)

	db, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	// prevent "database locked" errors
	// @see https://github.com/mattn/go-sqlite3#faq
	db.SetMaxOpenConns(1)

	// create tables if necessary & apply migrations
	if err := dbApplyMigrations(); err != nil {
		return err
	}

	dbFile = p
	dbLastAction = time.Now()

	sigs := make(chan os.Signal, 1)
	// catch all signals since not explicitly listing
	// Program that will listen to the SIGINT and SIGTERM
	// SIGINT will listen to CTRL-C.
	// SIGTERM will be caught if kill command executed
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	// method invoked upon seeing signal
	go func() {
		s := <-sigs
		fmt.Printf("[db] got %s signal, shutting down\n", s)
		Close()
		os.Exit(0)
	}()

	// auto-prune & delete
	go dbCron()

	go dataMigrations()

	return nil
}

// Create tables and apply migrations if required
func dbApplyMigrations() error {
	driver := darwin.NewGenericDriver(db, darwin.SqliteDialect{})

	d := darwin.New(driver, dbMigrations, nil)

	return d.Migrate()
}

// Close will close the database, and delete if a temporary table
func Close() {
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Log().Warning("[db] error closing database, ignoring")
		}
	}

	if dbIsTemp && isFile(dbFile) {
		logger.Log().Debugf("[db] deleting temporary file %s", dbFile)
		if err := os.Remove(dbFile); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
		}
	}
}

// Store will save an email to the database tables
func Store(body []byte) (string, error) {
	// Parse message body with enmime
	env, err := enmime.ReadEnvelope(bytes.NewReader(body))
	if err != nil {
		logger.Log().Warningf("[db] %s", err.Error())
		return "", nil
	}

	from := &mail.Address{}
	fromJSON := addressToSlice(env, "From")
	if len(fromJSON) > 0 {
		from = fromJSON[0]
	} else if env.GetHeader("From") != "" {
		from = &mail.Address{Name: env.GetHeader("From")}
	}

	messageID := strings.Trim(env.Root.Header.Get("Message-ID"), "<>")

	obj := DBMailSummary{
		From: from,
		To:   addressToSlice(env, "To"),
		Cc:   addressToSlice(env, "Cc"),
		Bcc:  addressToSlice(env, "Bcc"),
	}

	created := time.Now()

	// use message date instead of created date
	if config.UseMessageDates {
		if mDate, err := env.Date(); err == nil {
			created = mDate
		}
	}

	// generate the search text
	searchText := createSearchText(env)

	// generate unique ID
	id := uuid.NewV4().String()

	summaryJSON, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	// extract tags from body matches based on --tag
	tagStr := findTagsInRawMessage(&body)

	// extract tags from X-Tags header
	headerTags := strings.TrimSpace(env.Root.Header.Get("X-Tags"))
	if headerTags != "" {
		tagStr += "," + headerTags
	}

	tagData := uniqueTagsFromString(tagStr)

	tagJSON, err := json.Marshal(tagData)
	if err != nil {
		return "", err
	}

	// begin a transaction to ensure both the message
	// and data are stored successfully
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	// roll back if it fails
	defer tx.Rollback()

	subject := env.GetHeader("Subject")
	size := len(body)
	inline := len(env.Inlines)
	attachments := len(env.Attachments)

	// insert mail summary data
	_, err = tx.Exec("INSERT INTO mailbox(Created, ID, MessageID, Subject, Metadata, Size, Inline, Attachments, SearchText, Tags, Read) values(?,?,?,?,?,?,?,?,?,?,0)",
		created.UnixMilli(), id, messageID, subject, string(summaryJSON), size, inline, attachments, searchText, string(tagJSON))
	if err != nil {
		return "", err
	}

	// insert compressed raw message
	compressed := dbEncoder.EncodeAll(body, make([]byte, 0, len(body)))
	_, err = tx.Exec("INSERT INTO mailbox_data(ID, Email) values(?,?)", id, string(compressed))
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	c := &MessageSummary{}
	if err := json.Unmarshal(summaryJSON, c); err != nil {
		return "", err
	}

	c.Created = created
	c.ID = id
	c.MessageID = messageID
	c.Attachments = attachments
	c.Subject = subject
	c.Size = size
	c.Tags = tagData

	websockets.Broadcast("new", c)

	dbLastAction = time.Now()

	return id, nil
}

// List returns a subset of messages from the mailbox,
// sorted latest to oldest
func List(start, limit int) ([]MessageSummary, error) {
	results := []MessageSummary{}

	q := sqlf.From("mailbox").
		Select(`Created, ID, MessageID, Subject, Metadata, Size, Attachments, Read, Tags`).
		OrderBy("Created DESC").
		Limit(limit).
		Offset(start)

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var created int64
		var id string
		var messageID string
		var subject string
		var metadata string
		var size int
		var attachments int
		var tags string
		var read int
		em := MessageSummary{}

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &tags); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(metadata), &em); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(tags), &em.Tags); err != nil {
			logger.Log().Error(err)
			return
		}

		em.Created = time.UnixMilli(created)
		em.ID = id
		em.MessageID = messageID
		em.Subject = subject
		em.Size = size
		em.Attachments = attachments
		em.Read = read == 1

		results = append(results, em)

		// logger.PrettyPrint(em)

	}); err != nil {
		return results, err
	}

	dbLastAction = time.Now()

	return results, nil
}

// Search will search a mailbox for search terms.
// The search is broken up by segments (exact phrases can be quoted), and interprets specific terms such as:
// is:read, is:unread, has:attachment, to:<term>, from:<term> & subject:<term>
// Negative searches also also included by prefixing the search term with a `-` or `!`
func Search(search string, start, limit int) ([]MessageSummary, int, error) {
	results := []MessageSummary{}
	allResults := []MessageSummary{}
	tsStart := time.Now()
	nrResults := 0
	if limit < 0 {
		limit = 50
	}

	s := strings.ToLower(search)
	// add another quote if missing closing quote
	quotes := strings.Count(s, `"`)
	if quotes%2 != 0 {
		s += `"`
	}

	p := shellwords.NewParser()
	args, err := p.Parse(s)
	if err != nil {
		return results, nrResults, errors.New("Your search contains invalid characters")
	}

	// generate the SQL based on arguments
	q := searchParser(args)

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var created int64
		var id string
		var messageID string
		var subject string
		var metadata string
		var size int
		var attachments int
		var tags string
		var read int
		var ignore string
		em := MessageSummary{}

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &tags, &ignore, &ignore, &ignore, &ignore); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(metadata), &em); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(tags), &em.Tags); err != nil {
			logger.Log().Error(err)
			return
		}

		em.Created = time.UnixMilli(created)
		em.ID = id
		em.MessageID = messageID
		em.Subject = subject
		em.Size = size
		em.Attachments = attachments
		em.Read = read == 1

		allResults = append(allResults, em)
	}); err != nil {
		return results, nrResults, err
	}

	dbLastAction = time.Now()

	nrResults = len(allResults)

	if nrResults > start {
		end := nrResults
		if nrResults >= start+limit {
			end = start + limit
		}

		results = allResults[start:end]
	}

	elapsed := time.Since(tsStart)

	logger.Log().Debugf("[db] search for \"%s\" in %s", search, elapsed)

	return results, nrResults, err
}

// GetMessage returns a Message generated from the mailbox_data collection.
// If the message lacks a date header, then the received datetime is used.
func GetMessage(id string) (*Message, error) {
	raw, err := GetMessageRaw(id)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(raw)

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

	messageID := strings.Trim(env.GetHeader("Message-ID"), "<>")

	returnPath := strings.Trim(env.GetHeader("Return-Path"), "<>")
	if returnPath == "" && from != nil {
		returnPath = from.Address
	}

	date, err := env.Date()
	if err != nil {
		// return received datetime when message does not contain a date header
		q := sqlf.From("mailbox").
			Select(`Created`).
			Where(`ID = ?`, id)

		if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
			var created int64

			if err := row.Scan(&created); err != nil {
				logger.Log().Error(err)
				return
			}

			logger.Log().Debugf("[db] %s does not contain a date header, using received datetime", id)

			date = time.UnixMilli(created)
		}); err != nil {
			logger.Log().Error(err)
		}
	}

	obj := Message{
		ID:         id,
		MessageID:  messageID,
		Read:       true,
		From:       from,
		Date:       date,
		To:         addressToSlice(env, "To"),
		Cc:         addressToSlice(env, "Cc"),
		Bcc:        addressToSlice(env, "Bcc"),
		ReplyTo:    addressToSlice(env, "Reply-To"),
		ReturnPath: returnPath,
		Subject:    env.GetHeader("Subject"),
		Tags:       getMessageTags(id),
		Size:       len(raw),
		Text:       env.Text,
	}

	obj.HTML = env.HTML
	obj.Inline = []Attachment{}
	obj.Attachments = []Attachment{}

	for _, i := range env.Inlines {
		if i.FileName != "" || i.ContentID != "" {
			obj.Inline = append(obj.Inline, AttachmentSummary(i))
		}
	}

	for _, i := range env.OtherParts {
		if i.FileName != "" || i.ContentID != "" {
			obj.Inline = append(obj.Inline, AttachmentSummary(i))
		}
	}

	for _, a := range env.Attachments {
		if a.FileName != "" || a.ContentID != "" {
			obj.Attachments = append(obj.Attachments, AttachmentSummary(a))
		}
	}

	// mark message as read
	if err := MarkRead(id); err != nil {
		return &obj, err
	}

	dbLastAction = time.Now()

	return &obj, nil
}

// GetMessageRaw returns an []byte of the full message
func GetMessageRaw(id string) ([]byte, error) {
	var i string
	var msg string
	q := sqlf.From("mailbox_data").
		Select(`ID`).To(&i).
		Select(`Email`).To(&msg).
		Where(`ID = ?`, id)

	err := q.QueryRowAndClose(context.Background(), db)
	if err != nil {
		return nil, err
	}

	if i == "" {
		return nil, errors.New("message not found")
	}

	raw, err := dbDecoder.DecodeAll([]byte(msg), nil)
	if err != nil {
		return nil, fmt.Errorf("error decompressing message: %s", err.Error())
	}

	dbLastAction = time.Now()

	return raw, err
}

// GetAttachmentPart returns an *enmime.Part (attachment or inline) from a message
func GetAttachmentPart(id, partID string) (*enmime.Part, error) {
	raw, err := GetMessageRaw(id)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(raw)

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

	dbLastAction = time.Now()

	return nil, errors.New("attachment not found")
}

// MarkRead will mark a message as read
func MarkRead(id string) error {
	if !IsUnread(id) {
		return nil
	}

	_, err := sqlf.Update("mailbox").
		Set("Read", 1).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	if err == nil {
		logger.Log().Debugf("[db] marked message %s as read", id)
	}

	return err
}

// MarkAllRead will mark all messages as read
func MarkAllRead() error {
	var (
		start = time.Now()
		total = CountUnread()
	)

	_, err := sqlf.Update("mailbox").
		Set("Read", 1).
		Where("Read = ?", 0).
		ExecAndClose(context.Background(), db)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] marked %d messages as read in %s", total, elapsed)

	dbLastAction = time.Now()

	return nil
}

// MarkAllUnread will mark all messages as unread
func MarkAllUnread() error {
	var (
		start = time.Now()
		total = CountRead()
	)

	_, err := sqlf.Update("mailbox").
		Set("Read", 0).
		Where("Read = ?", 1).
		ExecAndClose(context.Background(), db)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] marked %d messages as unread in %s", total, elapsed)

	dbLastAction = time.Now()

	return nil
}

// MarkUnread will mark a message as unread
func MarkUnread(id string) error {
	if IsUnread(id) {
		return nil
	}

	_, err := sqlf.Update("mailbox").
		Set("Read", 0).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	if err == nil {
		logger.Log().Debugf("[db] marked message %s as unread", id)
	}

	dbLastAction = time.Now()

	return err
}

// DeleteOneMessage will delete a single message from a mailbox
func DeleteOneMessage(id string) error {
	// begin a transaction to ensure both the message
	// and data are deleted successfully
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	// roll back if it fails
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM mailbox WHERE ID  = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM mailbox_data WHERE ID  = ?", id)
	if err != nil {
		return err
	}

	err = tx.Commit()

	if err == nil {
		logger.Log().Debugf("[db] deleted message %s", id)
	}

	dbLastAction = time.Now()
	dbDataDeleted = true

	return err
}

// DeleteAllMessages will delete all messages from a mailbox
func DeleteAllMessages() error {
	var (
		start = time.Now()
		total int
	)

	_ = sqlf.From("mailbox").
		Select("COUNT(*)").To(&total).
		QueryRowAndClose(nil, db)

	// begin a transaction to ensure both the message
	// summaries and data are deleted successfully
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	// roll back if it fails
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM mailbox")
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM mailbox_data")
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	_, err = db.Exec("VACUUM")
	if err == nil {
		elapsed := time.Since(start)
		logger.Log().Debugf("[db] deleted %d messages in %s", total, elapsed)
	}

	dbLastAction = time.Now()
	dbDataDeleted = false

	websockets.Broadcast("prune", nil)

	return err
}

// StatsGet returns the total/unread statistics for a mailbox
func StatsGet() MailboxStats {
	var (
		total  = CountTotal()
		unread = CountUnread()
	)

	dbLastAction = time.Now()

	q := sqlf.From("mailbox").
		Select(`DISTINCT Tags`).
		Where("Tags != ?", "[]")

	var tags = []string{}

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var tagData string
		t := []string{}

		if err := row.Scan(&tagData); err != nil {
			logger.Log().Error(err)
			return
		}

		if err := json.Unmarshal([]byte(tagData), &t); err != nil {
			logger.Log().Error(err)
			return
		}

		for _, tag := range t {
			if !inArray(tag, tags) {
				tags = append(tags, tag)
			}
		}

	}); err != nil {
		logger.Log().Error(err)
	}

	sort.Strings(tags)

	return MailboxStats{
		Total:  total,
		Unread: unread,
		Tags:   tags,
	}
}

// CountTotal returns the number of emails in the database
func CountTotal() int {
	var total int

	_ = sqlf.From("mailbox").
		Select("COUNT(*)").To(&total).
		QueryRowAndClose(nil, db)

	return total
}

// CountUnread returns the number of emails in the database that are unread.
func CountUnread() int {
	var total int

	q := sqlf.From("mailbox").
		Select("COUNT(*)").To(&total).
		Where("Read = ?", 0)

	_ = q.QueryRowAndClose(nil, db)

	return total
}

// CountRead returns the number of emails in the database that are read.
func CountRead() int {
	var total int

	q := sqlf.From("mailbox").
		Select("COUNT(*)").To(&total).
		Where("Read = ?", 1)

	_ = q.QueryRowAndClose(nil, db)

	return total
}

// IsUnread returns the number of emails in the database that are unread.
// If an ID is supplied, then it is just limited to that message.
func IsUnread(id string) bool {
	var unread int

	q := sqlf.From("mailbox").
		Select("COUNT(*)").To(&unread).
		Where("Read = ?", 0).
		Where("ID = ?", id)

	_ = q.QueryRowAndClose(nil, db)

	return unread == 1
}

// MessageIDExists checks whether a Message-ID exists in the DB
func MessageIDExists(id string) bool {
	var total int

	q := sqlf.From("mailbox").
		Select("COUNT(*)").To(&total).
		Where("MessageID = ?", id)

	_ = q.QueryRowAndClose(nil, db)

	return total != 0
}
