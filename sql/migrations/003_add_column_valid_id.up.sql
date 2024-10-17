ALTER TABLE playlist_item ADD COLUMN has_valid_spotify_id INTEGER NOT NULL DEFAULT TRUE;

UPDATE playlist_item
SET id = SUBSTR(title || artists, 1, 22),
	has_valid_spotify_id = FALSE
WHERE id = '';
