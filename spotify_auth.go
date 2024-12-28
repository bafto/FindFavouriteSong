package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/zmb3/spotify/v2"
)

const (
	session_present_key   = "ffs-session-present"
	session_present_value = "present"
)

func SpotifyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s := sessions.Default(c)
		if s.Get(session_present_key) != session_present_value {
			logger := getLogger(c)

			state := generateState(state_length)
			stateMap.Store(c.ClientIP(), state)
			authURL := spotifyAuth.AuthURL(state)

			s.Set(session_present_key, session_present_value)
			if err := s.Save(); err != nil {
				logger.Warn("failed to save session", "err", err, "session-id", s.ID())
			}
			logger.Debug("redirecting to login page")
			c.Redirect(http.StatusTemporaryRedirect, authURL)
			c.Abort()
			return
		}

		c.Next()
	}
}

func authHandler(c *gin.Context) {
	logger := getLogger(c)

	ip := c.ClientIP()
	logger.Info("got an auth request")
	state, ok := stateMap.Load(ip)
	if !ok {
		c.AbortWithError(http.StatusForbidden, fmt.Errorf("no state for ip %s present", ip))
		return
	}

	tok, err := spotifyAuth.Token(c, state, c.Request)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("Couldn't get token: %w", err))
		return
	}
	if st := c.Query("state"); st != state {
		logger.Error("state mismatch", "expected", state, "got", st)
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("state mismatch: %w", err))
		return
	}
	stateMap.Delete(ip)
	logger.Debug("received spotify token with valid state")

	spotifyClient = spotify.New(spotifyAuth.Client(context.Background(), tok), spotify.WithRetry(true))
	logger.Debug("created spotify client")
	userData, err := spotifyClient.CurrentUser(context.Background())
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("unable to retrieve user info: %w", err))
		return
	}
	logger = logger.With("user-id", userData.ID)
	logger.Info("received user info")

	tx, err := db_conn.BeginTx(c, nil)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to create DB transaction: %w", err))
		return
	}
	defer tx.Rollback()
	queries := queries.WithTx(tx)

	user, err := queries.GetUser(c, userData.ID)
	if err != nil {
		logger.Debug("could not retrieve user from DB, adding him")
		user, err = queries.AddUser(c, userData.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("unable to load user info from db: %w", err))
			return
		}
		logger.Info("successfully added user to DB")
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err))
		return
	}

	s := sessions.Default(c)
	s.Set(session_id_key, user.ID)
	activeUserMap.Store(userData.ID, &ActiveUser{client: spotifyClient, User: user})

	if err := s.Save(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to save session: %w", err))
		return
	}
	logger.Info("Login completed")
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func generateState(length int) string {
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length+2)
	gen.Read(b)
	return fmt.Sprintf("%x", b)[2 : length+2]
}
