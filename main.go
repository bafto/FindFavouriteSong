package main

import (
	"context"
	"database/sql"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func read_config() {
	viper.SetDefault("spotify_client_id", "")
	viper.SetDefault("spotify_client_secret", "")
	viper.SetDefault("db_path", "ffs.db")
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("redirect_url", "http://localhost:8080/spotifyauthentication")

	viper.SetEnvPrefix("ffs")
	viper.AutomaticEnv()

	viper.AddConfigPath(".")
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

func configure_slog() {
	var level slog.Level
	if err := level.UnmarshalText(
		[]byte(viper.GetString("log_level")),
	); err != nil {
		panic(err)
	}

	addSource := false
	if level <= slog.LevelDebug {
		addSource = true
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(level)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: addSource,
		Level:     levelVar,
	}))

	slog.SetDefault(logger)
}

const (
	session_cookie = "ffs-session-cookie"
	session_id_key = "ffs-user-id"
	state_length   = 16
)

type ActiveUser struct {
	db.User
	client *spotify.Client
}

var (
	selectSongsHtml    = template.Must(template.ParseFiles("select_songs.html"))
	selectPlaylistHtml = template.Must(template.ParseFiles("select_playlist.html"))

	ctx     = context.Background()
	queries *db.Queries

	authKey        = securecookie.GenerateRandomKey(64)
	encryptionKey  = securecookie.GenerateRandomKey(32)
	sessionManager = sessions.NewCookieStore(authKey, encryptionKey)

	spotifyClient *spotify.Client
	spotifyAuth   *spotifyauth.Authenticator
	stateMap      = SyncMap[string, string]{}
	activeUserMap = SyncMap[string, *ActiveUser]{}
)

func main() {
	read_config()
	configure_slog()

	spotifyAuth = spotifyauth.New(
		spotifyauth.WithRedirectURL(viper.GetString("redirect_url")),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopeUserReadPrivate),
		spotifyauth.WithClientSecret(viper.GetString("SPOTIFY_CLIENT_SECRET")),
		spotifyauth.WithClientID(viper.GetString("SPOTIFY_CLIENT_ID")),
	)

	conn, err := create_db(ctx, "ffs.db")
	if err != nil {
		slog.Error("Error opening DB connection", "err", err)
		os.Exit(1)
	}
	defer conn.Close()
	slog.Info("Connected to database")

	queries = db.New(conn)

	mux := http.NewServeMux()

	mux.HandleFunc("/", withMiddlewareSession(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddlewareSession(authHandler))
	mux.HandleFunc("POST /api/select_playlist", withMiddlewareSession(selectPlaylistHandler))
	// mux.HandleFunc("/api/select_song/{selected}", selectSongHandler)
	// mux.HandleFunc("/select_song", selectSongPageHandler)
	// mux.HandleFunc("/winner", winnerHandler)
	// mux.HandleFunc("/save", saveHandler)

	server := &http.Server{Addr: ":" + viper.GetString("port"), Handler: mux}
	slog.Error("Server exited", "err", server.ListenAndServe())
}

func defaultHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger := getLogger(r)

	user, ok := getActiveUser(s)
	if !ok {
		logAndErr(w, logger, "error retrieving user", http.StatusInternalServerError)
		return
	}

	if !user.CurrentSession.Valid {
		selectPlaylistHtml.Execute(w, nil)
		return
	}

	logger.Info("user already has a session")
}

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger := getLogger(r)

	user, ok := getActiveUser(s)
	if !ok {
		logAndErr(w, logger, "no user found", http.StatusInternalServerError)
		return
	}

	if user.CurrentSession.Valid {
		logAndErr(w, logger, "active session already exists", http.StatusBadRequest)
		return
	}

	playlistUrl := r.FormValue("playlist_url")
	playlistId, err := getPlaylistIdFromURL(playlistUrl)
	if err != nil {
		logAndErr(w, logger, "could not parse spotify id from playlist url", http.StatusNotFound, "err", err)
		return
	}

	playlist, err := user.client.GetPlaylist(r.Context(), spotify.ID(playlistId))
	if err != nil {
		logAndErr(w, logger, "could not parse spotify id from playlist url", http.StatusNotFound, "err", err)
		return
	}

	if err := queries.AddOrUpdatePlaylist(r.Context(), db.AddOrUpdatePlaylistParams{
		ID:   playlistId,
		Name: notNull(playlist.Name),
		Url:  notNull(playlistUrl),
	}); err != nil {
		logAndErr(w, logger, "could not insert playlist into db", http.StatusInternalServerError, "err", err)
		return
	}

	playlistItems, err := getAllPlaylistItems(ctx, user.client, playlist.ID)
	if err != nil {
		logAndErr(w, logger, "could not load songs from playlist", http.StatusInternalServerError, "err", err)
		return
	}

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

	sessionId, err := queries.AddSession(r.Context(), playlistId)
	if err != nil {
		logAndErr(w, logger, "could not insert session into db", http.StatusInternalServerError, "err", err)
	}

	if err = queries.SetUserSession(r.Context(), db.SetUserSessionParams{
		CurrentSession: sql.NullInt64{Int64: sessionId, Valid: true},
		ID:             user.ID,
	}); err != nil {
		logAndErr(w, logger, "could not set session for user", http.StatusInternalServerError, "err", err)
	}
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

func getIp(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}

func getActiveUser(session *sessions.Session) (*ActiveUser, bool) {
	userID, ok := session.Values[session_id_key]
	if !ok {
		return nil, false
	}
	return activeUserMap.Load(userID.(string))
}

func notNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
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
}

func logAndErr(w http.ResponseWriter, logger *slog.Logger, msg string, status int, args ...any) {
	logger.Error(msg, args...)
	http.Error(w, msg, status)
}
