package main

import (
	"context"
	"io/fs"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type loginAttemptState struct {
	WindowStart  time.Time
	Failures     int
	BlockedUntil time.Time
}

type contextKey string

const sessionContextKey contextKey = "session"

type sessionState struct {
	Token     string
	Username  string
	ExpiresAt time.Time
}

type authAuditEvent struct {
	Username  string
	EventType string
	Outcome   string
	ClientIP  string
	Details   string
	RequestID string
	CreatedAt int64
}

// App holds shared application dependencies.
type App struct {
	db                *pgxpool.Pool
	taskRepo          TaskRepository
	taskSvc           TaskService
	authSvc           AuthService
	bootstrapPassword string
	cookieSecure      bool
	webPreviewFS      fs.FS
	loginMu           sync.Mutex
	loginAttempts     map[string]loginAttemptState
	passwordVerifier  func(context.Context, string) (bool, error)
	readinessChecker  func(context.Context) error
	sessionGetter     func(context.Context, string) (sessionState, error)
	sessionCreator    func(context.Context, string, string, string, time.Time) error
	sessionDeleter    func(context.Context, string) error
	auditRecorder     func(context.Context, authAuditEvent) error
}
