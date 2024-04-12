-- CREATE SETTINGS TABLE
CREATE TABLE IF NOT EXISTS {{ tenant "settings" }} (
	Key TEXT,
	Value TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS {{ tenant "idx_settings_key" }} ON {{ tenant "settings" }} (Key);
INSERT INTO {{ tenant "settings" }} (Key, Value) VALUES ("DeletedSize", (SELECT SUM(Size)/2 FROM {{ tenant "mailbox" }}));
