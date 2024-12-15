package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	logger := getLogger(r)

	logger.Debug("starting healthcheck")
	healthcheckResult := performHealthcheck(logger)
	logger.Debug("healthcheck done")

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthcheckResult); err != nil {
		logger.Warn("Error marshalling healthcheck result", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !healthcheckResult.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
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
