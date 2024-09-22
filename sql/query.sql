-- name: GetPlaylist :one
SELECT * FROM playlist
WHERE id = ? LIMIT 1;
