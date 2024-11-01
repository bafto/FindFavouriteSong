ALTER TABLE match ADD COLUMN creation_timestamp DATETIME;
UPDATE match SET creation_timestamp = CURRENT_TIMESTAMP;
