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
			getLogger(r).Info("redirecting to login page")
			return
		}

		nextHandler(w, r, s)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger := getLogger(r)

	ip := getIp(r)
	state, ok := stateMap.Load(ip)
	if !ok {
		logAndErr(w, logger, "no state for ip present", http.StatusForbidden, "ip", ip)
		return
	}

	tok, err := spotifyAuth.Token(r.Context(), state, r)
	if err != nil {
		logAndErr(w, logger, "Couldn't get token", http.StatusForbidden, "err", err)
		return
	}
	if st := r.FormValue("state"); st != state {
		logAndErr(w, logger, "state mismatch", http.StatusForbidden, "expected", state, "got", st)
		return
	}
	stateMap.Delete(ip)

	spotifyClient = spotify.New(spotifyAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	userData, err := spotifyClient.CurrentUser(context.Background())
	if err != nil {
		logAndErr(w, logger, "unable to retrieve user info", http.StatusInternalServerError, "err", err)
		return
	}

	tx, err := db_conn.BeginTx(r.Context(), nil)
	if err != nil {
		logAndErr(w, logger, "failed to create DB transaction", http.StatusInternalServerError, "err", err)
		return
	}
	defer tx.Rollback()
	queries := queries.WithTx(tx)

	user, err := queries.GetUser(r.Context(), userData.ID)
	if err != nil {
		user, err = queries.AddUser(r.Context(), userData.ID)
		if err != nil {
			logAndErr(w, logger, "unable to load user info from db", http.StatusInternalServerError, "err", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logAndErr(w, logger, "failed to commit DB transaction", http.StatusInternalServerError, "err", err)
		return
	}

	s.Values[session_id_key] = user.ID
	activeUserMap.Store(userData.ID, &ActiveUser{client: spotifyClient, User: user})

	s.Save(r, w)
	logger.Info("Login completed")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}
