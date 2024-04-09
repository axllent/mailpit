package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/internal/tools"
	"github.com/axllent/mailpit/server/webhook"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/jhillyerd/enmime"
	"github.com/leporo/sqlf"
	"github.com/lithammer/shortuuid/v4"
)

// Store will save an email to the database tables.
// Returns the database ID of the saved message.
func Store(body *[]byte) (string, error) {
	// Parse message body with enmime
	env, err := enmime.ReadEnvelope(bytes.NewReader(*body))
	if err != nil {
		logger.Log().Warnf("[message] %s", err.Error())
		return "", nil
	}

	from := &mail.Address{}
	fromJSON := addressToSlice(env, "From")
	if len(fromJSON) > 0 {
		from = fromJSON[0]
	} else if env.GetHeader("From") != "" {
		from = &mail.Address{Name: env.GetHeader("From")}
	}

	obj := DBMailSummary{
		From:    from,
		To:      addressToSlice(env, "To"),
		Cc:      addressToSlice(env, "Cc"),
		Bcc:     addressToSlice(env, "Bcc"),
		ReplyTo: addressToSlice(env, "Reply-To"),
	}

	messageID := strings.Trim(env.Root.Header.Get("Message-ID"), "<>")
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
	id := shortuuid.New()

	summaryJSON, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}

	// extract tags from body matches based on --tag, plus addresses & X-Tags header
	tagStr := findTagsInRawMessage(body) + "," +
		obj.tagsFromPlusAddresses() + "," +
		strings.TrimSpace(env.Root.Header.Get("X-Tags"))

	tagData := uniqueTagsFromString(tagStr)

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
	size := float64(len(*body))
	inline := len(env.Inlines)
	attachments := len(env.Attachments)
	snippet := tools.CreateSnippet(env.Text, env.HTML)

	sql := fmt.Sprintf(`INSERT INTO %s 
		(Created, ID, MessageID, Subject, Metadata, Size, Inline, Attachments, SearchText, Read, Snippet) 
		VALUES(?,?,?,?,?,?,?,?,?,0,?)`,
		tenant("mailbox"),
	) // #nosec

	// insert mail summary data
	_, err = tx.Exec(sql, created.UnixMilli(), id, messageID, subject, string(summaryJSON), size, inline, attachments, searchText, snippet)
	if err != nil {
		return "", err
	}

	// insert compressed raw message
	encoded := dbEncoder.EncodeAll(*body, make([]byte, 0, int(size)))
	hexStr := hex.EncodeToString(encoded)
	_, err = tx.Exec(fmt.Sprintf(`INSERT INTO %s (ID, Email) VALUES(?, x'%s')`, tenant("mailbox_data"), hexStr), id) // #nosec
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	if len(tagData) > 0 {
		// set tags after tx.Commit()
		if err := SetMessageTags(id, tagData); err != nil {
			return "", err
		}
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
	c.Snippet = snippet

	websockets.Broadcast("new", c)
	webhook.Send(c)

	dbLastAction = time.Now()

	BroadcastMailboxStats()

	return id, nil
}

