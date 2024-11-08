-- DROP LEGACY MIGRATION TABLE
DROP TABLE IF EXISTS {{ tenant "darwin_migrations" }};

-- DROP LEGACY TAGS COLUMN
DROP INDEX IF EXISTS {{ tenant "idx_tags" }};
ALTER TABLE {{ tenant "mailbox" }} DROP COLUMN Tags;
