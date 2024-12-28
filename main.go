package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
		Level: levelVar,
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

func (user *ActiveUser) CurrentSessionNotNull() int64 {
	if user.CurrentSession.Valid {
		return user.CurrentSession.Int64
	}
	return -1
}

// to be run concurrently
func checkpoint_ticker(ctx context.Context, db *sql.DB) {
	ticker := time.Tick(config.CheckpointInterval)
	for range ticker {
		slog.Info("starting db checkpoint")
		func() {
			ctx, cancel := context.WithTimeout(ctx, config.CheckpointTimeout)
			defer cancel()

			if err := checkpoint_db(ctx, db); err != nil {
				slog.Error("failed to checkpoint db", "err", err)
				return
			}
			slog.Info("done checkpointing db")
		}()
	}
}

var (
	config Config

	selectSongsHtml    = template.Must(template.ParseFiles("select_songs.gohtml"))
	selectPlaylistHtml = template.Must(template.ParseFiles("select_playlist.gohtml"))
	winnerHtml         = template.Must(template.ParseFiles("winner.gohtml"))
	statsHtml          = template.Must(template.ParseFiles("stats.gohtml"))
	errorHtml          = template.Must(template.ParseFiles("error.gohtml"))

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

	db_conn, err = create_db(ctx, config.Datasource)
	if err != nil {
		slog.Error("Error opening DB connection", "err", err)
		return
	}
	defer db_conn.Close()
	slog.Info("Connected to database, migrating schema")
	if err := migrate_db(ctx, db_conn); err != nil {
		slog.Error("Error migrating db schema", "err", err)
		return
	}

	queries, err = db.Prepare(ctx, db_conn)
	if err != nil {
		slog.Error("failed to prepare DB queries", "err", err)
		return
	}
	defer queries.Close()

	go checkpoint_ticker(ctx, db_conn)

	mux := http.NewServeMux()

	mux.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))

	mux.HandleFunc("/", withMiddleware(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddleware(authHandler))
	mux.HandleFunc("POST /api/select_playlist", withMiddleware(selectPlaylistHandler))
	mux.HandleFunc("POST /api/select_session", withMiddleware(selectSessionHandler))
	mux.HandleFunc("/api/select_new_playlist", withMiddleware(selectNewPlaylistHandler))
	mux.HandleFunc("/select_song", withMiddleware(selectSongPageHandler))
	mux.HandleFunc("POST /api/select_song", withMiddleware(selectSongHandler))
	mux.HandleFunc("/winner", withMiddleware(winnerHandler))
	mux.HandleFunc("/stats", withMiddleware(statsPageHandler))
	mux.HandleFunc("GET /api/health", withHealthMiddleware(healthcheckHandler))

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

func defaultHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger := getLogger(r)

	user, err := getActiveUser(s)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error retrieving user: %w", err)
	}

	if !user.CurrentSession.Valid {
		logger.Debug("user has no active session, displaying select_playlist.html")

		playlists, err := queries.GetPlaylistsForUser(r.Context(), user.ID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("Error loading playlist for user: %w", err)
		}

		sessions, err := queries.GetNonActiveUserSessions(r.Context(), db.GetNonActiveUserSessionsParams{
			User:          user.ID,
			Activesession: user.CurrentSessionNotNull(),
		})

		logger.Debug("recieved non-active sessions", "sessions", sessions)

		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to get non active user sessions: %w", err)
		}

		selectPlaylistHtml.Execute(w, map[string]any{
			"Playlists": mapPlaylists(playlists),
			"Sessions":  mapSessions(r.Context(), logger, sessions),
		})
		return http.StatusOK, nil
	}

	logger.Debug("redirecting to /select_song")
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

func winnerHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	winnerID := r.FormValue("winner")
	if winnerID == "" {
		return http.StatusBadRequest, fmt.Errorf("no winner provided in form")
	}

	winnerItem, err := queries.GetPlaylistItem(r.Context(), winnerID)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("winner not found in DB: %w", err)
	}

	winnerHtml.Execute(w, map[string]string{
		"Image":   winnerItem.Image.String,
		"Title":   winnerItem.Title.String,
		"Artists": winnerItem.Artists.String,
	})
	return http.StatusOK, nil
}

func getIp(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}

func getActiveUser(session *sessions.Session) (*ActiveUser, error) {
	userID, ok := session.Values[session_id_key]
	if !ok {
		return nil, fmt.Errorf("User ID not found in session")
	}
	user, ok := activeUserMap.Load(userID.(string))
	if !ok {
		return nil, fmt.Errorf("User not in activeuserMap")
	}
	return user, nil
}

func getLoggerUserTransactionQueries(w http.ResponseWriter, r *http.Request, s *sessions.Session) (*slog.Logger, *ActiveUser, *sql.Tx, *db.Queries, error) {
	logger := getLogger(r)

	user, err := getActiveUser(s)
	if err != nil {
		return logger, nil, nil, nil, fmt.Errorf("failed to get activeUser, user not found: %w", err)
	}

	tx, err := db_conn.BeginTx(r.Context(), nil)
	if err != nil {
		return logger, user, nil, nil, fmt.Errorf("failed to create DB transaction: %w", err)
	}

	return logger, user, tx, queries.WithTx(tx), nil
}

func notNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

type TemplatePlaylist struct {
	Name string
	Url  string
}

func mapPlaylists(playlists []db.Playlist) []TemplatePlaylist {
	result := make([]TemplatePlaylist, 0, len(playlists))
	for _, playlist := range playlists {
		if playlist.Name.Valid && playlist.Url.Valid {
			result = append(result, TemplatePlaylist{Name: playlist.Name.String, Url: playlist.Url.String})
		}
	}
	return result
}

type TemplateSession struct {
	ID       int64
	Playlist string
}

func mapSessions(ctx context.Context, logger *slog.Logger, sessions []db.Session) []TemplateSession {
	result := make([]TemplateSession, 0, len(sessions))
	for _, session := range sessions {
		playlist, err := queries.GetPlaylist(ctx, session.Playlist)
		if err != nil {
			logger.Warn("could not get playlist from db", "err", err)
			playlist.Name = notNull(session.Playlist)
		}

		result = append(result, TemplateSession{ID: session.ID, Playlist: playlist.Name.String})
	}
	return result
}

func commitTransaction(tx *sql.Tx) (int, error) {
	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}
	return -1, nil
}