// List returns a subset of messages from the mailbox,
// sorted latest to oldest
func List(start, limit int) ([]MessageSummary, error) {
	results := []MessageSummary{}
	tsStart := time.Now()

	q := sqlf.From(tenant("mailbox") + " m").
		Select(`m.Created, m.ID, m.MessageID, m.Subject, m.Metadata, m.Size, m.Attachments, m.Read, m.Snippet`).
		OrderBy("m.Created DESC").
		Limit(limit).
		Offset(start)

	if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
		var created float64
		var id string
		var messageID string
		var subject string
		var metadata string
		var size float64
		var attachments int
		var read int
		var snippet string
		em := MessageSummary{}

		if err := row.Scan(&created, &id, &messageID, &subject, &metadata, &size, &attachments, &read, &snippet); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
			return
		}

		if err := json.Unmarshal([]byte(metadata), &em); err != nil {
			logger.Log().Errorf("[json] %s", err.Error())
			return
		}

		em.Created = time.UnixMilli(int64(created))
		em.ID = id
		em.MessageID = messageID
		em.Subject = subject
		em.Size = size
		em.Attachments = attachments
		em.Read = read == 1
		em.Snippet = snippet
		// artificially generate ReplyTo if legacy data is missing Reply-To field
		if em.ReplyTo == nil {
			em.ReplyTo = []*mail.Address{}
		}

		results = append(results, em)
	}); err != nil {
		return results, err
	}

	// set tags for listed messages only
	for i, m := range results {
		results[i].Tags = getMessageTags(m.ID)
	}

	dbLastAction = time.Now()

	elapsed := time.Since(tsStart)

	logger.Log().Debugf("[db] list INBOX in %s", elapsed)

	return results, nil
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
		q := sqlf.From(tenant("mailbox")).
			Select(`Created`).
			Where(`ID = ?`, id)

		if err := q.QueryAndClose(context.TODO(), db, func(row *sql.Rows) {
			var created float64

			if err := row.Scan(&created); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				return
			}

			logger.Log().Debugf("[db] %s does not contain a date header, using received datetime", id)

			date = time.UnixMilli(int64(created))
		}); err != nil {
			logger.Log().Errorf("[db] %s", err.Error())
		}
	}

	obj := Message{
		ID:         id,
		MessageID:  messageID,
		From:       from,
		Date:       date,
		To:         addressToSlice(env, "To"),
		Cc:         addressToSlice(env, "Cc"),
		Bcc:        addressToSlice(env, "Bcc"),
		ReplyTo:    addressToSlice(env, "Reply-To"),
		ReturnPath: returnPath,
		Subject:    env.GetHeader("Subject"),
		Tags:       getMessageTags(id),
		Size:       float64(len(raw)),
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

	// get List-Unsubscribe links if set
	obj.ListUnsubscribe = ListUnsubscribe{}
	obj.ListUnsubscribe.Links = []string{}
	if env.GetHeader("List-Unsubscribe") != "" {
		l := env.GetHeader("List-Unsubscribe")
		links, err := tools.ListUnsubscribeParser(l)
		obj.ListUnsubscribe.Header = l
		obj.ListUnsubscribe.Links = links
		if err != nil {
			obj.ListUnsubscribe.Errors = err.Error()
		}
		obj.ListUnsubscribe.HeaderPost = env.GetHeader("List-Unsubscribe-Post")
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
	q := sqlf.From(tenant("mailbox_data")).
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

	var data []byte
	if sqlDriver == "rqlite" {
		data, err = base64.StdEncoding.DecodeString(msg)
		if err != nil {
			return nil, fmt.Errorf("error decoding base64 message: %w", err)
		}
	} else {
		data = []byte(msg)
	}

	raw, err := dbDecoder.DecodeAll(data, nil)
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

// LatestID returns the latest message ID
//
// If a query argument is set in the request the function will return the
// latest message matching the search
func LatestID(r *http.Request) (string, error) {
	var messages []MessageSummary
	var err error

	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if search != "" {
		messages, _, err = Search(search, 0, 1)
		if err != nil {
			return "", err
		}
	} else {
		messages, err = List(0, 1)
		if err != nil {
			return "", err
		}
	}
	if len(messages) == 0 {
		return "", errors.New("Message not found")
	}

	return messages[0].ID, nil
}

// MarkRead will mark a message as read
func MarkRead(id string) error {
	if !IsUnread(id) {
		return nil
	}

	_, err := sqlf.Update(tenant("mailbox")).
		Set("Read", 1).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	if err == nil {
		logger.Log().Debugf("[db] marked message %s as read", id)
	}

	BroadcastMailboxStats()

	return err
}

// MarkAllRead will mark all messages as read
func MarkAllRead() error {
	var (
		start = time.Now()
		total = CountUnread()
	)

	_, err := sqlf.Update(tenant("mailbox")).
		Set("Read", 1).
		Where("Read = ?", 0).
		ExecAndClose(context.Background(), db)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] marked %v messages as read in %s", total, elapsed)

	BroadcastMailboxStats()

	dbLastAction = time.Now()

	return nil
}

// MarkAllUnread will mark all messages as unread
func MarkAllUnread() error {
	var (
		start = time.Now()
		total = CountRead()
	)

	_, err := sqlf.Update(tenant("mailbox")).
		Set("Read", 0).
		Where("Read = ?", 1).
		ExecAndClose(context.Background(), db)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] marked %v messages as unread in %s", total, elapsed)

	BroadcastMailboxStats()

	dbLastAction = time.Now()

	return nil
}

