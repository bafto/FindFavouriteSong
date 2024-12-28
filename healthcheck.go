package main

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func healthcheckHandler(c *gin.Context) {
	logger := getLogger(c)

	logger.Debug("starting healthcheck")
	healthcheckResult := performHealthcheck(logger)
	logger.Debug("healthcheck done")

	status := http.StatusOK
	if !healthcheckResult.Healthy {
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, healthcheckResult)
}

type DBHealthcheckResult struct {
	Healthy bool    `json:"healthy"`
	Error   *string `json:"error,omitempty"`
}

type HealthcheckResult struct {
	Healthy  bool                `json:"healthy"`
	DBStatus DBHealthcheckResult `json:"db-status"`
}

func performHealthcheck(logger *slog.Logger) (result HealthcheckResult) {
	result.Healthy = true

	if err := db_conn.Ping(); err != nil {
		logger.Error("healthcheck found the DB to be disconnected", "err", err.Error())
		errstr := err.Error()

		result.Healthy = false
		result.DBStatus = DBHealthcheckResult{
			Healthy: false,
			Error:   &errstr,
		}
	} else {
		result.DBStatus = DBHealthcheckResult{
			Healthy: true,
			Error:   nil,
		}
	}

	return result
}
