// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"database/sql"
)

type Match struct {
	ID      int64
	RoundID int64
	Winner  int64
	Loser   int64
}

type Playlist struct {
	ID        int64
	SpotifyID string
	Name      sql.NullString
	Url       sql.NullString
}

type PlaylistItem struct {
	ID        int64
	SpotifyID string
	Title     sql.NullString
	Artists   sql.NullString
	Image     sql.NullString
	Playlist  int64
}

type Round struct {
	ID        int64
	SessionID int64
	Number    int64
}

type Session struct {
	ID       int64
	Playlist int64
}