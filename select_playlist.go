package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify/v2"
)

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger, user, tx, queries, ok := getLoggerUserTransactionQueries(w, r, s)
	if !ok {
		return
	}
	defer tx.Rollback()

	if user.CurrentSession.Valid {
		logAndErr(w, logger, "active session already exists", http.StatusBadRequest)
		return
	}

	// parse playlist url
	playlistUrl := r.FormValue("playlist_url")
	logger.Info("User selected playlist", "playlist-url", playlistUrl)
	playlistId, err := getPlaylistIdFromURL(playlistUrl)
	if err != nil {
		logAndErr(w, logger, "could not parse spotify id from playlist url", http.StatusNotFound, "err", err)
		return
	}
	logger.Info("parsed playlist id", "playlist-id", playlistId)

	// fetch playlist info
	playlist, err := user.client.GetPlaylist(r.Context(), spotify.ID(playlistId))
	if err != nil {
		logAndErr(w, logger, "could not parse spotify id from playlist url", http.StatusNotFound, "err", err)
		return
	}

	// add playlist to DB
	if err := queries.AddOrUpdatePlaylist(r.Context(), db.AddOrUpdatePlaylistParams{
		ID:   playlistId,
		Name: notNull(playlist.Name),
		Url:  notNull(playlistUrl),
	}); err != nil {
		logAndErr(w, logger, "could not insert playlist into db", http.StatusInternalServerError, "err", err)
		return
	}

	// fetch playlist items
	playlistItems, err := getAllPlaylistItems(ctx, user.client, playlist.ID)
	if err != nil {
		logAndErr(w, logger, "could not load songs from playlist", http.StatusInternalServerError, "err", err)
		return
	}

	// add playlist items to DB
	for i := range playlistItems {
		it := &playlistItems[i]
		if err := queries.AddOrUpdatePlaylistItem(r.Context(), db.AddOrUpdatePlaylistItemParams{
			ID:       string(it.Track.Track.ID),
			Title:    notNull(it.Track.Track.Name),
			Artists:  notNull(artistsToString(it.Track.Track.Artists)),
			Image:    notNull(getPlaylistItemImage(it)),
			Playlist: playlistId,
		}); err != nil {
			logAndErr(w, logger, "could not insert playlist item into db", http.StatusInternalServerError, "err", err)
			return
		}
	}

	// create new session
	sessionID, err := queries.AddSession(r.Context(), playlistId)
	if err != nil {
		logAndErr(w, logger, "could not insert session into db", http.StatusInternalServerError, "err", err)
	}
	logger = logger.With("session-id", sessionID)
	logger.Info("created new session")

	// add new session to DB
	if err = queries.SetUserSession(r.Context(), db.SetUserSessionParams{
		CurrentSession: sql.NullInt64{Int64: sessionID, Valid: true},
		ID:             user.ID,
	}); err != nil {
		logAndErr(w, logger, "could not set session for user", http.StatusInternalServerError, "err", err)
	}

	if err := tx.Commit(); err != nil {
		logAndErr(w, logger, "failed to commit DB transaction", http.StatusInternalServerError, "err", err)
		return
	}

	user.CurrentSession = sql.NullInt64{Int64: sessionID, Valid: true}
	logger.Info("added session to user")

	logger.Info("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
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
		items = append(items, page.Items...)
		if err == spotify.ErrNoMorePages {
			return items, nil
		}
		if err != nil {
			return items, err
		}
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
