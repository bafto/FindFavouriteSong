package main

import (
	"database/sql"
	"fmt"
	"net/url"
)

type Persister interface {
	Persist(*sql.DB) error
}

type Playlist struct {
	id         int
	spotify_id string
	name       string
	url        url.URL
}

type PlaylistItem struct {
	id         int
	spotify_id string
	artists    []string
	image      url.URL
	playlist   *Playlist
}

type Session struct {
	id       int
	playlist *Playlist
}

type Round struct {
	id      int
	session *Session
	number  int
}

type Match struct {
	id     int
	round  *Round
	winner *PlaylistItem
	loser  *PlaylistItem
}

var playlist_persist_stmt *sql.Stmt

func prepare_stmts(db *sql.DB) (err error) {
	playlist_persist_stmt, err = db.Prepare(`
	INSERT INTO playlist (id, name, url) VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare playlist_persist_stmt: %w", err)
	}
	return nil
}

func (p *Playlist) Persist(db *sql.DB) error {
	playlist_persist_stmt.
}
