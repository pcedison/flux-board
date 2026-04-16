package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		session, err := a.getActiveSession(r.Context(), cookie.Value)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				a.recordAuthEvent(r.Context(), authAuditEvent{
					EventType: "session",
					Outcome:   "error",
					ClientIP:  clientIDFromRequest(r),
					Details:   "session lookup error",
				})
				writeError(w, http.StatusInternalServerError, "db error")
				return
			}
			a.recordAuthEvent(r.Context(), authAuditEvent{
				EventType: "session",
				Outcome:   "invalid",
				ClientIP:  clientIDFromRequest(r),
				Details:   "session lookup failed",
			})
			clearSessionCookie(w, a.cookieSecure)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next(w, r.WithContext(ctx))
	}
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFromRequest(r)
	if !a.allowLoginAttempt(clientID) {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			EventType: "login",
			Outcome:   "blocked",
			ClientIP:  clientID,
			Details:   "too many failed attempts",
		})
		writeError(w, http.StatusTooManyRequests, "too many attempts, try again later")
		return
	}

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

	username := bootstrapAdmin
	match, err := a.verifyLoginPassword(r.Context(), password)
	if err != nil {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientID,
			Details:   "password verification failed",
		})
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if !match {
		a.recordFailedLogin(clientID)
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "failed",
			ClientIP:  clientID,
		})
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}

	a.clearLoginAttempts(clientID)

	token := newToken()
	expiry := time.Now().Add(sessionDuration)
	if err := a.createSession(r.Context(), token, username, clientID, expiry); err != nil {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientID,
			Details:   "session create failed",
		})
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	a.recordAuthEvent(r.Context(), authAuditEvent{
		Username:  username,
		EventType: "login",
		Outcome:   "success",
		ClientIP:  clientID,
	})
	setSessionCookie(w, token, expiry, a.cookieSecure)
	jsonResp(w, map[string]any{"ok": true, "expiresAt": expiry.UnixMilli()})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFromRequest(r)
	cookie, err := r.Cookie(cookieName)
	if err == nil {
		if delErr := a.deleteSession(r.Context(), cookie.Value); delErr != nil {
			a.recordAuthEvent(r.Context(), authAuditEvent{
				EventType: "logout",
				Outcome:   "error",
				ClientIP:  clientID,
				Details:   "session delete failed",
			})
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  bootstrapAdmin,
			EventType: "logout",
			Outcome:   "success",
			ClientIP:  clientID,
		})
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

func setSessionCookie(w http.ResponseWriter, token string, expiry time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func sessionFromContext(ctx context.Context) (sessionState, bool) {
	session, ok := ctx.Value(sessionContextKey).(sessionState)
	return session, ok
}

func clientIDFromRequest(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (a *App) allowLoginAttempt(clientID string) bool {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	return !now.Before(state.BlockedUntil)
}

func (a *App) recordFailedLogin(clientID string) {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	if state.WindowStart.IsZero() || now.Sub(state.WindowStart) > loginWindow {
		state = loginAttemptState{WindowStart: now}
	}

	state.Failures++
	if state.Failures >= maxLoginFailures {
		state.BlockedUntil = now.Add(loginBlockDuration)
		state.Failures = 0
		state.WindowStart = now
	}

	a.loginAttempts[clientID] = state
}

func (a *App) clearLoginAttempts(clientID string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	delete(a.loginAttempts, clientID)
}

func newToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("generate token: %v", err)
	}
	return hex.EncodeToString(bytes)
}
