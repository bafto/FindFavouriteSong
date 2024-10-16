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
	return func(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
		if s.IsNew {
			logger := getLogger(r)

			state := generateState(state_length)
			stateMap.Store(getIp(r), state)
			authURL := spotifyAuth.AuthURL(state)

			if err := s.Save(r, w); err != nil {
				logger.Warn("failed to save session", "err", err, "session-name", s.Name())
			}
			logger.Debug("redirecting to login page")
			http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
			return http.StatusTemporaryRedirect, nil
		}

		return nextHandler(w, r, s)
	}
}

func authHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger := getLogger(r)

	ip := getIp(r)
	logger.Info("got an auth request")
	state, ok := stateMap.Load(ip)
	if !ok {
		return http.StatusForbidden, fmt.Errorf("no state for ip %s present", ip)
	}

	tok, err := spotifyAuth.Token(r.Context(), state, r)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Couldn't get token: %w", err)
	}
	if st := r.FormValue("state"); st != state {
		logger.Error("state mismatch", "expected", state, "got", st)
		return http.StatusInternalServerError, fmt.Errorf("state mismatch: %w", err)
	}
	stateMap.Delete(ip)
	logger.Debug("received spotify token with valid state")

	spotifyClient = spotify.New(spotifyAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	logger.Debug("created spotify client")
	userData, err := spotifyClient.CurrentUser(context.Background())
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("unable to retrieve user info: %w", err)
	}
	logger = logger.With("user-id", userData.ID)
	logger.Info("received user info")

	tx, err := db_conn.BeginTx(r.Context(), nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to create DB transaction: %w", err)
	}
	defer tx.Rollback()
	queries := queries.WithTx(tx)

	user, err := queries.GetUser(r.Context(), userData.ID)
	if err != nil {
		logger.Debug("could not retrieve user from DB, adding him")
		user, err = queries.AddUser(r.Context(), userData.ID)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("unable to load user info from db: %w", err)
		}
		logger.Info("successfully added user to DB")
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	s.Values[session_id_key] = user.ID
	activeUserMap.Store(userData.ID, &ActiveUser{client: spotifyClient, User: user})

	s.Save(r, w)
	logger.Info("Login completed")
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	return http.StatusTemporaryRedirect, nil
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}
