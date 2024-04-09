package storage

import (
	"context"
	"database/sql"

	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

// SettingGet returns a setting string value, blank is it does not exist
func SettingGet(k string) string {
	var result sql.NullString
	err := sqlf.From(tenant("settings")).
		Select("Value").To(&result).
		Where("Key = ?", k).
		Limit(1).
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return ""
	}

	return result.String
}

// SettingPut sets a setting string value, inserting if new
func SettingPut(k, v string) error {
	_, err := db.Exec(`INSERT INTO `+tenant("settings")+` (Key, Value) VALUES(?, ?) ON CONFLICT(Key) DO UPDATE SET Value = ?`, k, v, v)
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	return err
}

// The total deleted message size as an int64 value
func getDeletedSize() float64 {
	var result sql.NullFloat64
	err := sqlf.From(tenant("settings")).
		Select("Value").To(&result).
		Where("Key = ?", "DeletedSize").
		Limit(1).
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return 0
	}

	return result.Float64
}

// The total raw non-compressed messages size in bytes of all messages in the database
func totalMessagesSize() float64 {
	var result sql.NullFloat64
	err := sqlf.From(tenant("mailbox")).
		Select("SUM(Size)").To(&result).
		QueryAndClose(context.TODO(), db, func(row *sql.Rows) {})
	if err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
		return 0
	}

	return result.Float64
}

// AddDeletedSize will add the value to the DeletedSize setting
func addDeletedSize(v int64) {
	if _, err := db.Exec(`INSERT OR IGNORE INTO `+tenant("settings")+` (Key, Value) VALUES(?, ?)`, "DeletedSize", 0); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}

	if _, err := db.Exec(`UPDATE `+tenant("settings")+` SET Value = Value + ? WHERE Key = ?`, v, "DeletedSize"); err != nil {
		logger.Log().Errorf("[db] %s", err.Error())
	}
}
