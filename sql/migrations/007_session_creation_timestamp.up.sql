ALTER TABLE session ADD COLUMN creation_timestamp DATETIME;
UPDATE session SET creation_timestamp = CURRENT_TIMESTAMP;

