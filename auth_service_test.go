package main

import (
	"context"
	"errors"
	"testing"
	"time"

	authservice "flux-board/internal/service/auth"

	"github.com/jackc/pgx/v5"
)

func TestAuthServiceAuthenticateSuccess(t *testing.T) {
	var createdToken string
	var createdUser string
	var auditEvents []authAuditEvent

	app := &App{
		passwordVerifier: func(context.Context, string) (bool, error) {
			return true, nil
		},
		sessionCreator: func(_ context.Context, token, username, _ string, expiresAt time.Time) error {
			if token == "" {
				t.Fatal("expected token to be generated")
			}
			if expiresAt.Before(time.Now()) {
				t.Fatalf("expected future expiry, got %v", expiresAt)
			}
			createdToken = token
			createdUser = username
			return nil
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			auditEvents = append(auditEvents, event)
			return nil
		},
	}

	result, err := app.authService().Authenticate(context.Background(), "secret", "127.0.0.1")
	if err != nil {
		t.Fatalf("Authenticate returned error: %v", err)
	}
	if result.Token == "" || result.Token != createdToken {
		t.Fatalf("expected generated token, got %+v", result)
	}
	if result.Username != bootstrapAdmin || createdUser != bootstrapAdmin {
		t.Fatalf("expected bootstrap admin session, got %+v", result)
	}
	if len(auditEvents) != 1 || auditEvents[0].Outcome != "success" {
		t.Fatalf("expected one success audit event, got %+v", auditEvents)
	}
}

func TestAuthServiceAuthenticateBlocksBeforePasswordCheck(t *testing.T) {
	tracker := authservice.NewLoginTracker()
	clientID := "127.0.0.1"
	now := time.Now()
	for i := 0; i < authservice.MaxLoginFailures; i++ {
		tracker.RecordFailure(clientID, now)
	}

	app := &App{
		authTracker: tracker,
		passwordVerifier: func(context.Context, string) (bool, error) {
			t.Fatal("password verifier should not be called while blocked")
			return false, nil
		},
	}

	_, err := app.authService().Authenticate(context.Background(), "secret", clientID)
	if !errors.Is(err, errAuthBlocked) {
		t.Fatalf("expected errAuthBlocked, got %v", err)
	}
}

func TestAuthServiceSessionFromTokenMapsInvalidSession(t *testing.T) {
	var auditEvents []authAuditEvent
	app := &App{
		sessionGetter: func(context.Context, string) (sessionState, error) {
			return sessionState{}, pgx.ErrNoRows
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			auditEvents = append(auditEvents, event)
			return nil
		},
	}

	_, err := app.authService().SessionFromToken(context.Background(), "missing", "127.0.0.1")
	if !errors.Is(err, errAuthInvalidSession) {
		t.Fatalf("expected errAuthInvalidSession, got %v", err)
	}
	if len(auditEvents) != 1 || auditEvents[0].Outcome != "invalid" {
		t.Fatalf("expected invalid-session audit event, got %+v", auditEvents)
	}
}

func TestAuthServiceLogoutMapsDeleteFailure(t *testing.T) {
	var auditEvents []authAuditEvent
	app := &App{
		sessionDeleter: func(context.Context, string) error {
			return context.DeadlineExceeded
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			auditEvents = append(auditEvents, event)
			return nil
		},
	}

	err := app.authService().Logout(context.Background(), "session-token", "127.0.0.1")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected delete failure, got %v", err)
	}
	if len(auditEvents) != 1 || auditEvents[0].Outcome != "error" {
		t.Fatalf("expected logout error audit event, got %+v", auditEvents)
	}
}
