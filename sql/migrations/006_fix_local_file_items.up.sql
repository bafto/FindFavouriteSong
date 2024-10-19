PRAGMA foreign_keys = OFF;

UPDATE playlist_item 
SET id = REPLACE(id, ' ', '_')
WHERE has_valid_spotify_id = FALSE;

UPDATE match 
SET winner = REPLACE(winner, ' ', '_'),
	loser = REPLACE(loser, ' ', '_');

UPDATE session
SET winner = REPLACE(winner, ' ', '_');

PRAGMA foreign_keys = ON;
