package main

import (
	"fmt"
	"net/http"

	"github.com/bafto/FindFavouriteSong/db"
	"github.com/gin-gonic/gin"
)

func playlistStatisticsHandler(c *gin.Context) {
	user, err := getActiveUser(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to get activeUser, user not found: %w", err))
		return
	}

	playlistId := c.Query("playlist")
	if playlistId == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no playlist given"))
		return
	}

	result, err := queries.GetStatistics1(c, db.GetStatistics1Params{
		User:     user.ID,
		Playlist: playlistId,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("failed to retreive statistics: %w", err))
		return
	}

	c.JSON(http.StatusOK, Statistics1ToJson(result))
}

type GetStatisticsJsonResult struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Artists string `json:"artists"`
	Image   string `json:"image"`
	Points  int64  `json:"points"`
}

func Statistics1ToJson(result []db.GetStatistics1Row) []GetStatisticsJsonResult {
	ret := make([]GetStatisticsJsonResult, len(result))
	for i, result := range result {
		ret[i] = GetStatisticsJsonResult{
			ID:      result.ID,
			Title:   result.Title.String,
			Artists: result.Artists.String,
			Image:   result.Image.String,
			Points:  result.Points,
		}
	}
	return ret
}
