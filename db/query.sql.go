// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: query.sql

package db

import (
	"context"
)

const getPlaylist = `-- name: GetPlaylist :one
SELECT id, spotify_id, name, url FROM playlist
WHERE id = ? LIMIT 1
`

func (q *Queries) GetPlaylist(ctx context.Context, id int64) (Playlist, error) {
	row := q.db.QueryRowContext(ctx, getPlaylist, id)
	var i Playlist
	err := row.Scan(
		&i.ID,
		&i.SpotifyID,
		&i.Name,
		&i.Url,
	)
	return i, err
}
