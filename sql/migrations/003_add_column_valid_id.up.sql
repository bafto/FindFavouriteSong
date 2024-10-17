ALTER TABLE playlist_item ADD COLUMN has_valid_spotify_id INTEGER;

UPDATE playlist_item
SET id = SUBSTR(title || artists, 1, 22)
SET has_valid_spotify_id = FALSE
WHERE id = '' OR id = NULL;

UPDATE playlist_item
SET has_valid_spotify_id = TRUE
WHERE id != '' AND id != NULL;
