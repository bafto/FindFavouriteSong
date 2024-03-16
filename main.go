package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

// init configuration
func init() {
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetDefault("SPOTIFY_CLIENT_ID", "")
	viper.SetDefault("SPOTIFY_CLIENT_SECRET", "")
	viper.SetDefault("DB_PATH", "ffs.db")

	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); err != nil && !ok {
		log.Fatalf("Error reading config file, %s", err)
	}
}

const (
	redirect_url = "http://localhost:8080/spotifyauthentication"
	state_length = 8
	timeout      = time.Second * 7
)

var (
	mux                = http.NewServeMux()
	server             = &http.Server{Addr: ":8080", Handler: mux}
	indexHtml          = template.Must(template.ParseFiles("index.html"))
	selectPlaylistHtml = template.Must(template.ParseFiles("select_playlist.html"))

	spClient *spotify.Client
	spAuth   *spotifyauth.Authenticator
	authURL  string
	state    = generateState(8)
)

func main() {
	spAuth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirect_url),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopeUserReadPrivate),
		spotifyauth.WithClientSecret(viper.GetString("SPOTIFY_CLIENT_SECRET")),
		spotifyauth.WithClientID(viper.GetString("SPOTIFY_CLIENT_ID")),
	)
	authURL = spAuth.AuthURL(state)

	mux.HandleFunc("/", defaultHandler)
	mux.HandleFunc("/spotifyauthentication", authHandler)
	mux.HandleFunc("/select_playlist", selectPlaylistHandler)
	mux.HandleFunc("/select_song", selectSongHandler)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %s", err)
	}
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	if spClient == nil {
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		log.Printf("Redirecting to %s\n", authURL)
		return
	}

	if playlist == nil {
		selectPlaylistHtml.Execute(w, nil)
		return
	}

	song1, song2 := playlist.nextPair()

	if song2 == nil {
		indexHtml.Execute(w, map[string]string{
			"Winner": song1.Track.Track.Name,
		})
		return
	}

	indexHtml.Execute(w, map[string]string{
		"Song1": song1.Track.Track.Name,
		"Song2": song2.Track.Track.Name,
	})
}

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	if spClient == nil {
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		log.Printf("Redirecting to %s\n", authURL)
		return
	}

	log.Println("Loading playlist")
	var err error
	if playlist, err = loadPlaylist(r.FormValue("playlist_url")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error loading playlist: %s\n", err)
		return
	}
	log.Printf("Playlist loaded with %d items\n", len(playlist.Items))
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func selectSongHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("got request %s %s\n", r.Method, r.URL)
	selected, err := strconv.Atoi(r.FormValue("select_song"))
	if err != nil {
		http.Error(w, "Invalid selection", http.StatusBadRequest)
		return
	}

	log.Println(selected)
	playlist.selected(selected)

	song1, song2 := playlist.nextPair()
	song1_name, song2_name := song1.Track.Track.Name, song2.Track.Track.Name

	if song2 == nil {
		http.Error(w, "No more songs", http.StatusNotFound)
		return
	}
	io.WriteString(w, fmt.Sprintf(`
		<option value="1">%s</option>
		<option value="2">%s</option>
	`, song1_name, song2_name))
}

func authHandler(w http.ResponseWriter, r *http.Request) {
	if spClient != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	tok, err := spAuth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Printf("Couldn't get token: %s\n", err)
		return
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Printf("State mismatch: %s != %s\n", st, state)
		return
	}

	spClient = spotify.New(spAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	log.Println("Login completed")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}
