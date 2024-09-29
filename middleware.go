package main

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/sessions"
)

func withPanicMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered in handler", "err", err)
			}
		}()
		nextHandler(w, r)
	}
}

type loggerkey struct{}

func getLogger(r *http.Request, args ...any) *slog.Logger {
	logger := r.Context().Value(loggerkey{})
	if logger, ok := logger.(*slog.Logger); !ok {
		return slog.Default().With(args...)
	} else {
		return logger.With(args...)
	}
}

func withLogger(r *http.Request, logger *slog.Logger) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), loggerkey{}, logger))
}

func withLoggingMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With(
			"ip", getIp(r),
		)

		logger.Debug("Got a request", "url", r.URL)
		nextHandler(w, withLogger(r, logger))
	}
}

type SessionHandlerFunc func(http.ResponseWriter, *http.Request, *sessions.Session)

func withSessionMiddleware(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := getLogger(r)

		session, err := sessionManager.Get(r, session_cookie)
		if err != nil && !session.IsNew {
			logAndErr(w, getLogger(r), "Could not decode session", http.StatusInternalServerError, "err", err)
			return
		}

		if user, ok := getActiveUser(session); ok {
			r = withLogger(r, logger.With("user-id", user.ID))
		}

		nextHandler(w, r, session)
	}
}

func withMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(
		withLoggingMiddleware(
			withSessionMiddleware(
				withAuthMiddleware(func(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
					nextHandler(w, r)
				}),
			),
		),
	)
}

func withMiddlewareSession(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(
		withLoggingMiddleware(
			withSessionMiddleware(
				withAuthMiddleware(nextHandler),
			),
		),
	)
}
