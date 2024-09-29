-- name: GetPlaylist :one
SELECT * FROM playlist
WHERE id = ? LIMIT 1;

-- name: AddOrUpdatePlaylist :exec
INSERT OR REPLACE INTO playlist
(id, name, url) VALUES (?, ?, ?);

-- name: AddOrUpdatePlaylistItem :exec
INSERT OR REPLACE INTO playlist_item
(id, title, artists, image, playlist) VALUES (?, ?, ?, ?, ?);

-- name: AddUser :one
INSERT OR IGNORE INTO user (id, current_session) VALUES (?, NULL)
RETURNING *;

-- name: GetUser :one
SELECT * FROM user
WHERE id = ? LIMIT 1;

-- name: SetUserSession :exec
UPDATE user
SET current_session = ?
WHERE user.id = ?;

-- name: AddSession :one
INSERT INTO session
(id, playlist) VALUES (NULL, ?)
RETURNING session.id;

-- name: GetCurrentRound :one
SELECT COALESCE(max(round_number), 0) FROM match
WHERE session = ?
ORDER BY round_number DESC
LIMIT 1;

-- name: GetNextPair :many
WITH already_lost AS (
	SELECT loser FROM match
	WHERE session = ?
)
SELECT * FROM playlist_item
WHERE id NOT IN already_lost
ORDER BY RANDOM() DESC LIMIT 2;

-- name: AddMatch :exec
INSERT INTO match
(id, session, round_number, winner, loser) VALUES (NULL, ?, ?, ?, ?)
