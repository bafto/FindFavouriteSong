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

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(healthcheckResult); err != nil {
		logger.Warn("Error marshalling healthcheck result", "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

type DBHealthcheckResult struct {
	Status string  `json:"status"`
	Error  *string `json:"error,omitempty"`
}

type HealthcheckResult struct {
	Status   string              `json:"status"`
	DBStatus DBHealthcheckResult `json:"db-status"`
}

func performHealthcheck(logger *slog.Logger) (result HealthcheckResult) {
	result.Status = "UP"

	if err := db_conn.Ping(); err != nil {
		logger.Error("healthcheck found the DB to be disconnected", "err", err.Error())
		errstr := err.Error()

		result.Status = "DOWN"
		result.DBStatus = DBHealthcheckResult{
			Status: "DOWN",
			Error:  &errstr,
		}
	} else {
		result.DBStatus = DBHealthcheckResult{
			Status: "UP",
			Error:  nil,
		}
	}

	return result
}
