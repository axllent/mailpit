-- Rebuild message_tags to remove FOREIGN KEY REFERENCES
PRAGMA foreign_keys=OFF;

DROP INDEX IF EXISTS {{ tenant "idx_message_tag_id" }};
DROP INDEX IF EXISTS {{ tenant "idx_message_tag_tagid" }};

ALTER TABLE {{ tenant "message_tags" }} RENAME TO _{{ tenant "message_tags" }}_old;

CREATE TABLE IF NOT EXISTS {{ tenant "message_tags" }} (
	Key INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	ID TEXT NOT NULL,
	TagID INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tags_id" }} ON {{ tenant "message_tags" }} (ID);
CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tags_tagid" }} ON {{ tenant "message_tags" }} (TagID);

INSERT INTO {{ tenant "message_tags" }} SELECT * FROM _{{ tenant "message_tags" }}_old;

DROP TABLE IF EXISTS _{{ tenant "message_tags" }}_old;

PRAGMA foreign_keys=ON;
