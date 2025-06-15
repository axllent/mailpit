-- Add Username column to mailbox for SMTP username tracking
ALTER TABLE {{ tenant "mailbox" }} ADD COLUMN Username TEXT;
