package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	errAuthBlocked         = errors.New("auth blocked")
	errAuthInvalidPassword = errors.New("invalid password")
	errAuthInvalidSession  = errors.New("invalid session")
)

type AuthService interface {
	Authenticate(context.Context, string, string) (authLoginResult, error)
	Logout(context.Context, string, string) error
	SessionFromToken(context.Context, string, string) (sessionState, error)
}

type authLoginResult struct {
	Token     string
	Username  string
	ExpiresAt time.Time
}

type defaultAuthService struct {
	app *App
}

func (a *App) authService() AuthService {
	if a.authSvc != nil {
		return a.authSvc
	}
	return defaultAuthService{app: a}
}

func (s defaultAuthService) Authenticate(ctx context.Context, password, clientIP string) (authLoginResult, error) {
	if !s.app.allowLoginAttempt(clientIP) {
		s.app.recordAuthEvent(ctx, authAuditEvent{
			EventType: "login",
			Outcome:   "blocked",
			ClientIP:  clientIP,
			Details:   "too many failed attempts",
		})
		return authLoginResult{}, errAuthBlocked
	}

	username := bootstrapAdmin
	match, err := s.app.verifyLoginPassword(ctx, password)
	if err != nil {
		s.app.recordAuthEvent(ctx, authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "password verification failed",
		})
		return authLoginResult{}, err
	}
	if !match {
		s.app.recordFailedLogin(clientIP)
		s.app.recordAuthEvent(ctx, authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "failed",
			ClientIP:  clientIP,
		})
		return authLoginResult{}, errAuthInvalidPassword
	}

	s.app.clearLoginAttempts(clientIP)

	token := newToken()
	expiry := time.Now().Add(sessionDuration)
	if err := s.app.createSession(ctx, token, username, clientIP, expiry); err != nil {
		s.app.recordAuthEvent(ctx, authAuditEvent{
			Username:  username,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session create failed",
		})
		return authLoginResult{}, err
	}

	s.app.recordAuthEvent(ctx, authAuditEvent{
		Username:  username,
		EventType: "login",
		Outcome:   "success",
		ClientIP:  clientIP,
	})

	return authLoginResult{
		Token:     token,
		Username:  username,
		ExpiresAt: expiry,
	}, nil
}

func (s defaultAuthService) Logout(ctx context.Context, token, clientIP string) error {
	if err := s.app.deleteSession(ctx, token); err != nil {
		s.app.recordAuthEvent(ctx, authAuditEvent{
			EventType: "logout",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session delete failed",
		})
		return err
	}

	s.app.recordAuthEvent(ctx, authAuditEvent{
		Username:  bootstrapAdmin,
		EventType: "logout",
		Outcome:   "success",
		ClientIP:  clientIP,
	})

	return nil
}

func (s defaultAuthService) SessionFromToken(ctx context.Context, token, clientIP string) (sessionState, error) {
	session, err := s.app.getActiveSession(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.app.recordAuthEvent(ctx, authAuditEvent{
				EventType: "session",
				Outcome:   "invalid",
				ClientIP:  clientIP,
				Details:   "session lookup failed",
			})
			return sessionState{}, errAuthInvalidSession
		}

		s.app.recordAuthEvent(ctx, authAuditEvent{
			EventType: "session",
			Outcome:   "error",
			ClientIP:  clientIP,
			Details:   "session lookup error",
		})
		return sessionState{}, err
	}

	return session, nil
}

func (a *App) verifyLoginPassword(ctx context.Context, given string) (bool, error) {
	if a.passwordVerifier != nil {
		return a.passwordVerifier(ctx, given)
	}
	return a.verifyBootstrapPassword(ctx, given)
}

func (a *App) verifyBootstrapPassword(ctx context.Context, given string) (bool, error) {
	var hash string
	if err := a.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, bootstrapAdmin).Scan(&hash); err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(given)) == nil, nil
}

func (a *App) getActiveSession(ctx context.Context, token string) (sessionState, error) {
	if a.sessionGetter != nil {
		return a.sessionGetter(ctx, token)
	}

	var session sessionState
	var expiresAt int64
	if err := a.db.QueryRow(ctx, `
		SELECT username, expires_at
		FROM sessions
		WHERE token=$1 AND revoked_at IS NULL AND expires_at > $2
	`, token, time.Now().UnixMilli()).Scan(&session.Username, &expiresAt); err != nil {
		return sessionState{}, err
	}

	session.Token = token
	session.ExpiresAt = time.UnixMilli(expiresAt)
	return session, nil
}

func (a *App) createSession(ctx context.Context, token, username, clientIP string, expiresAt time.Time) error {
	if a.sessionCreator != nil {
		return a.sessionCreator(ctx, token, username, clientIP, expiresAt)
	}

	now := time.Now().UnixMilli()
	_, err := a.db.Exec(ctx, `
		INSERT INTO sessions (token, username, created_at, expires_at, revoked_at, last_seen_at, client_ip)
		VALUES ($1, $2, $3, $4, NULL, $3, $5)
	`, token, username, now, expiresAt.UnixMilli(), clientIP)
	return err
}

func (a *App) deleteSession(ctx context.Context, token string) error {
	if a.sessionDeleter != nil {
		return a.sessionDeleter(ctx, token)
	}

	_, err := a.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

func (a *App) recordAuthEvent(ctx context.Context, event authAuditEvent) {
	event.CreatedAt = time.Now().UnixMilli()
	if event.RequestID == "" {
		event.RequestID = requestIDFromContext(ctx)
	}
	if a.auditRecorder != nil {
		if err := a.auditRecorder(ctx, event); err != nil {
			log.Printf("auth audit recorder error request_id=%s: %v", event.RequestID, err)
		}
		return
	}
	if a.db == nil {
		return
	}

	if _, err := a.db.Exec(ctx, `
		INSERT INTO auth_audit_logs (username, event_type, outcome, client_ip, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.Username, event.EventType, event.Outcome, event.ClientIP, event.Details, event.CreatedAt); err != nil {
		log.Printf("auth audit insert error request_id=%s: %v", event.RequestID, err)
	}
}
