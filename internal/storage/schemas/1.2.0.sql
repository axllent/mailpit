-- CREATING NEW MAILBOX FORMAT
CREATE TABLE IF NOT EXISTS {{ tenant "mailboxtmp" }} (
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

INSERT INTO {{ tenant "mailboxtmp" }}
	(Created, ID, MessageID, Subject, Metadata, Size, Inline, Attachments, SearchText, Read, Tags) 
	SELECT 
		Sort, ID, '', json_extract(Data, '$.Subject'),Data, 
		json_extract(Data, '$.Size'), json_extract(Data, '$.Inline'), json_extract(Data, '$.Attachments'), 
		Search, Read, Tags
	FROM {{ tenant "mailbox" }};

DROP TABLE IF EXISTS {{ tenant "mailbox" }};

ALTER TABLE {{ tenant "mailboxtmp" }} RENAME TO {{ tenant "mailbox" }};

CREATE INDEX IF NOT EXISTS {{ tenant "idx_created" }} ON {{ tenant "mailbox" }} (Created);
CREATE UNIQUE INDEX IF NOT EXISTS {{ tenant "idx_id" }} ON {{ tenant "mailbox" }} (ID);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_id" }} ON {{ tenant "mailbox" }} (MessageID);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_subject" }} ON {{ tenant "mailbox" }} (Subject);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_size" }} ON {{ tenant "mailbox" }} (Size);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_inline" }} ON {{ tenant "mailbox" }} (Inline);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_attachments" }} ON {{ tenant "mailbox" }} (Attachments);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_read" }} ON {{ tenant "mailbox" }} (Read);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_tags" }} ON {{ tenant "mailbox" }} (Tags);
