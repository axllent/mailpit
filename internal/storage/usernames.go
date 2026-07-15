package storage

import (
	"context"
	"database/sql"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

// GetAllUsernames returns all distinct SMTP/Send-API authentication usernames
// currently in use, sorted alphabetically. Messages received without
// authentication (blank username) are excluded.
func GetAllUsernames() []string {
	var usernames = []string{}
	var name string

	if err := sqlf.
		Select(`DISTINCT json_extract(Metadata, '$.Username')`).To(&name).
		From(tenant("mailbox")).
		Where(`json_extract(Metadata, '$.Username') IS NOT NULL`).
		Where(`json_extract(Metadata, '$.Username') != ''`).
		OrderBy(`json_extract(Metadata, '$.Username')`).
		QueryAndClose(context.TODO(), db, func(_ *sql.Rows) {
			usernames = append(usernames, name)
		}); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return usernames
}
