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

func withSpotifyAuthMiddleware(nextHandler SessionHandlerFunc) SessionHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
		if s.IsNew {
			logger := getLogger(r)

			state := generateState(state_length)
			stateMap.Store(getIp(r), state)
			authURL := spotifyAuth.AuthURL(state)

			if err := s.Save(r, w); err != nil {
				logger.Warn("failed to save session", "err", err, "session-name", s.Name())
			}
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			logger.Info("redirecting to login page")
			return
		}

		nextHandler(w, r, s)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger := getLogger(r)

	ip := getIp(r)
	logger.Info("got an auth request")
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
	logger.Info("received spotify token with valid state")

	spotifyClient = spotify.New(spotifyAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	logger.Info("created spotify client")
	userData, err := spotifyClient.CurrentUser(context.Background())
	if err != nil {
		logAndErr(w, logger, "unable to retrieve user info", http.StatusInternalServerError, "err", err)
		return
	}
	logger = logger.With("user-id", userData.ID)
	logger.Info("received user info")

	tx, err := db_conn.BeginTx(r.Context(), nil)
	if err != nil {
		logAndErr(w, logger, "failed to create DB transaction", http.StatusInternalServerError, "err", err)
		return
	}
	defer tx.Rollback()
	queries := queries.WithTx(tx)

	user, err := queries.GetUser(r.Context(), userData.ID)
	if err != nil {
		logger.Info("could not retrieve user from DB, adding him")
		user, err = queries.AddUser(r.Context(), userData.ID)
		if err != nil {
			logAndErr(w, logger, "unable to load user info from db", http.StatusInternalServerError, "err", err)
			return
		}
		logger.Info("successfully added user to DB")
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
