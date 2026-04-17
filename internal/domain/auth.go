package domain

import (
	"context"
	"time"
)

type Session struct {
	Token     string
	Username  string
	ExpiresAt time.Time
}

type AuthAuditEvent struct {
	Username  string
	EventType string
	Outcome   string
	ClientIP  string
	Details   string
	RequestID string
	CreatedAt int64
}

type AuthRepository interface {
	BootstrapPasswordHash(context.Context, string) (string, error)
	EnsureBootstrapAdmin(context.Context, string, string) error
	GetActiveSession(context.Context, string) (Session, error)
	CreateSession(context.Context, string, string, string, time.Time) error
	DeleteSession(context.Context, string) error
	RecordAuthEvent(context.Context, AuthAuditEvent) error
}
