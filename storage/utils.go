package storage

import (
	"context"
	"database/sql"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/axllent/mailpit/config"
	"github.com/axllent/mailpit/logger"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/jhillyerd/enmime"
	"github.com/k3a/html2text"
	"github.com/leporo/sqlf"
)

// Return a header field as a []*mail.Address, or "null" is not found/empty
func addressToSlice(env *enmime.Envelope, key string) []*mail.Address {
	data, _ := env.AddressList(key)

	return data
}

// Return the headers as a mail.Header
func headersToMap(env *enmime.Envelope) map[string]string {
	headers := make(map[string]string)
	hkeys := env.GetHeaderKeys()
	for _, hkey := range hkeys {
		headers[hkey] = env.GetHeader(hkey)
	}
	return headers
}

// Generate the search text based on some header fields (to, from, subject etc)
// and either the stripped HTML body (if exists) or text body
func createSearchText(env *enmime.Envelope) string {
	var b strings.Builder

	b.WriteString(env.GetHeader("From") + " ")
	b.WriteString(env.GetHeader("Subject") + " ")
	b.WriteString(env.GetHeader("To") + " ")
	b.WriteString(env.GetHeader("Cc") + " ")
	b.WriteString(env.GetHeader("Bcc") + " ")
	h := strings.TrimSpace(html2text.HTML2Text(env.HTML))
	if h != "" {
		b.WriteString(h + " ")
	} else {
		b.WriteString(env.Text + " ")
	}
	// add attachment filenames
	for _, a := range env.Attachments {
		b.WriteString(a.FileName + " ")
	}

	d := cleanString(b.String())

	return d
}

// CleanString removes unwanted characters from stored search text and search queries
func cleanString(str string) string {
	// remove/replace new lines
	re := regexp.MustCompile(`(\r?\n|\t|>|<|"|:|\,|;)`)
	str = re.ReplaceAllString(str, " ")

	// remove duplicate whitespace and trim
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(str)), " "))
}

// Auto-prune runs every minute to automatically delete oldest messages
// if total is greater than the threshold
func dbCron() {
	for {
		time.Sleep(60 * time.Second)
		start := time.Now()

		// check if database contains deleted data and has not beein in use
		// for 5 minutes, if so VACUUM
		currentTime := time.Now()
		diff := currentTime.Sub(dbLastAction)
		if dbDataDeleted && diff.Minutes() > 5 {
			dbDataDeleted = false
			_, err := db.Exec("VACUUM")
			if err == nil {
				elapsed := time.Since(start)
				logger.Log().Debugf("[db] compressed idle database in %s", elapsed)
			}
			continue
		}

		if config.MaxMessages > 0 {
			q := sqlf.Select("ID").
				From("mailbox").
				OrderBy("Sort DESC").
				Limit(5000).
				Offset(config.MaxMessages)

			ids := []string{}
			if err := q.Query(nil, db, func(row *sql.Rows) {
				var id string

				if err := row.Scan(&id); err != nil {
					logger.Log().Errorf("[db] %s", err.Error())
					return
				}
				ids = append(ids, id)

			}); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}

			if len(ids) == 0 {
				continue
			}

			tx, err := db.BeginTx(context.Background(), nil)
			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}

			args := make([]interface{}, len(ids))
			for i, id := range ids {
				args[i] = id
			}

			_, err = tx.Query(`DELETE FROM mailbox WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}

			_, err = tx.Query(`DELETE FROM mailbox_data WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}

			err = tx.Commit()

			if err != nil {
				logger.Log().Errorf(err.Error())
				if err := tx.Rollback(); err != nil {
					logger.Log().Errorf(err.Error())
				}
			}

			dbDataDeleted = true

			elapsed := time.Since(start)
			logger.Log().Debugf("[db] auto-pruned %d messages in %s", len(ids), elapsed)

			websockets.Broadcast("prune", nil)
		}
	}
}

// IsFile returns whether a path is a file
func isFile(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) || !info.Mode().IsRegular() {
		return false
	}

	return true
}

// escPercentChar replaces `%` with `%%` for SQL searches
func escPercentChar(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}
