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
