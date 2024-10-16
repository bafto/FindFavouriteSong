package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
)

func withPanicMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered in handler", "err", err, "url", r.URL, "stacktrace", string(debug.Stack()))
				errorPage(w, fmt.Errorf("%v", err), http.StatusInternalServerError)
			}
		}()
		nextHandler(w, r)
	}
}

func errorPage(w http.ResponseWriter, err error, status int) {
	errorHtml.Execute(w, map[string]any{
		"error":       err.Error(),
		"status":      status,
		"status_text": http.StatusText(status),
	})
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
			"request-id", uuid.New(),
		)

		logger.Debug("Got a request", "url", r.URL)
		nextHandler(w, withLogger(r, logger))
	}
}

type ErrorHandlerFunc func(http.ResponseWriter, *http.Request) (int, error)

func withErrorMiddleware(nextHandler ErrorHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := getLogger(r)
		status, err := nextHandler(w, r)
		if err != nil {
			logger.Error("request handler returned error", "err", err, "status", status)
			w.WriteHeader(status)
			errorPage(w, err, status)
		}
	}
}

func withAuthMiddleware(nextHandler ErrorHandlerFunc) ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (int, error) {
		logger := getLogger(r)

		user, passwd, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			return http.StatusUnauthorized, fmt.Errorf("no BasicAuth header given")
		}

		r = withLogger(r, logger.With("basic-auth-user", user))

		if expectedPassword, ok := config.Users[user]; !ok || expectedPassword != passwd {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			return http.StatusUnauthorized, fmt.Errorf("invalid password for user")
		}

		return nextHandler(w, r)
	}
}

type SessionHandlerFunc func(http.ResponseWriter, *http.Request, *sessions.Session) (int, error)

func withSessionMiddleware(nextHandler SessionHandlerFunc) ErrorHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) (int, error) {
		logger := getLogger(r)

		session, err := sessionManager.Get(r, session_cookie)
		if err != nil && !session.IsNew {
			return http.StatusInternalServerError, fmt.Errorf("Failed to decode session: %w", err)
		}

		if user, err := getActiveUser(session); err == nil {
			r = withLogger(r, logger.With("user-spotify-id", user.ID))
		}

		return nextHandler(w, r, session)
	}
}

func withTimerMiddleware(nextHandler SessionHandlerFunc) SessionHandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, s *sessions.Session) (int, error) {
		defer func(start time.Time) {
			logger := getLogger(r)
			logger.Debug("request done", "took", time.Since(start))
		}(time.Now())
		return nextHandler(w, r, s)
	}
}

func withMiddleware(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(
		withLoggingMiddleware(
			withErrorMiddleware(
				withAuthMiddleware(
					withSessionMiddleware(
						withSpotifyAuthMiddleware(
							withTimerMiddleware(
								nextHandler,
							),
						),
					),
				),
			),
		),
	)
}
