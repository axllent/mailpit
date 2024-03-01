package storage

import (
	"database/sql"
	"encoding/json"

	"github.com/GuiaBolso/darwin"
	"github.com/axllent/mailpit/internal/logger"
	"github.com/leporo/sqlf"
)

var (
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
		{
			Version:     1.3,
			Description: "Create snippet column",
			Script:      `ALTER TABLE mailbox ADD COLUMN Snippet Text NOT NULL DEFAULT '';`,
		},
		{
			Version:     1.4,
			Description: "Create tag tables",
			Script: `CREATE TABLE IF NOT EXISTS tags (
				ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				Name TEXT COLLATE NOCASE
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_tag_name ON tags (Name);

			CREATE TABLE IF NOT EXISTS message_tags(
				Key INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
				ID TEXT REFERENCES mailbox(ID),
				TagID INT REFERENCES tags(ID)
			);
			CREATE INDEX IF NOT EXISTS idx_message_tag_id ON message_tags (ID);
			CREATE INDEX IF NOT EXISTS idx_message_tag_tagid ON message_tags (TagID);`,
		},
		{
			// assume deleted messages account for 50% of storage
			// to handle previously-deleted messages
			Version:     1.5,
			Description: "Create settings table",
			Script: `CREATE TABLE IF NOT EXISTS settings (
				Key TEXT,
				Value TEXT
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_key ON settings (Key);
			INSERT INTO settings (Key, Value) VALUES("DeletedSize", (SELECT SUM(Size)/2 FROM mailbox));`,
		},
	}
)

// Create tables and apply migrations if required
func dbApplyMigrations() error {
	driver := darwin.NewGenericDriver(db, darwin.SqliteDialect{})

	d := darwin.New(driver, dbMigrations, nil)

	return d.Migrate()
}

// These functions are used to migrate data formats/structure on startup.
func dataMigrations() {
	// ensure DeletedSize has a value if empty
	if SettingGet("DeletedSize") == "" {
		_ = SettingPut("DeletedSize", "0")
	}

	migrateTagsToManyMany()
}

// Migrate tags to ManyMany structure
// Migration task implemented 12/2023
// Can be removed end 06/2024 and Tags column & index dropped from mailbox
func migrateTagsToManyMany() {
	toConvert := make(map[string][]string)
	q := sqlf.
		Select("ID, Tags").
		From("mailbox").
		Where("Tags != ?", "[]").
		Where("Tags IS NOT NULL")

	if err := q.QueryAndClose(nil, db, func(row *sql.Rows) {
		var id string
		var jsonTags string
		if err := row.Scan(&id, &jsonTags); err != nil {
			logger.Log().Errorf("[migration] %s", err.Error())
			return
		}

		tags := []string{}

		if err := json.Unmarshal([]byte(jsonTags), &tags); err != nil {
			logger.Log().Errorf("[json] %s", err.Error())
			return
		}

		toConvert[id] = tags
	}); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}

	if len(toConvert) > 0 {
		logger.Log().Infof("[migration] converting %d message tags", len(toConvert))
		for id, tags := range toConvert {
			if err := SetMessageTags(id, tags); err != nil {
				logger.Log().Errorf("[migration] %s", err.Error())
			} else {
				if _, err := sqlf.Update("mailbox").
					Set("Tags", nil).
					Where("ID = ?", id).
					ExecAndClose(nil, db); err != nil {
					logger.Log().Errorf("[migration] %s", err.Error())
				}
			}
		}

		logger.Log().Info("[migration] tags conversion complete")
	}

	// set all legacy `[]` tags to NULL
	if _, err := sqlf.Update("mailbox").
		Set("Tags", nil).
		Where("Tags = ?", "[]").
		ExecAndClose(nil, db); err != nil {
		logger.Log().Errorf("[migration] %s", err.Error())
	}
}
