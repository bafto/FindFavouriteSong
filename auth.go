package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/zmb3/spotify/v2"
)

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
