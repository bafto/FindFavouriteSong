package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gin-gonic/gin"
)

func selectNewPlaylistHandler(c *gin.Context) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	if !user.CurrentSession.Valid {
		logger.Warn("select_new_playlist called without active session")
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no active session exists"))
		return
	}

	if delete := c.Query("delete"); delete != "" {
		logger = logger.With("delete-query-param", delete)

		deleteId, err := strconv.Atoi(delete)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("session must be a valid int: %w", err))
			return
		}

		deleteSession, err := queries.GetSession(c, int64(deleteId))
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("delete must be a valid session id: %w", err))
			return
		}

		logger.Debug("deleting incomplete session", "session-to-be-deleted", deleteSession.ID)

		if err := queries.DeletePossibleNextItemsForSession(c, deleteSession.ID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to delete possible next items: %w", err))
			return
		}
		if err := queries.DeleteMatchesForSession(c, deleteSession.ID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to delete matches: %w", err))
			return
		}
		if err := queries.DeleteSession(c, deleteSession.ID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to delete session: %w", err))
			return
		}

		logger.Debug("deleted incomplete session", "deleted-session", deleteSession.ID)
	}

	sessions, err := queries.GetNonActiveUserSessions(c, db.GetNonActiveUserSessionsParams{
		User:          user.ID,
		Activesession: user.CurrentSessionNotNull(),
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error retrieving sessions for user: %w", err))
		return
	}

	if len(sessions) < 3 {
		if err := queries.SetUserSession(c, db.SetUserSessionParams{
			CurrentSession: sql.NullInt64{Valid: false},
			ID:             user.ID,
		}); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to set user session: %w", err))
			return
		}

		if err := tx.Commit(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err))
			return
		}

		user.CurrentSession.Valid = false
		logger.Debug("set user session to null")

		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	incompleteSessions := mapIncompleteSessions(c, logger, queries, sessions)

	if err := tx.Commit(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": incompleteSessions})
}

type IncompleteSession struct {
	ID               int64  `json:"id"`
	Playlist         string `json:"playlist"`
	Started          string `json:"started"`
	MatchesCompleted int64  `json:"matches_completed"`
}

func mapIncompleteSessions(ctx context.Context, logger *slog.Logger, queries *db.Queries, sessions []db.Session) []IncompleteSession {
	result := make([]IncompleteSession, 0, len(sessions))
	for _, session := range sessions {
		matches_completed, err := queries.GetNumberOfMatchesCompleted(ctx, session.ID)
		if err != nil {
			logger.Warn("unable to retrieve matches_completed", "err", err)
			matches_completed = -1
		}

		playlist, err := queries.GetPlaylist(ctx, session.Playlist)
		if err != nil {
			logger.Warn("could not get playlist from db", "err", err)
			playlist.Name = notNull(session.Playlist)
		}

		result = append(result, IncompleteSession{
			ID:               session.ID,
			Playlist:         playlist.Name.String,
			Started:          session.CreationTimestamp.Time.Format(time.DateOnly),
			MatchesCompleted: matches_completed,
		})
	}
	return result
}
