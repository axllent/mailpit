-- CREATE TABLES
CREATE TABLE IF NOT EXISTS {{ tenant "mailbox" }} (
	Sort INTEGER PRIMARY KEY AUTOINCREMENT,
	ID TEXT NOT NULL,
	Data BLOB,
	Search TEXT,
	Read INTEGER
);

CREATE INDEX IF NOT EXISTS {{ tenant "idx_sort" }} ON {{ tenant "mailbox" }} (Sort);
CREATE UNIQUE INDEX IF NOT EXISTS {{ tenant "idx_id" }} ON {{ tenant "mailbox" }} (ID);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_read" }} ON {{ tenant "mailbox" }} (Read);

CREATE TABLE IF NOT EXISTS {{ tenant "mailbox_data" }} (
	ID TEXT KEY NOT NULL,
	Email BLOB
);

CREATE UNIQUE INDEX IF NOT EXISTS {{ tenant "idx_data_id" }} ON {{ tenant "mailbox_data" }} (ID);
