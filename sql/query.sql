-- name: GetPlaylist :one
SELECT * FROM playlist
WHERE id = ? LIMIT 1;

-- name: AddOrUpdatePlaylist :exec
INSERT OR REPLACE INTO playlist
(id, name, url) VALUES (?, ?, ?);

-- name: GetPlaylistItem :one
SELECT * FROM playlist_item
WHERE id = ?;

-- name: AddOrUpdatePlaylistItem :exec
INSERT OR REPLACE INTO playlist_item
(id, title, artists, image, has_valid_spotify_id) VALUES (?, ?, ?, ?, ?);

-- name: AddUser :one
INSERT OR IGNORE INTO user (id, current_session) VALUES (?, NULL)
RETURNING *;

-- name: GetUser :one
SELECT * FROM user
WHERE id = ? LIMIT 1;

-- name: SetUserSession :exec
UPDATE user
SET current_session = ?
WHERE id = ?;

-- name: GetAllWinnersForUser :many
SELECT winner FROM session
WHERE user = ? AND winner IS NOT NULL;

-- name: AddSession :one
INSERT INTO session
(id, playlist, current_round, user, winner) VALUES (NULL, ?, 0, ?, NULL)
RETURNING session.id;

-- name: GetWinner :one
SELECT winner FROM session
WHERE id = ?;

-- name: SetWinner :exec
UPDATE session
SET winner = ?
WHERE id = ?;

-- name: GetCurrentRound :one
SELECT current_round FROM session
WHERE id = ?;

-- name: SetCurrentRound :exec
UPDATE session
SET current_round = ?
WHERE id = ?;

-- name: GetNextPair :many
WITH already_lost_this_session AS (
	SELECT loser FROM match
	WHERE session = sqlc.arg(session)
),
already_won_this_round AS (
	SELECT m.winner AS winner FROM match m
	WHERE m.session = sqlc.arg(session) AND m.round_number = sqlc.arg(round_number)
)
SELECT item.* FROM 
playlist_item item 
INNER JOIN playlist_item_belongs_to_playlist belongs ON item.id = belongs.playlist_item
INNER JOIN session s ON s.playlist = belongs.playlist
WHERE s.id = sqlc.arg(session) AND
item.id NOT IN already_lost_this_session AND item.id NOT IN already_won_this_round
ORDER BY RANDOM() DESC LIMIT 2;

-- name: AddMatch :exec
INSERT INTO match
(id, session, round_number, winner, loser) VALUES (NULL, ?, ?, ?, ?);

-- name: AddPlaylistAddedByUser :exec
INSERT OR IGNORE INTO playlist_added_by_user
(user, playlist) VALUES (?, ?);

-- name: GetPlaylistsForUser :many
SELECT p.* FROM playlist_added_by_user pa, playlist p
WHERE pa.user = ? AND p.id = pa.playlist;

-- name: AddPlaylistItemBelongsToPlaylist :exec
INSERT OR IGNORE INTO playlist_item_belongs_to_playlist
(playlist_item, playlist) VALUES (?, ?);
