-- name: InitializePossibleNextItemsForSession :exec
INSERT INTO possible_next_items (session, playlist_item, lost, won_round)
SELECT ?, item.id, FALSE, -1 
FROM playlist_item item
INNER JOIN playlist_item_belongs_to_playlist belongs ON item.id = belongs.playlist_item
WHERE belongs.playlist = ?;

-- name: GetNextPair :many
SELECT item.* 
FROM possible_next_items pn 
INNER JOIN playlist_item item ON pn.playlist_item = item.id
WHERE pn.session = ? AND pn.lost = FALSE AND pn.won_round != sqlc.arg(current_round)
ORDER BY RANDOM() DESC LIMIT 2;

-- name: DeletePossibleNextItemsForSession :exec
DELETE FROM possible_next_items WHERE session = ?;
