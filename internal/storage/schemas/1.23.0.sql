-- CREATE Compressed COLUMN IN mailbox_data
ALTER TABLE {{ tenant "mailbox_data" }} ADD COLUMN Compressed INTEGER NOT NULL DEFAULT '0';

-- SET Compressed = 1 for all existing data
UPDATE {{ tenant "mailbox_data" }} SET Compressed = 1;
