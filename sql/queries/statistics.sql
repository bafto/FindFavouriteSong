-- name: GetStatistics1 :many
WITH winners AS
(SELECT m.winner AS winner FROM
session s
INNER JOIN match m ON m.session = s.id
WHERE s.user = ?
AND s.playlist = sqlc.arg(playlist) 
AND s.winner IS NOT NULL)
SELECT pi.id, pi.title, pi.artists, pi.image, CAST(IFNULL(ct, 0) AS INTEGER) AS points
FROM playlist_item_belongs_to_playlist pibtp
LEFT JOIN
(SELECT winner AS winner, COUNT(*) AS ct FROM winners GROUP BY winner) CountQuery
ON pibtp.playlist_item = CountQuery.winner
INNER JOIN playlist_item pi
ON pi.id = pibtp.playlist_item
WHERE pibtp.playlist = sqlc.arg(playlist)
ORDER BY IFNULL(ct, 0) ASC;
