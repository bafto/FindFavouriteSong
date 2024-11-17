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

-- name: AddPlaylistItemBelongsToPlaylist :exec
INSERT OR IGNORE INTO playlist_item_belongs_to_playlist
(playlist_item, playlist) VALUES (?, ?);

-- name: DeleteItemFromPlaylist :exec
DELETE FROM playlist_item_belongs_to_playlist WHERE playlist = ? AND playlist_item = ?;

-- name: GetItemIdsForPlaylist :many
SELECT playlist_item FROM playlist_item_belongs_to_playlist WHERE playlist = ?;
