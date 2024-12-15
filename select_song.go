package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func selectSongPageHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	if err := selectSongsHtml.Execute(w, nil); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error executing selectSongsHtml template: %w", err)
	}
	return http.StatusOK, nil
}

type SelectSongResponse struct {
	Round         int    `json:"round"`
	Matches       int    `json:"matches"`
	Song1_Title   string `json:"song1_title"`
	Song1_Artists string `json:"song1_artists"`
	Song1_Image   string `json:"song1_image"`
	Song1_ID      string `json:"song1_id"`
	Song2_Title   string `json:"song2_title"`
	Song2_Artists string `json:"song2_artists"`
	Song2_Image   string `json:"song2_image"`
	Song2_ID      string `json:"song2_id"`
}

func selectSongHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	start := time.Now()

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
	// if we have both ids, we selected a song
	// if one is missing we only retrieve the next pair
	if winnerID != "" && loserID != "" {
		logger = logger.With("winner-id", winnerID, "loser-id", loserID)
		logger.Debug("user selected song")

		if err := queries.AddMatch(r.Context(), db.AddMatchParams{
			Session:     sessionID,
			RoundNumber: currentRound,
			Winner:      winnerID,
			Loser:       loserID,
		}); err != nil {
			return http.StatusInternalServerError, fmt.Errorf("could not create match in db: %w", err)
		}

		logger.Debug("inserted match into db", "since-start", time.Since(start))
	}

	nextPair, err := queries.GetNextPair(r.Context(), db.GetNextPairParams{
		Session:      sessionID,
		CurrentRound: currentRound,
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
			Session:      sessionID,
			CurrentRound: currentRound,
		})
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("error getting next pair from DB: %w", err)
		}

		switch len(nextPair) {
		case 0:
			panic("unexpected pair length 0") // TODO: investiage error
		case 1:
			winnerID := nextPair[0].ID

			logger.Debug("found winner for session, updating db", "winner", nextPair[0].ID)
			if err := queries.SetWinner(r.Context(), db.SetWinnerParams{
				Winner: notNull(winnerID),
				ID:     sessionID,
			}); err != nil {
				return http.StatusInternalServerError, fmt.Errorf("failed to set winner in DB: %w", err)
			}

			if err := queries.SetUserSession(r.Context(), db.SetUserSessionParams{
				ID:             user.ID,
				CurrentSession: sql.NullInt64{Valid: false},
			}); err != nil {
				return http.StatusBadRequest, fmt.Errorf("unable to reset current session in DB: %w", err)
			}

			if status, err := commitTransaction(tx); err != nil {
				return status, err
			}
			user.CurrentSession.Valid = false
			logger.Debug("reset user session to NULL")

			logger.Debug("redirecting to /winner")
			http.Redirect(w, r, "/winner?winner="+url.QueryEscape(winnerID), http.StatusTemporaryRedirect)
			return http.StatusTemporaryRedirect, nil
		}
	}

	matchesCount, err := queries.CountMatchesForRound(r.Context(), db.CountMatchesForRoundParams{
		Session:     sessionID,
		RoundNumber: currentRound,
	})
	if err != nil {
		logger.Warn("could not retrieve number of matches", "err", err)
		matchesCount = 0
	}

	if status, err := commitTransaction(tx); err != nil {
		return status, err
	}

	if err := json.NewEncoder(w).Encode(SelectSongResponse{
		Round:         int(currentRound),
		Matches:       int(matchesCount),
		Song1_Title:   nextPair[0].Title.String,
		Song1_Artists: nextPair[0].Artists.String,
		Song1_Image:   nextPair[0].Image.String,
		Song1_ID:      nextPair[0].ID,
		Song2_Title:   nextPair[1].Title.String,
		Song2_Artists: nextPair[1].Artists.String,
		Song2_Image:   nextPair[1].Image.String,
		Song2_ID:      nextPair[1].ID,
	}); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error marshalling SelectSongResponse: %w", err)
	}
	logger.Debug("select_song done", "since-start", time.Since(start))
	return http.StatusOK, nil
}
