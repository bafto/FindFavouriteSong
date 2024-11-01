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

-- name: AddPlaylistAddedByUser :exec
INSERT OR IGNORE INTO playlist_added_by_user
(user, playlist) VALUES (?, ?);

-- name: GetPlaylistsForUser :many
SELECT p.* FROM playlist_added_by_user pa, playlist p
WHERE pa.user = ? AND p.id = pa.playlist;

-- name: GetNonActiveUserSessions :many
SELECT * FROM session
WHERE user = ? AND id != sqlc.arg(activeSession) AND winner IS NULL;
