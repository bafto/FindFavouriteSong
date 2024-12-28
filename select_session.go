package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gin-gonic/gin"
)

func selectSessionHandler(c *gin.Context) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	if user.CurrentSession.Valid {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("user already has session"))
		return
	}

	// parse playlist url
	sessionIdQuery := c.Query("session_id")
	logger.Debug("User selected session", "session-id", sessionIdQuery)
	sessionId, err := strconv.Atoi(sessionIdQuery)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("session id must be a valid number"))
		return
	}

	session, err := queries.GetSession(c, int64(sessionId))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("session does not exist"))
		return
	}

	if session.User != user.ID {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("session does not belong to user"))
		return
	}

	if err := queries.SetUserSession(c, db.SetUserSessionParams{
		CurrentSession: sql.NullInt64{Int64: int64(sessionId), Valid: true},
		ID:             user.ID,
	}); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to set user session: %w", err))
		return
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err))
		return
	}

	user.CurrentSession = sql.NullInt64{Int64: int64(sessionId), Valid: true}
}
