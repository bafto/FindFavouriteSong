// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: statistics.sql

package db

import (
	"context"
	"database/sql"
)

const getStatistics1 = `-- name: GetStatistics1 :many
WITH winners AS
(SELECT m.winner AS winner FROM
session s
INNER JOIN match m ON m.session = s.id
WHERE s.user = ?
AND s.playlist = ?2 
AND s.winner IS NOT NULL)
SELECT pi.id, pi.title, pi.artists, pi.image, CAST(IFNULL(ct, 0) AS INTEGER) AS points
FROM playlist_item_belongs_to_playlist pibtp
LEFT JOIN
(SELECT winner AS winner, COUNT(*) AS ct FROM winners GROUP BY winner) CountQuery
ON pibtp.playlist_item = CountQuery.winner
INNER JOIN playlist_item pi
ON pi.id = pibtp.playlist_item
WHERE pibtp.playlist = ?2
ORDER BY IFNULL(ct, 0) ASC
`

type GetStatistics1Params struct {
	User     string
	Playlist string
}

type GetStatistics1Row struct {
	ID      string
	Title   sql.NullString
	Artists sql.NullString
	Image   sql.NullString
	Points  int64
}

func (q *Queries) GetStatistics1(ctx context.Context, arg GetStatistics1Params) ([]GetStatistics1Row, error) {
	rows, err := q.query(ctx, q.getStatistics1Stmt, getStatistics1, arg.User, arg.Playlist)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetStatistics1Row
	for rows.Next() {
		var i GetStatistics1Row
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Artists,
			&i.Image,
			&i.Points,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
