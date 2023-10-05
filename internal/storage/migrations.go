package storage

import "github.com/GuiaBolso/darwin"

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
	}
)

// Create tables and apply migrations if required
func dbApplyMigrations() error {
	driver := darwin.NewGenericDriver(db, darwin.SqliteDialect{})

	d := darwin.New(driver, dbMigrations, nil)

	return d.Migrate()
}
