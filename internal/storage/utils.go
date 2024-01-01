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
	"github.com/axllent/mailpit/internal/html2text"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/axllent/mailpit/server/websockets"
	"github.com/jhillyerd/enmime"
	"github.com/leporo/sqlf"
)

// Return a header field as a []*mail.Address, or "null" is not found/empty
func addressToSlice(env *enmime.Envelope, key string) []*mail.Address {
	data, err := env.AddressList(key)
	if err != nil || data == nil {
		return []*mail.Address{}
	}

	return data
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
	b.WriteString(env.GetHeader("Reply-To") + " ")
	b.WriteString(env.GetHeader("Return-Path") + " ")

	h := html2text.Strip(env.HTML, true)
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
	// replace \uFEFF with space, see https://github.com/golang/go/issues/42274#issuecomment-1017258184
	str = strings.ReplaceAll(str, string('\uFEFF'), " ")

	// remove/replace new lines
	re := regexp.MustCompile(`(\r?\n|\t|>|<|"|\,|;|\(|\))`)
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

		// check if database contains deleted data and has not been in use
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
				OrderBy("Created DESC").
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

			_, err = tx.Query(`DELETE FROM message_tags WHERE ID IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...) // #nosec
			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				continue
			}

			err = tx.Commit()

			if err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
				if err := tx.Rollback(); err != nil {
					logger.Log().Errorf("[db] %s", err.Error())
				}
			}

			if err := pruneUnusedTags(); err != nil {
				logger.Log().Errorf("[db] %s", err.Error())
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

// InArray tests if a string in within an array. It is not case sensitive.
func inArray(k string, arr []string) bool {
	k = strings.ToLower(k)
	for _, v := range arr {
		if strings.ToLower(v) == k {
			return true
		}
	}

	return false
}

// escPercentChar replaces `%` with `%%` for SQL searches
func escPercentChar(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}

// Escape certain characters in search phrases
func escSearch(str string) string {
	dest := make([]byte, 0, 2*len(str))
	var escape byte
	for i := 0; i < len(str); i++ {
		c := str[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
			break
		case '\n': /* Must be escaped for logs */
			escape = 'n'
			break
		case '\r':
			escape = 'r'
			break
		case '\\':
			escape = '\\'
			break
		case '\'':
			escape = '\''
			break
		case '\032': //十进制26,八进制32,十六进制1a, /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			dest = append(dest, '\\', escape)
		} else {
			dest = append(dest, c)
		}
	}

	return string(dest)
}
