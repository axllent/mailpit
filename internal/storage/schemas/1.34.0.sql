-- CREATE MetaSearchText COLUMN IN mailbox
-- Case-folded JSON copy of the From, To, Cc, Bcc, ReplyTo, Username & Subject fields, used
-- for case-insensitive field searches (eg: `from:`). Existing messages are populated on
-- startup by backfillMetaSearchText() in schemas.go.
ALTER TABLE {{ tenant "mailbox" }} ADD COLUMN MetaSearchText TEXT;
