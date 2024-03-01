package storage

import (
	"database/sql"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

// SettingGet returns a setting string value, blank is it does not exist
func SettingGet(k string) string {
	var result sql.NullString
	err := sqlf.From("settings").
		Select("Value").To(&result).
		Where("Key = ?", k).
		Limit(1).
		QueryAndClose(nil, db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return ""
	}

	return result.String
}

// SettingPut sets a setting string value, inserting if new
func SettingPut(k, v string) error {
	_, err := db.Exec("INSERT INTO settings (Key, Value) VALUES(?, ?) ON CONFLICT(Key) DO UPDATE SET Value = ?", k, v, v)
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return err
}

// The total deleted message size as an int64 value
func getDeletedSize() int64 {
	var result sql.NullInt64
	err := sqlf.From("settings").
		Select("Value").To(&result).
		Where("Key = ?", "DeletedSize").
		Limit(1).
		QueryAndClose(nil, db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return 0
	}

	return result.Int64
}

// The total raw non-compressed messages size in bytes of all messages in the database
func totalMessagesSize() int64 {
	var result int64
	err := sqlf.From("mailbox").
		Select("SUM(Size)").To(&result).
		QueryAndClose(nil, db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return 0
	}

	return result
}

// AddDeletedSize will add the value to the DeletedSize setting
func addDeletedSize(v int64) {
	if _, err := db.Exec("INSERT OR IGNORE INTO settings (Key, Value) VALUES(?, ?)", "DeletedSize", 0); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	if _, err := db.Exec("UPDATE settings SET Value = Value + ? WHERE Key = ?", v, "DeletedSize"); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}
}
