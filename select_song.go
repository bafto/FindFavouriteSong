package main

import (
	"fmt"
	"net/http"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func selectSongHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	logger.Debug("selected song")

	sessionID := user.CurrentSession.Int64
	logger = logger.With("session-id", sessionID)

	currentRound, err := queries.GetCurrentRound(r.Context(), sessionID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not determine current round number from DB: %w", err)
	}

	winnerID, loserID := r.FormValue("winner"), r.FormValue("loser")
	logger = logger.With("winner-id", winnerID, "loser-id", loserID)
	if winnerID == "" || loserID == "" {
		return http.StatusBadRequest, fmt.Errorf("winner or loser missing")
	}

	if err := queries.AddMatch(r.Context(), db.AddMatchParams{
		Session:     sessionID,
		RoundNumber: currentRound,
		Winner:      winnerID,
		Loser:       loserID,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not create match in db: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to commit DB transaction: %w", err)
	}
	logger.Debug("inserted match into db")
	return http.StatusOK, nil
}

func selectSongPageHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	sessionID := user.CurrentSession.Int64
	logger = logger.With("session-id", sessionID)

	currentRound, err := queries.GetCurrentRound(r.Context(), sessionID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("could not determine current round number from DB: %w", err)
	}

	nextPair, err := queries.GetNextPair(r.Context(), db.GetNextPairParams{
		Session:     sessionID,
		RoundNumber: currentRound,
	})
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("error getting next pair from DB: %w", err)
	}
	// default case is 2 where we don't have to do anything
	if len(nextPair) != 2 {
		currentRound++
		if err := queries.SetCurrentRound(r.Context(), db.SetCurrentRoundParams{
			ID:           sessionID,
			CurrentRound: currentRound,
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error updating current_round in DB: %w", err)
		}

		nextPair, err = queries.GetNextPair(r.Context(), db.GetNextPairParams{
			Session:     sessionID,
			RoundNumber: currentRound,
		})
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error getting next pair from DB: %w", err)
		}

		switch len(nextPair) {
		case 0:
			panic("unexpected pair length 0")
		case 1:
			winnerID := nextPair[0].ID

			logger.Debug("found winner for session, updating db", "winner", nextPair[0].ID)
			if err := queries.SetWinner(r.Context(), db.SetWinnerParams{
				Winner: notNull(winnerID),
				ID:     sessionID,
			}); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to set winner in DB: %w", err)
			}

			if status, err := commitTransaction(tx); err != nil {
				return status, err
			}
			logger.Debug("redirecting to /winner")
			http.Redirect(w, r, "/winner?winner="+winnerID, http.StatusTemporaryRedirect)
			return http.StatusTemporaryRedirect, nil
		}
	}

	if status, err := commitTransaction(tx); err != nil {
		return status, err
	}

	selectSongsHtml.Execute(w, map[string]string{
		"Song1":         nextPair[0].Title.String,
		"Song1_Artists": nextPair[0].Artists.String,
		"Song1_Image":   nextPair[0].Image.String,
		"Song1_ID":      nextPair[0].ID,
		"Song2":         nextPair[1].Title.String,
		"Song2_Artists": nextPair[1].Artists.String,
		"Song2_Image":   nextPair[1].Image.String,
		"Song2_ID":      nextPair[1].ID,
	})
	return http.StatusOK, nil
}
