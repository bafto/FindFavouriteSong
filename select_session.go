package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func selectSessionHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if user.CurrentSession.Valid {
		return http.StatusBadRequest, fmt.Errorf("user already has session")
	}

	// parse playlist url
	sessionIdQuery := r.URL.Query().Get("session_id")
	logger.Debug("User selected session", "session-id", sessionIdQuery)
	sessionId, err := strconv.Atoi(sessionIdQuery)
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("session id must be a valid number")
	}

	session, err := queries.GetSession(r.Context(), int64(sessionId))
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("session does not exist")
	}

	if session.User != user.ID {
		return http.StatusBadRequest, fmt.Errorf("session does not belong to user")
	}

	if err := queries.SetUserSession(r.Context(), db.SetUserSessionParams{
		CurrentSession: sql.NullInt64{Int64: int64(sessionId), Valid: true},
		ID:             user.ID,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to set user session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	user.CurrentSession = sql.NullInt64{Int64: int64(sessionId), Valid: true}

	return http.StatusOK, nil
}
