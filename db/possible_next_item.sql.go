// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: possible_next_item.sql

package db

import (
	"context"
)

const getNextPair = `-- name: GetNextPair :many
SELECT item.id, item.title, item.artists, item.image, item.has_valid_spotify_id 
FROM possible_next_items pn 
INNER JOIN playlist_item item ON pn.playlist_item = item.id
WHERE pn.session = ? AND pn.lost = FALSE AND pn.won_round != ?2
ORDER BY RANDOM() DESC LIMIT 2
`

type GetNextPairParams struct {
	Session      int64
	CurrentRound int64
}

func (q *Queries) GetNextPair(ctx context.Context, arg GetNextPairParams) ([]PlaylistItem, error) {
	rows, err := q.query(ctx, q.getNextPairStmt, getNextPair, arg.Session, arg.CurrentRound)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []PlaylistItem
	for rows.Next() {
		var i PlaylistItem
		if err := rows.Scan(
			&i.ID,
			&i.Title,
			&i.Artists,
			&i.Image,
			&i.HasValidSpotifyID,
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

const initializePossibleNextItemsForSession = `-- name: InitializePossibleNextItemsForSession :exec
INSERT INTO possible_next_items (session, playlist_item, lost, won_round)
SELECT ?, item.id, FALSE, -1 
FROM playlist_item item
INNER JOIN playlist_item_belongs_to_playlist belongs ON item.id = belongs.playlist_item
WHERE belongs.playlist = ?
`

type InitializePossibleNextItemsForSessionParams struct {
	Session  int64
	Playlist string
}

func (q *Queries) InitializePossibleNextItemsForSession(ctx context.Context, arg InitializePossibleNextItemsForSessionParams) error {
	_, err := q.exec(ctx, q.initializePossibleNextItemsForSessionStmt, initializePossibleNextItemsForSession, arg.Session, arg.Playlist)
	return err
}