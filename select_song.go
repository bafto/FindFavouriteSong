package main

import (
	"net/http"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func selectSongHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
	logger, user, tx, queries, ok := getLoggerUserTransactionQueries(w, r, s)
	if !ok {
		return
	}
	defer tx.Rollback()

	sessionID := user.CurrentSession.Int64
	logger = logger.With("session-id", sessionID)

	currentRound, err := queries.GetCurrentRound(r.Context(), sessionID)
	if err != nil {
		logAndErr(w, logger, "could not determine current round number from DB", http.StatusInternalServerError, "err", err)
		return
	}

	winnerID, loserID := r.FormValue("winner"), r.FormValue("loser")
	if winnerID != "" && loserID != "" {
		logger = logger.With("winner-id", winnerID, "loser-id", loserID)

		if err := queries.AddMatch(r.Context(), db.AddMatchParams{
			Session:     sessionID,
			RoundNumber: currentRound,
			Winner:      winnerID,
			Loser:       loserID,
		}); err != nil {
			logAndErr(w, logger, "could not create match in db", http.StatusInternalServerError, "err", err)
			return
		}

	}

	nextPair, err := queries.GetNextPair(r.Context(), db.GetNextPairParams{
		Session:     sessionID,
		RoundNumber: currentRound,
	})
	if err != nil {
		logAndErr(w, logger, "error getting next pair from DB", http.StatusInternalServerError, "err", err)
		return
	}
	// default case is 2 where we don't have to do anything
	if len(nextPair) != 2 {
		currentRound++
		if err := queries.SetCurrentRound(r.Context(), db.SetCurrentRoundParams{
			ID:           sessionID,
			CurrentRound: currentRound,
		}); err != nil {
			logAndErr(w, logger, "error updating current_round in DB", http.StatusInternalServerError, "err", err)
			return
		}

		nextPair, err = queries.GetNextPair(r.Context(), db.GetNextPairParams{
			Session:     sessionID,
			RoundNumber: currentRound,
		})
		if err != nil {
			logAndErr(w, logger, "error getting next pair from DB", http.StatusInternalServerError, "err", err)
			return
		}

		switch len(nextPair) {
		case 0:
			panic("unexpected pair length 0")
		case 1:
			winnerID := nextPair[0].ID

			logger.Info("found winner for session, updating db", "winner", nextPair[0].ID)
			if err := queries.SetWinner(r.Context(), db.SetWinnerParams{
				Winner: notNull(winnerID),
				ID:     sessionID,
			}); err != nil {
				logAndErr(w, logger, "failed to set winner in DB", http.StatusInternalServerError, "err", err)
				return
			}

			if err := tx.Commit(); err != nil {
				logAndErr(w, logger, "failed to commit DB transaction", http.StatusInternalServerError, "err", err)
				return
			}
			logger.Info("redirecting to /winner")
			http.Redirect(w, r, "/winner?winner="+winnerID, http.StatusTemporaryRedirect)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logAndErr(w, logger, "failed to commit DB transaction", http.StatusInternalServerError, "err", err)
		return
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
}
