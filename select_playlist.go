package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
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

	logger.Debug("adding playlist to DB")
	if status, err := addPlaylistToDB(r.Context(), logger, user, queries, playlistId, playlistUrl); err != nil {
		return status, err
	}

	logger.Debug("preparing new session")
	if status, err := prepareNewSession(r.Context(), logger, user, queries, tx, playlistId); err != nil {
		return status, err
	}

	logger.Debug("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

// helper function for selectPlaylistHandler
func prepareNewSession(ctx context.Context, logger *slog.Logger, user *ActiveUser, queries *db.Queries, tx *sql.Tx, playlistId string) (int, error) {
	// create new session
	sessionID, err := queries.AddSession(ctx, db.AddSessionParams{
		Playlist: playlistId,
		User:     user.ID,
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert session into db: %w", err)
	}
	logger = logger.With("session-id", sessionID)
	logger.Debug("created new session")

	if err := queries.InitializePossibleNextItemsForSession(ctx, db.InitializePossibleNextItemsForSessionParams{
		Session:  sessionID,
		Playlist: playlistId,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not initialize possible_next_items: %w", err)
	}
	logger.Debug("initialized possible_next_items")

	// add new session to DB
	if err = queries.SetUserSession(ctx, db.SetUserSessionParams{
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

	return -1, nil
}

// helper function for selectPlaylistHandler
func addPlaylistToDB(ctx context.Context, logger *slog.Logger, user *ActiveUser, queries *db.Queries, playlistId, playlistUrl string) (int, error) {
	// fetch playlist info
	playlist, err := user.client.GetPlaylist(ctx, spotify.ID(playlistId))
	if err != nil {
		return http.StatusNotFound, fmt.Errorf("could not parse spotify id from playlist url: %w", err)
	}
	logger = logger.With("playlist-id", playlist.ID)
	logger.Debug("fetched playlist")

	// add playlist to DB
	if err := queries.AddOrUpdatePlaylist(ctx, db.AddOrUpdatePlaylistParams{
		ID:   playlistId,
		Name: notNull(playlist.Name),
		Url:  notNull(playlistUrl),
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert playlist into db: %w", err)
	}
	logger.Debug("added playlist to db")
	if err := queries.AddPlaylistAddedByUser(ctx, db.AddPlaylistAddedByUserParams{
		User:     user.ID,
		Playlist: playlistId,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not insert playlist_added_by_user into db: %w", err)
	}
	logger.Debug("added playlist to user")

	// fetch playlist items
	playlistItems, err := getAllPlaylistItems(ctx, user.client, playlist.ID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not load songs from playlist: %w", err)
	}
	logger.Debug("fetched playlist items", "n-items", len(playlistItems))

	playlistItemsSet := map[spotify.ID]struct{}{}
	// add playlist items to DB
	for i := range playlistItems {
		it := &playlistItems[i]
		has_valid_spotif_id := true
		if it.Track.Track.ID == "" {
			it.Track.Track.ID = spotify.ID(strings.ReplaceAll(it.Track.Track.Name+artistsToString(it.Track.Track.Artists), " ", "_"))[:22]
			has_valid_spotif_id = false
		}

		if err := queries.AddOrUpdatePlaylistItem(ctx, db.AddOrUpdatePlaylistItemParams{
			ID:                string(it.Track.Track.ID),
			Title:             notNull(it.Track.Track.Name),
			Artists:           notNull(artistsToString(it.Track.Track.Artists)),
			Image:             notNull(getPlaylistItemImage(it)),
			HasValidSpotifyID: int64(boolToInt(has_valid_spotif_id)),
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not insert playlist item into db: %w", err)
		}

		if err := queries.AddPlaylistItemBelongsToPlaylist(ctx, db.AddPlaylistItemBelongsToPlaylistParams{
			PlaylistItem: string(it.Track.Track.ID),
			Playlist:     playlistId,
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not insert playlist_item_belongs_to_playlist into db: %w", err)
		}

		playlistItemsSet[it.Track.Track.ID] = struct{}{}
	}
	logger.Debug("added playlist items to db")

	playlistItemIds, err := queries.GetItemIdsForPlaylist(ctx, playlistId)
	if err != nil {
		logger.Warn("Could not retrieve items for playlist from db: not deleting any items", "err", err, "playlist-id", playlistId)
		return -1, nil
	}

	for _, item := range playlistItemIds {
		if _, ok := playlistItemsSet[spotify.ID(item)]; ok {
			continue
		}

		if err := queries.DeleteItemFromPlaylist(ctx, db.DeleteItemFromPlaylistParams{
			Playlist:     playlistId,
			PlaylistItem: item,
		}); err != nil {
			logger.Warn("Error deleting item from playlist_item_belongs_to_playlist", "err", err, "playlist-item-id", item, "playlist-id", playlistId)
		}
	}

	return -1, nil
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
