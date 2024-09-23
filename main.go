package main

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"os"
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

	mux.HandleFunc("/", withMiddleware(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddlewareSession(authHandler))
	mux.HandleFunc("POST /api/select_playlist", withMiddlewareSession(selectPlaylistHandler))
	// mux.HandleFunc("/api/select_song/{selected}", selectSongHandler)
	// mux.HandleFunc("/select_song", selectSongPageHandler)
	// mux.HandleFunc("/winner", winnerHandler)
	// mux.HandleFunc("/save", saveHandler)

	server := &http.Server{Addr: ":" + viper.GetString("port"), Handler: mux}
	slog.Error("Server exited", "err", server.ListenAndServe())
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	selectPlaylistHtml.Execute(w, nil)
}

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	user, ok := getActiveUser(s)
	if !ok {
		http.Error(w, "user not found", http.StatusInternalServerError)
		slog.Error("user not found")
		return
	}
	slog.Info("loaded user", "userID", user.ID)
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
