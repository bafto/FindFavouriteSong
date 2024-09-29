package main

import (
	"context"
	"database/sql"
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
	db_conn *sql.DB
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

	var err error
	db_conn, err = create_db(ctx, "ffs.db")
	if err != nil {
		slog.Error("Error opening DB connection", "err", err)
		os.Exit(1)
	}
	defer db_conn.Close()
	slog.Info("Connected to database")

	queries = db.New(db_conn)

	mux := http.NewServeMux()

	mux.HandleFunc("/", withMiddlewareSession(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddlewareSession(authHandler))
	mux.HandleFunc("POST /api/select_playlist", withMiddlewareSession(selectPlaylistHandler))
	mux.HandleFunc("/select_song", withMiddlewareSession(selectSongHandler))
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
		logger.Info("user has no active session")
		selectPlaylistHtml.Execute(w, nil)
		return
	}

	logger.Info("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
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

func getLoggerUserTransactionQueries(w http.ResponseWriter, r *http.Request, s *sessions.Session) (*slog.Logger, *ActiveUser, *sql.Tx, *db.Queries, bool) {
	logger := getLogger(r)

	user, ok := getActiveUser(s)
	if !ok {
		logAndErr(w, logger, "no user found", http.StatusInternalServerError)
		return logger, nil, nil, nil, false
	}
	
	tx, err := db_conn.BeginTx(r.Context(), nil)
	if err != nil {
		logAndErr(w, logger, "failed to create DB transaction", http.StatusInternalServerError, "err", err)
		return logger, user, nil, nil, false
	}

	return logger, user, tx, queries.WithTx(tx), true
}

func notNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}


func logAndErr(w http.ResponseWriter, logger *slog.Logger, msg string, status int, args ...any) {
	logger.Error(msg, args...)
	http.Error(w, msg, status)
}
