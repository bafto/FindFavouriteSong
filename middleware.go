package main

import (
	"net/http"

	"github.com/gorilla/sessions"
)

func withPanicMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("panic recovered in handler", "err", err)
			}
		}()
		nextHandler(w, r)
	}
}

type SessionHandlerFunc func(http.ResponseWriter, *http.Request, *sessions.Session)

func withSessionMiddleware(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := sessionManager.Get(r, session_cookie)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error("Could not decode session", "err", err)
			return
		}
		nextHandler(w, r, session)
	}
}

func withMiddleware(nextHandler http.HandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(
		withSessionMiddleware(
			withAuthMiddleware(func(w http.ResponseWriter, r *http.Request, s *sessions.Session) {
				nextHandler(w, r)
			}),
		),
	)
}

func withMiddlewareSession(nextHandler SessionHandlerFunc) http.HandlerFunc {
	return withPanicMiddleware(
		withSessionMiddleware(
			withAuthMiddleware(nextHandler),
		),
	)
}
