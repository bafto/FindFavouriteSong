package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func selectNewPlaylistHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	if !user.CurrentSession.Valid {
		logger.Warn("select_new_playlist called without active session")
		return http.StatusBadRequest, fmt.Errorf("no active session exists")
	}

	if delete := r.URL.Query().Get("delete"); delete != "" {
		logger = logger.With("delete-query-param", delete)

		deleteId, err := strconv.Atoi(delete)
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("session must be a valid int: %w", err)
		}

		deleteSession, err := queries.GetSession(r.Context(), int64(deleteId))
		if err != nil {
			return http.StatusBadRequest, fmt.Errorf("delete must be a valid session id: %w", err)
		}

		logger.Debug("deleting incomplete session", "session-to-be-deleted", deleteSession.ID)

		if err := queries.DeletePossibleNextItemsForSession(r.Context(), deleteSession.ID); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete possible next items: %w", err)
		}
		if err := queries.DeleteMatchesForSession(r.Context(), deleteSession.ID); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete matches: %w", err)
		}
		if err := queries.DeleteSession(r.Context(), deleteSession.ID); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to delete session: %w", err)
		}

		logger.Debug("deleted incomplete session", "deleted-session", deleteSession.ID)
	}

	sessions, err := queries.GetNonActiveUserSessions(r.Context(), db.GetNonActiveUserSessionsParams{
		User:          user.ID,
		Activesession: user.CurrentSessionNotNull(),
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error retrieving sessions for user: %w", err)
	}

	if len(sessions) < 3 {
		if err := queries.SetUserSession(r.Context(), db.SetUserSessionParams{
			CurrentSession: sql.NullInt64{Valid: false},
			ID:             user.ID,
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to set user session: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
		}

		user.CurrentSession.Valid = false
		logger.Debug("set user session to null")

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return http.StatusTemporaryRedirect, nil
	}

	incompleteSessions := mapIncompleteSessions(r.Context(), logger, queries, sessions)

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"sessions": incompleteSessions}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error json encoding incompleteSessions: %w", err)
	}

	return http.StatusOK, nil
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
