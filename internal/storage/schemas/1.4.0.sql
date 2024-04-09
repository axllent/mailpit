-- CREATE TAG TABLES
CREATE TABLE IF NOT EXISTS {{ tenant "tags" }} (
	ID INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	Name TEXT COLLATE NOCASE
);

CREATE UNIQUE INDEX IF NOT EXISTS {{ tenant "idx_tag_name" }} ON {{ tenant "tags" }} (Name);

CREATE TABLE IF NOT EXISTS {{ tenant "message_tags" }} (
	Key INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	ID TEXT REFERENCES {{ tenant "mailbox" }} (ID),
	TagID INT REFERENCES {{ tenant "tags" }} (ID)
);

CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tag_id" }} ON {{ tenant "message_tags" }} (ID);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tag_tagid" }} ON {{ tenant "message_tags" }} (TagID);
