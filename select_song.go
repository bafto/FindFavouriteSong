package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gin-gonic/gin"
)

func selectSongPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "select_songs.gohtml", nil)
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

func selectSongHandler(c *gin.Context) {
	start := time.Now()

	logger, user, tx, queries, err := getLoggerUserTransactionQueries(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	logger.Debug("selected song")

	sessionID := user.CurrentSession.Int64
	logger = logger.With("session-id", sessionID)

	currentRound, err := queries.GetCurrentRound(c, sessionID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not determine current round number from DB: %w", err))
		return
	}

	winnerID, loserID := c.Query("winner"), c.Query("loser")
	// if we have both ids, we selected a song
	// if one is missing we only retrieve the next pair
	if winnerID != "" && loserID != "" {
		logger = logger.With("winner-id", winnerID, "loser-id", loserID)
		logger.Debug("user selected song")

		if err := queries.AddMatch(c, db.AddMatchParams{
			Session:     sessionID,
			RoundNumber: currentRound,
			Winner:      winnerID,
			Loser:       loserID,
		}); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not create match in db: %w", err))
			return
		}

		logger.Debug("inserted match into db", "since-start", time.Since(start))
	}

	nextPair, err := queries.GetNextPair(c, db.GetNextPairParams{
		Session:      sessionID,
		CurrentRound: currentRound,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error getting next pair from DB: %w", err))
		return
	}

	// default case is 2 where we don't have to do anything
	if len(nextPair) != 2 {
		currentRound++
		if err := queries.SetCurrentRound(c, db.SetCurrentRoundParams{
			ID:           sessionID,
			CurrentRound: currentRound,
		}); err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error updating current_round in DB: %w", err))
			return
		}

		nextPair, err = queries.GetNextPair(c, db.GetNextPairParams{
			Session:      sessionID,
			CurrentRound: currentRound,
		})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error getting next pair from DB: %w", err))
			return
		}

		switch len(nextPair) {
		case 0:
			panic("unexpected pair length 0") // TODO: investiage error
		case 1:
			winnerID := nextPair[0].ID

			logger.Debug("found winner for session, updating db", "winner", nextPair[0].ID)
			if err := queries.SetWinner(c, db.SetWinnerParams{
				Winner: notNull(winnerID),
				ID:     sessionID,
			}); err != nil {
				c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to set winner in DB: %w", err))
				return
			}

			if err := queries.SetUserSession(c, db.SetUserSessionParams{
				ID:             user.ID,
				CurrentSession: sql.NullInt64{Valid: false},
			}); err != nil {
				c.AbortWithError(http.StatusBadRequest, fmt.Errorf("unable to reset current session in DB: %w", err))
			}

			if status, err := commitTransaction(tx); err != nil {
				c.AbortWithError(status, err)
				return
			}
			user.CurrentSession.Valid = false
			logger.Debug("reset user session to NULL")

			logger.Debug("redirecting to /winner")
			c.Redirect(http.StatusTemporaryRedirect, "/winner?winner="+url.QueryEscape(winnerID))
			return
		}
	}

	matchesCount, err := queries.CountMatchesForRound(c, db.CountMatchesForRoundParams{
		Session:     sessionID,
		RoundNumber: currentRound,
	})
	if err != nil {
		logger.Warn("could not retrieve number of matches", "err", err)
		matchesCount = 0
	}

	if status, err := commitTransaction(tx); err != nil {
		c.AbortWithError(status, err)
		return
	}

	c.JSON(http.StatusOK, SelectSongResponse{
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
	})
	logger.Debug("select_song done", "since-start", time.Since(start))
}
