-- CREATE TAGS COLUMN
ALTER TABLE {{ tenant "mailbox" }} ADD COLUMN Tags Text NOT NULL DEFAULT '[]';
CREATE INDEX IF NOT EXISTS {{ tenant "idx_tags" }} ON {{ tenant "mailbox" }} (Tags);
