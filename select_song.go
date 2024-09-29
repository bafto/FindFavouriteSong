package main

import (
	"fmt"
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
			RoundNumber: currentRound.(int64),
			Winner:      winnerID,
			Loser:       loserID,
		}); err != nil {
			logAndErr(w, logger, "could not create match in db", http.StatusInternalServerError, "err", err)
			return
		}

	}

	nextPair, err := queries.GetNextPair(r.Context(), db.GetNextPairParams{
		Session:     sessionID,
		RoundNumber: currentRound.(int64),
	})
	if err != nil {
		logAndErr(w, logger, "error getting next pair from DB", http.StatusInternalServerError, "err", err)
		return
	}
	if len(nextPair) != 2 {
		panic(fmt.Sprintf("not yet implemented for len %d", len(nextPair)))
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
