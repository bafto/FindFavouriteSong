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
WHERE id = ?;

-- name: AddSession :one
INSERT INTO session
(id, playlist, current_round) VALUES (NULL, ?, 0)
RETURNING session.id;

-- name: GetCurrentRound :one
SELECT current_round FROM session
WHERE id = ?;

-- name: SetCurrentRound :exec
UPDATE session
SET current_round = ?
WHERE id = ?;

-- name: GetNextPair :many
WITH already_lost_this_session AS (
	SELECT m.loser FROM match m
	WHERE m.session = sqlc.arg(session)
),
already_won_this_round AS (
	SELECT m.winner FROM match m
	WHERE m.session = sqlc.arg(session) AND m.round_number = sqlc.arg(round_number)
)
SELECT * FROM playlist_item
WHERE id NOT IN already_lost_this_session AND id NOT IN already_won_this_round
ORDER BY RANDOM() DESC LIMIT 2;

-- name: AddMatch :exec
INSERT INTO match
(id, session, round_number, winner, loser) VALUES (NULL, ?, ?, ?, ?)