// MarkUnread will mark a message as unread
func MarkUnread(id string) error {
	if IsUnread(id) {
		return nil
	}

	_, err := sqlf.Update(tenant("mailbox")).
		Set("Read", 0).
		Where("ID = ?", id).
		ExecAndClose(context.Background(), db)

	if err == nil {
		logger.Log().Debugf("[db] marked message %s as unread", id)
	}

	dbLastAction = time.Now()

	BroadcastMailboxStats()

	return err
}

// DeleteMessages deletes one or more messages in bulk
func DeleteMessages(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()

	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	sql := fmt.Sprintf(`SELECT ID, Size FROM %s WHERE  ID IN (?%s)`, tenant("mailbox"), strings.Repeat(",?", len(args)-1)) // #nosec
	rows, err := db.Query(sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	toDelete := []string{}
	var totalSize float64

	for rows.Next() {
		var id string
		var size float64
		if err := rows.Scan(&id, &size); err != nil {
			return err
		}
		toDelete = append(toDelete, id)
		totalSize = totalSize + size
	}

	if err = rows.Err(); err != nil {
		return err
	}

	if len(toDelete) == 0 {
		return nil // nothing to delete
	}

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	args = make([]interface{}, len(toDelete))
	for i, id := range toDelete {
		args[i] = id
	}

	tables := []string{"mailbox", "mailbox_data", "message_tags"}

	for _, t := range tables {
		sql = fmt.Sprintf(`DELETE FROM %s WHERE ID IN (?%s)`, tenant(t), strings.Repeat(",?", len(ids)-1))

		_, err = tx.Exec(sql, args...) // #nosec
		if err != nil {
			return err
		}
	}

	err = tx.Commit()

	dbLastAction = time.Now()
	addDeletedSize(int64(totalSize))

	logMessagesDeleted(len(toDelete))

	_ = pruneUnusedTags()

	elapsed := time.Since(start)

	messages := "messages"
	if len(toDelete) == 1 {
		messages = "message"
	}

	logger.Log().Debugf("[db] deleted %d %s in %s", len(toDelete), messages, elapsed)

	BroadcastMailboxStats()

	return nil
}

// DeleteAllMessages will delete all messages from a mailbox
func DeleteAllMessages() error {
	var (
		start = time.Now()
		total int
	)

	_ = sqlf.From(tenant("mailbox")).
		Select("COUNT(*)").To(&total).
		QueryRowAndClose(context.TODO(), db)

	// begin a transaction to ensure both the message
	// summaries and data are deleted successfully
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	// roll back if it fails
	defer tx.Rollback()

	tables := []string{"mailbox", "mailbox_data", "tags", "message_tags"}

	for _, t := range tables {
		sql := fmt.Sprintf(`DELETE FROM %s`, tenant(t)) // #nosec
		_, err := tx.Exec(sql)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	elapsed := time.Since(start)
	logger.Log().Debugf("[db] deleted %d messages in %s", total, elapsed)

	vacuumDb()

	dbLastAction = time.Now()
	if err := SettingPut("DeletedSize", "0"); err != nil {
		logger.Log().Warnf("[db] %s", err.Error())
	}

	logMessagesDeleted(total)

	websockets.Broadcast("prune", nil)
	BroadcastMailboxStats()

	return err
}
