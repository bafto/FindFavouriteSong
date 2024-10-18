package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify/v2"
)

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if user.CurrentSession.Valid {
		return http.StatusBadRequest, fmt.Errorf("active session already exists")
	}

	// parse playlist url
	playlistUrl := r.FormValue("playlist_url")
	logger.Debug("User selected playlist", "playlist-url", playlistUrl)
	playlistId, err := getPlaylistIdFromURL(playlistUrl)
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("could not parse spotify id from playlist url: %w", err)
	}
	logger.Debug("parsed playlist id", "playlist-id", playlistId)

	// fetch playlist info
	playlist, err := user.client.GetPlaylist(r.Context(), spotify.ID(playlistId))
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("could not parse spotify id from playlist url: %w", err)
	}

	// add playlist to DB
	if err := queries.AddOrUpdatePlaylist(r.Context(), db.AddOrUpdatePlaylistParams{
		ID:   playlistId,
		Name: notNull(playlist.Name),
		Url:  notNull(playlistUrl),
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert playlist into db: %w", err)
	}
	if err := queries.AddPlaylistAddedByUser(r.Context(), db.AddPlaylistAddedByUserParams{
		User:     user.ID,
		Playlist: playlistId,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert playlist_added_by_user into db: %w", err)
	}

	// fetch playlist items
	playlistItems, err := getAllPlaylistItems(ctx, user.client, playlist.ID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not load songs from playlist: %w", err)
	}

	// add playlist items to DB
	for i := range playlistItems {
		it := &playlistItems[i]
		has_valid_spotif_id := true
		if it.Track.Track.ID == "" {
			it.Track.Track.ID = spotify.ID(it.Track.Track.Name + artistsToString(it.Track.Track.Artists))[:22]
			has_valid_spotif_id = false
		}

		if err := queries.AddOrUpdatePlaylistItem(r.Context(), db.AddOrUpdatePlaylistItemParams{
			ID:                string(it.Track.Track.ID),
			Title:             notNull(it.Track.Track.Name),
			Artists:           notNull(artistsToString(it.Track.Track.Artists)),
			Image:             notNull(getPlaylistItemImage(it)),
			HasValidSpotifyID: int64(boolToInt(has_valid_spotif_id)),
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not insert playlist item into db: %w", err)
		}

		if err := queries.AddPlaylistItemBelongsToPlaylist(r.Context(), db.AddPlaylistItemBelongsToPlaylistParams{
			PlaylistItem: string(it.Track.Track.ID),
			Playlist:     playlistId,
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not insert playlist_item_belongs_to_playlist into db: %w", err)
		}
	}

	// create new session
	sessionID, err := queries.AddSession(r.Context(), db.AddSessionParams{
		Playlist: playlistId,
		User:     user.ID,
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert session into db: %w", err)
	}
	logger = logger.With("session-id", sessionID)
	logger.Debug("created new session")

	// add new session to DB
	if err = queries.SetUserSession(r.Context(), db.SetUserSessionParams{
		CurrentSession: sql.NullInt64{Int64: sessionID, Valid: true},
		ID:             user.ID,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not set session for user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	user.CurrentSession = sql.NullInt64{Int64: sessionID, Valid: true}
	logger.Info("added session to user")

	logger.Debug("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

func getPlaylistIdFromURL(playlistUrl string) (string, error) {
	parsed, err := url.Parse(playlistUrl)
	if err != nil {
		return "", err
	}
	return path.Base(parsed.Path), nil
}

func getAllPlaylistItems(ctx context.Context, client *spotify.Client, playlistId spotify.ID) ([]spotify.PlaylistItem, error) {
	page, err := client.GetPlaylistItems(ctx, playlistId)
	if err != nil {
		return nil, err
	}
	items := make([]spotify.PlaylistItem, 0, page.Total)
	items = append(items, page.Items...)
	for {
		err = client.NextPage(ctx, page)
		if err == spotify.ErrNoMorePages {
			return items, nil
		}
		if err != nil {
			return items, err
		}
		items = append(items, page.Items...)
	}
}

func artistsToString(artists []spotify.SimpleArtist) string {
	result := strings.Builder{}
	for i, artist := range artists {
		result.WriteString(artist.Name)
		if i != len(artists)-1 {
			result.WriteString(", ")
		}
	}
	return result.String()
}

func getPlaylistItemImage(item *spotify.PlaylistItem) string {
	img := ""
	if len(item.Track.Track.Album.Images) > 0 {
		img = item.Track.Track.Album.Images[0].URL
	}
	if len(item.Track.Track.Album.Images) > 1 {
		img = item.Track.Track.Album.Images[1].URL
	}
	return img
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
