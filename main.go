package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

const (
	session_name   = "ffs-session"
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

	authKey       = securecookie.GenerateRandomKey(64)
	encryptionKey = securecookie.GenerateRandomKey(32)
	cookieStore   = cookie.NewStore(authKey, encryptionKey)

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
	configure_logging()

	cookieStore.Options(sessions.Options{SameSite: http.SameSiteLaxMode})

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

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: gin_log_formatter,
		Output:    io.Discard,
	}))
	r.Use(SlogMiddleware())
	r.Use(gin.BasicAuth(config.Users))
	r.Use(sessions.Sessions(session_name, cookieStore))
	r.Use(SpotifyAuthMiddleware())

	r.LoadHTMLGlob("*.gohtml")

	r.Static("/public", "./public")
	r.GET("/", defaultHandler)
	r.GET("/spotifyauthentication", authHandler)
	r.GET("/select_song", selectSongPageHandler)
	r.GET("/winner", winnerHandler)
	r.GET("/stats", statsPageHandler)

	api := r.Group("/api")
	{
		api.POST("/select_playlist", selectPlaylistHandler)
		api.POST("/select_session", selectSessionHandler)
		api.POST("/select_song", selectSongHandler)
		api.GET("/select_new_playlist", selectNewPlaylistHandler)
		api.GET("/health", healthcheckHandler)
		api.HEAD("/health", healthcheckHandler)
	}

	server := &http.Server{Addr: ":" + config.Port, Handler: r.Handler()}

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

func defaultHandler(c *gin.Context) {
	logger := getLogger(c)

	user, err := getActiveUser(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error retrieving user: %w", err))
		return
	}

	if !user.CurrentSession.Valid {
		logger.Debug("user has no active session, displaying select_playlist.html")

		playlists, err := queries.GetPlaylistsForUser(c, user.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Error loading playlist for user: %w", err))
			return
		}

		sessions, err := queries.GetNonActiveUserSessions(c, db.GetNonActiveUserSessionsParams{
			User:          user.ID,
			Activesession: user.CurrentSessionNotNull(),
		})

		logger.Debug("recieved non-active sessions", "sessions", sessions)

		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get non active user sessions: %w", err))
			return
		}

		c.HTML(http.StatusOK, "select_playlist.gohtml", gin.H{
			"Playlists": mapPlaylists(playlists),
			"Sessions":  mapSessions(c, logger, sessions),
		})
		return
	}

	logger.Debug("redirecting to /select_song")
	c.Redirect(http.StatusTemporaryRedirect, "/select_song")
}

func winnerHandler(c *gin.Context) {
	winnerID := c.Query("winner")
	if winnerID == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no winner provided in form"))
		return
	}

	winnerItem, err := queries.GetPlaylistItem(c, winnerID)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("winner not found in DB: %w", err))
		return
	}

	c.HTML(http.StatusOK, "winner.gohtml", gin.H{
		"Image":   winnerItem.Image.String,
		"Title":   winnerItem.Title.String,
		"Artists": winnerItem.Artists.String,
	})
}

func getActiveUser(c *gin.Context) (*ActiveUser, error) {
	session := sessions.Default(c)
	userID := session.Get(session_id_key)
	if userID == nil {
		return nil, fmt.Errorf("User ID not found in session")
	}
	user, ok := activeUserMap.Load(userID.(string))
	if !ok {
		return nil, fmt.Errorf("User not in activeuserMap")
	}
	return user, nil
}

func getLoggerUserTransactionQueries(c *gin.Context) (*slog.Logger, *ActiveUser, *sql.Tx, *db.Queries, error) {
	logger := getLogger(c)

	user, err := getActiveUser(c)
	if err != nil {
		return logger, nil, nil, nil, fmt.Errorf("failed to get activeUser, user not found: %w", err)
	}

	tx, err := db_conn.BeginTx(c, nil)
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
