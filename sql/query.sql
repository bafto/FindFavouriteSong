-- name: GetPlaylist :one
SELECT * FROM playlist
WHERE id = ? LIMIT 1;

-- name: AddUser :one
INSERT OR IGNORE INTO user (id, current_session) VALUES (?, NULL)
RETURNING *;

-- name: GetUser :one
SELECT * FROM user
WHERE id = ? LIMIT 1;
