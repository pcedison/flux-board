package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		session, err := a.authService().SessionFromToken(r.Context(), cookie.Value, clientIDFromRequest(r))
		if err != nil {
			if errors.Is(err, errAuthInvalidSession) {
				clearSessionCookie(w, a.cookieSecure)
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, authBodyLimit, &payload) {
		return
	}

	password := strings.TrimSpace(payload.Password)
	if password == "" {
		writeError(w, http.StatusBadRequest, "password is required")
		return
	}

	authSession, err := a.authService().Authenticate(r.Context(), password, clientIDFromRequest(r))
	if err != nil {
		switch {
		case errors.Is(err, errAuthBlocked):
			writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		case errors.Is(err, errAuthInvalidPassword):
			writeError(w, http.StatusUnauthorized, "invalid password")
		default:
			writeError(w, http.StatusInternalServerError, "db error")
		}
		return
	}

	setSessionCookie(w, authSession.Token, authSession.ExpiresAt, a.cookieSecure)
	jsonResp(w, map[string]any{"ok": true, "expiresAt": authSession.ExpiresAt.UnixMilli()})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		clearSessionCookie(w, a.cookieSecure)
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := a.authService().Logout(r.Context(), cookie.Value, clientIDFromRequest(r)); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	clearSessionCookie(w, a.cookieSecure)
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleGetSession(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jsonResp(w, map[string]any{
		"authenticated": true,
		"expiresAt":     session.ExpiresAt.UnixMilli(),
	})
}
