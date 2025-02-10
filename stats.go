package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func statsPageHandler(c *gin.Context) {
	logger, user, tx, queries, err := getLoggerUserTransactionQueries(c)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	logger.Debug("fetching user playlists", "user", user.ID)
	playlists, err := queries.GetPlaylistsForUser(c, user.ID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("error fetching user playlists: %w", err))
		return
	}

	c.HTML(http.StatusOK, "stats.gohtml", mapPlaylists(playlists))
}
