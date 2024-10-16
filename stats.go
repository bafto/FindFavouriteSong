package main

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gorilla/sessions"
)

func statsPageHandler(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(w, r, s)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	winners, err := getWinnerMap(r.Context(), queries, user.ID)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get winners from db: %w", err)
	}
	logger.Debug("retrieved winners from DB", "n-winners", len(winners))

	slices.SortFunc(winners, func(a, b Winner) int {
		return int(b.N - a.N)
	})

	statsHtml.Execute(w, winners)
	return http.StatusOK, nil
}

type Winner struct {
	Title   string
	Artists string
	Image   string
	N       int64
}

func getWinnerMap(ctx context.Context, queries *db.Queries, userID string) ([]Winner, error) {
	allWinnerIDs, err := queries.GetAllWinnersForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all winner IDs from DB: %w", err)
	}

	winners := map[string]Winner{}

	for _, id := range allWinnerIDs {
		if winner, ok := winners[id.String]; ok {
			winners[id.String] = Winner{
				Title:   winner.Title,
				Artists: winner.Artists,
				Image:   winner.Image,
				N:       winner.N + 1,
			}
			continue
		}

		dbWinner, err := queries.GetPlaylistItem(ctx, id.String)
		if err != nil {
			return nil, fmt.Errorf("failed to get winner for ID %s from DB: %w", id.String, err)
		}

		winners[id.String] = Winner{
			Title:   dbWinner.Title.String,
			Artists: dbWinner.Artists.String,
			Image:   dbWinner.Image.String,
			N:       1,
		}
	}

	return slices.Collect(maps.Values(winners)), nil
}
