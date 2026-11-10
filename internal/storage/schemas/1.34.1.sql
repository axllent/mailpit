-- CREATE Segment COLUMN IN mailbox, existing messages are in the default (empty) segment
ALTER TABLE {{ tenant "mailbox" }} ADD COLUMN Segment TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS {{ tenant "idx_segment_created" }} ON {{ tenant "mailbox" }} (Segment, Created);
