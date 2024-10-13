package main

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func configure_slog() {
	var level slog.Level
	if err := level.UnmarshalText(
		[]byte(config.Log_level),
	); err != nil {
		panic(err)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(level)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: level <= slog.LevelDebug,
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
	config Config

	selectSongsHtml    = template.Must(template.ParseFiles("select_songs.html"))
	selectPlaylistHtml = template.Must(template.ParseFiles("select_playlist.html"))
	winnerHtml         = template.Must(template.ParseFiles("winner.html"))
	statsHtml          = template.Must(template.ParseFiles("stats.html"))

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
	var err error
	config, err = read_config()
	if err != nil {
		panic(err)
	}
	configure_slog()

	sessionManager.Options.SameSite = http.SameSiteLaxMode

	spotifyAuth = spotifyauth.New(
		spotifyauth.WithRedirectURL(config.Redirect_url),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopeUserReadPrivate),
		spotifyauth.WithClientSecret(config.Spotify_client_secret),
		spotifyauth.WithClientID(config.Spotify_client_id),
	)

	db_conn, err = create_db(ctx, config.Db_path)
	if err != nil {
		slog.Error("Error opening DB connection", "err", err)
		return
	}
	defer db_conn.Close()
	slog.Info("Connected to database")

	queries, err = db.Prepare(ctx, db_conn)
	if err != nil {
		slog.Error("failed to prepare DB queries", "err", err)
		return
	}
	defer queries.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", withMiddleware(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddleware(authHandler))
	mux.HandleFunc("POST /api/select_playlist", withMiddleware(selectPlaylistHandler))
	mux.HandleFunc("/select_song", withMiddleware(selectSongHandler))
	mux.HandleFunc("/winner", withMiddleware(winnerHandler))
	mux.HandleFunc("/stats", withMiddleware(statsPageHandler))

	server := &http.Server{Addr: ":" + config.Port, Handler: mux}

	go func() {
		slog.Info("starting http server")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server errored", "err", err)
			panic(err)
		}
		slog.Warn("HTTP server stopped")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	slog.Warn("received signal, shutting down server", "signal", sig.String())

	shutdownCtx, cancelShutdown := context.WithTimeout(ctx, config.Shutdown_timeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error while shutting down HTTP server, closing it forcefully", "err", err)
		panic(errors.Join(err, server.Close()))
	}
	slog.Info("server shutdown gracefully")
}

func defaultHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger := getLogger(r)

	user, ok := getActiveUser(s)
	if !ok {
		logAndErr(w, logger, "error retrieving user", http.StatusInternalServerError)
		return
	}

	if !user.CurrentSession.Valid {
		logger.Debug("user has no active session, displaying select_playlist.html")
		selectPlaylistHtml.Execute(w, nil)
		return
	}

	logger.Debug("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
}

func winnerHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger, user, tx, queries, ok := getLoggerUserTransactionQueries(w, r, s)
	if !ok {
		return
	}
	defer tx.Rollback()

	winnerID := r.FormValue("winner")
	if winnerID == "" {
		logAndErr(w, logger, "no winner provided", http.StatusBadRequest)
		return
	}

	winnerItem, err := queries.GetPlaylistItem(r.Context(), winnerID)
	if err != nil {
		logAndErr(w, logger, "winner not found in DB", http.StatusBadRequest, "err", err)
		return
	}

	if err := queries.SetUserSession(r.Context(), db.SetUserSessionParams{
		ID:             user.ID,
		CurrentSession: sql.NullInt64{Valid: false},
	}); err != nil {
		logAndErr(w, logger, "unable to reset current session in DB", http.StatusBadRequest, "err", err)
		return
	}

	if err := tx.Commit(); err != nil {
		logAndErr(w, logger, "failed to commit DB transaction", http.StatusInternalServerError, "err", err)
		return
	}
	user.CurrentSession.Valid = false
	logger.Debug("reset user session to NULL")

	winnerHtml.Execute(w, map[string]string{
		"Image":   winnerItem.Image.String,
		"Title":   winnerItem.Title.String,
		"Artists": winnerItem.Artists.String,
	})
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
