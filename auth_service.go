package main

import (
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

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
	if a.auditRecorder != nil {
		if err := a.auditRecorder(ctx, event); err != nil {
			log.Printf("auth audit recorder error: %v", err)
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
		log.Printf("auth audit insert error: %v", err)
	}
}
