package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

func configure_logging() {
	var level slog.Level
	if err := level.UnmarshalText(
		[]byte(config.Log_level),
	); err != nil {
		panic(err)
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(level)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: levelVar,
	}))

	slog.SetDefault(logger.With("source", "slog"))
	gin_logger = logger.With("source", "gin")
}

var gin_logger *slog.Logger

func gin_log_formatter(param gin.LogFormatterParams) string {
	if param.Latency > time.Minute {
		param.Latency = param.Latency.Truncate(time.Second)
	}

	level := slog.LevelDebug
	if param.ErrorMessage != "" {
		level = slog.LevelError
	}
	gin_logger.Log(context.Background(), level, "processed request",
		"timestamp", param.TimeStamp.Format("2006/01/02 - 15:04:05"),
		"status", param.StatusCode,
		"latency", param.Latency,
		"client-ip", param.ClientIP,
		"method", param.Method,
		"path", param.Path,
		"err", param.ErrorMessage,
	)

	return ""
}
