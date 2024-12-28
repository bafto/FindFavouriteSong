package main

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const logger_key = "logger"

func getLogger(c *gin.Context, args ...any) *slog.Logger {
	logger, ok := c.Get(logger_key)
	if logger, isLogger := logger.(*slog.Logger); !ok || !isLogger {
		return slog.Default().With(args...)
	} else {
		return logger.With(args...)
	}
}

func SlogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := slog.Default().With(
			"ip", c.ClientIP(),
			"request-id", uuid.New(),
		)

		logger.Debug("Got a request", "url", c.Request.URL)
		c.Set(logger_key, logger)
		c.Next()
	}
}
