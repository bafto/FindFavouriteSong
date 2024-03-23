package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
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
	selectSongsHtml    = template.Must(template.ParseFiles("select_songs.html"))
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
	mux.HandleFunc("/api/select_playlist", selectPlaylistHandler)
	mux.HandleFunc("/api/select_song/{selected}", selectSongHandler)
	mux.HandleFunc("/select_song", selectSongPageHandler)
	mux.HandleFunc("/winner", winnerHandler)
	mux.HandleFunc("/save", saveHandler)

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

	http.NotFound(w, r)
}

func selectPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	if spClient == nil {
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
		log.Printf("Redirecting to %s\n", authURL)
		return
	}

	var err error
	if playlist, err = loadPlaylist(r.FormValue("playlist_url")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("Error loading playlist: %s\n", err)
		return
	}
	http.Redirect(w, r, "/select_song", http.StatusTemporaryRedirect)
}

func selectSongHandler(w http.ResponseWriter, r *http.Request) {
	selected, err := strconv.Atoi(r.PathValue("selected"))
	if err != nil || selected < 1 || selected > 2 {
		http.Error(w, "Invalid selection", http.StatusBadRequest)
		return
	}

	playlist.selected(selected)

	song1, song2 := playlist.nextPair()
	song1_name, song2_name := song1.Title, song2.Title

	if song2 == nil {
		w.Header().Add("HX-Redirect", fmt.Sprintf("/winner"))
		return
	}
	io.WriteString(w, fmt.Sprintf(`
		<button hx-get="/api/select_song/1" hx-target="#form" hx-swap="innerHTML">
			<img src="%s" />
			<h3>%s</h3>
			<h4>%s</h4>
		</button>
		<button hx-get="/api/select_song/2" hx-target="#form" hx-swap="innerHTML">
			<img src="%s" />
			<h3>%s</h3>
			<h4>%s</h4>
		</button>
	`, song1.Image, song1_name, song1.artistsString(), song2.Image, song2_name, song2.artistsString()))
}

func selectSongPageHandler(w http.ResponseWriter, r *http.Request) {
	song1, song2 := playlist.nextPair()

	if song2 == nil {
		w.Header().Add("HX-Redirect", fmt.Sprintf("/winner"))
		http.Redirect(w, r, url.PathEscape(fmt.Sprintf("/winner", song1.Title)), http.StatusTemporaryRedirect)
		return
	}

	selectSongsHtml.Execute(w, map[string]string{
		"Song1":         song1.Title,
		"Song1_Artists": song1.artistsString(),
		"Song1_Image":   song1.Image,
		"Song2":         song2.Title,
		"Song2_Artists": song2.artistsString(),
		"Song2_Image":   song2.Image,
	})
}

func winnerHandler(w http.ResponseWriter, r *http.Request) {
	winner := playlist.Winner
	if winner == nil {
		http.NotFound(w, r)
		return
	}

	if err := playlist.save(); err != nil {
		log.Printf("Error saving playlist: %s\n", err)
	}
	playlist = nil
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>

<head>
	<title>Find Favourite Song</title>
</head>


<body>
	<h1>Winner</h1>
	<div>
		<img src="%s" />
		<h3>%s</h3>
		<h4>%s</h4>
	</div>
</body>

</html>
	`, winner.Image, winner.Title, winner.artistsString())
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if err := playlist.save(); err != nil {
		log.Printf("Error saving playlist: %s\n", err)
		http.Error(w, fmt.Sprintf("Error saving playlist: %s\n", err), http.StatusInternalServerError)
	}
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
