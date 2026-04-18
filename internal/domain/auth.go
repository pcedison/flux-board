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
	BootstrapAdminExists(context.Context, string) (bool, error)
	UpdatePasswordHash(context.Context, string, string, int64) error
	GetActiveSession(context.Context, string) (Session, error)
	CreateSession(context.Context, string, string, string, time.Time) error
	DeleteSession(context.Context, string) error
	ListSessions(context.Context, string) ([]SessionInfo, error)
	DeleteSessionsExcept(context.Context, string, []string) error
	RecordAuthEvent(context.Context, AuthAuditEvent) error
}
