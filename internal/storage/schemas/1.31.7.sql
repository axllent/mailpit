-- CREATE COMPOSITE INDEX ON message_tags (ID, TagID)
CREATE INDEX IF NOT EXISTS {{ tenant "idx_message_tags_id_tagid" }} ON {{ tenant "message_tags" }} (ID, TagID);
