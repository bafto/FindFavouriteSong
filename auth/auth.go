package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

func init() {
	viper.SetConfigName("ffs_config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AutomaticEnv()
	viper.SetDefault("SPOTIFY_CLIENT_ID", "")
	viper.SetDefault("SPOTIFY_CLIENT_SECRET", "")

	err := viper.ReadInConfig()
	if _, ok := err.(viper.ConfigFileNotFoundError); err != nil && !ok {
		log.Fatalf("Error reading config file, %s", err)
	}
}

const (
	redirect_url = "http://localhost:8080/spotifyauthentication"
	state_length = 8
)

var (
	auth                *spotifyauth.Authenticator
	clientChan, errChan = make(chan *spotify.Client), make(chan error)
	state               = generateState(8)
)

// https://github.com/zmb3/spotify/blob/master/examples/authenticate/authcode/authenticate.go
func Authenticate() (*spotify.Client, error) {
	auth_mux := http.NewServeMux()
	auth_server := &http.Server{Addr: ":8080", Handler: auth_mux}

	auth_mux.HandleFunc("/spotifyauthentication", authCallback)
	auth_mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, redirect_url, http.StatusSeeOther)
	})

	wg := &sync.WaitGroup{}

	shutdown_auth_server := func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := auth_server.Shutdown(ctx); err != nil {
			return fmt.Errorf("error shutting down server: %w", err)
		}
		wg.Wait()

		return nil
	}

	auth = spotifyauth.New(
		spotifyauth.WithRedirectURL(redirect_url),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopeUserReadPrivate),
		spotifyauth.WithClientSecret(viper.GetString("SPOTIFY_CLIENT_SECRET")),
		spotifyauth.WithClientID(viper.GetString("SPOTIFY_CLIENT_ID")),
	)

	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := auth_server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %s", err)
		}
	}()

	url := auth.AuthURL(state)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)
	fmt.Printf("Expecting state %s\n", state)

	select {
	case client := <-clientChan:
		return client, shutdown_auth_server()
	case err := <-errChan:
		if shutdown_err := shutdown_auth_server(); shutdown_err != nil {
			return nil, errors.Join(err, shutdown_err)
		}
		return nil, err
	}
}

func authCallback(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(r.Context(), state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		errChan <- err
		return
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		errChan <- fmt.Errorf("state mismatch: %s != %s", st, state)
		return
	}

	client := spotify.New(auth.Client(context.Background(), tok), spotify.WithRetry(true))
	fmt.Println("Login completed")
	clientChan <- client
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}
