// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package db

import (
	"database/sql"
)

type Match struct {
	ID          int64
	Session     int64
	RoundNumber int64
	Winner      string
	Loser       string
}

type Playlist struct {
	ID   string
	Name sql.NullString
	Url  sql.NullString
}

type PlaylistAddedByUser struct {
	User     string
	Playlist string
}

type PlaylistItem struct {
	ID                string
	Title             sql.NullString
	Artists           sql.NullString
	Image             sql.NullString
	HasValidSpotifyID int64
}

type PlaylistItemBelongsToPlaylist struct {
	PlaylistItem string
	Playlist     string
}

type Session struct {
	ID           int64
	Playlist     string
	CurrentRound int64
	User         string
	Winner       sql.NullString
}

type User struct {
	ID             string
	CurrentSession sql.NullInt64
}
