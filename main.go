package main

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/spf13/viper"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

var log *slog.Logger

func read_config() {
	viper.SetDefault("spotify_client_id", "")
	viper.SetDefault("spotify_client_secret", "")
	viper.SetDefault("db_path", "ffs.db")
	viper.SetDefault("port", "8080")
	viper.SetDefault("log_level", "INFO")
	viper.SetDefault("redirect_url", "http://localhost:8080/spotifyauthentication")

	viper.AddConfigPath(".")
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("ffs")
	viper.AutomaticEnv()
}

func configure_logger() {
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
	log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: addSource,
		Level:     levelVar,
	}))

	slog.SetDefault(log)
}

const (
	session_cookie = "ffs-session-cookie"
	session_id_key = "ffs-user-id"
	state_length   = 16
)

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
	stateMap      = sync.Map{}
	clientMap     = sync.Map{}
)

func main() {
	read_config()
	configure_logger()

	spotifyAuth = spotifyauth.New(
		spotifyauth.WithRedirectURL(viper.GetString("redirect_url")),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopeUserReadPrivate),
		spotifyauth.WithClientSecret(viper.GetString("SPOTIFY_CLIENT_SECRET")),
		spotifyauth.WithClientID(viper.GetString("SPOTIFY_CLIENT_ID")),
	)

	conn, err := create_db(ctx, "ffs.db")
	if err != nil {
		log.Error("Error opening DB connection", "err", err)
		os.Exit(1)
	}
	defer conn.Close()
	log.Info("Connected to database")

	queries = db.New(conn)

	mux := http.NewServeMux()

	mux.HandleFunc("/", withMiddleware(defaultHandler))
	mux.HandleFunc("/spotifyauthentication", withMiddlewareSession(authHandler))
	// mux.HandleFunc("/api/select_playlist", selectPlaylistHandler)
	// mux.HandleFunc("/api/select_song/{selected}", selectSongHandler)
	// mux.HandleFunc("/select_song", selectSongPageHandler)
	// mux.HandleFunc("/winner", winnerHandler)
	// mux.HandleFunc("/save", saveHandler)

	server := &http.Server{Addr: ":" + viper.GetString("port"), Handler: mux}
	slog.Error("Server exited", "err", server.ListenAndServe())
}

func withPanicMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered in handler", "err", err)
			}
		}()
		nextHandler(w, r)
	}
}

type SessionHandlerFunc func(http.ResponseWriter, *http.Request, *sessions.Session)

func withSessionMiddleware(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionManager.Get(r, session_cookie)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error("Could not decode session", "err", err)
			return
		}
		nextHandler(w, r, session)
	}
}

func withAuthMiddleware(nextHandler SessionHandlerFunc) SessionHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
		if s.IsNew {
			state := generateState(state_length)
			stateMap.Store(getIp(r), state)
			authURL := spotifyAuth.AuthURL(state)

			s.Save(r, w)
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			log.Info("redirecting to login page")
			return
		}

		nextHandler(w, r, s)
	}
}

func withMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(withSessionMiddleware(withAuthMiddleware(func(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
		nextHandler(w, r)
	})))
}

func withMiddlewareSession(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(withSessionMiddleware(withAuthMiddleware(nextHandler)))
}

func authHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	ip := getIp(r)
	state, ok := stateMap.Load(ip)
	if !ok {
		http.Error(w, "no state for ip present", http.StatusForbidden)
		log.Warn("no state for ip present", "ip", ip)
		return
	}

	tok, err := spotifyAuth.Token(r.Context(), state.(string), r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Warn("Couldn't get token", "err", err)
		return
	}
	if st := r.FormValue("state"); st != state {
		http.Error(w, "state mismatch", http.StatusForbidden)
		log.Warn("State mismatch", "got", st, "expected", state)
		return
	}
	stateMap.Delete(ip)

	spotifyClient = spotify.New(spotifyAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	user, err := spotifyClient.CurrentUser(context.Background())
	if err != nil {
		http.Error(w, "unable to retrieve user info", http.StatusInternalServerError)
		log.Warn("unable to retrieve user info", "err", err)
		return
	}

	s.Values[session_id_key] = user.ID
	clientMap.Store(user.ID, spotifyClient)

	log.Info("Login completed", "ip", ip)
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	selectPlaylistHtml.Execute(w, nil)
}

func getIp(r *http.Request) string {
	return strings.Split(r.RemoteAddr, ":")[0]
}

func getClient(session *sessions.Session) (*spotify.Client, bool) {
	client, ok := clientMap.Load(session.Values[session_id_key])
	if !ok {
		return nil, false
	}
	return client.(*spotify.Client), true
}
