-- REPLACE idx_message_tags_id WITH A COMPOSITE (ID, TagID) INDEX
DROP INDEX IF EXISTS {{ tenant "idx_message_tags_id" }};
CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tags_id" }} ON {{ tenant "message_tags" }} (ID, TagID);
