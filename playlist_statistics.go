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

	playlistUrl := c.Query("playlist")
	if playlistUrl == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("no playlist given"))
		return
	}

	playlistId, err := getPlaylistIdFromURL(playlistUrl)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("could not extract playlist id from url"))
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

	c.JSON(http.StatusOK, result)
}
