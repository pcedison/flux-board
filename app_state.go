package main

import (
	"context"
	"io/fs"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionState = domain.Session
type authAuditEvent = domain.AuthAuditEvent

// App holds shared application dependencies.
type App struct {
	db                *pgxpool.Pool
	taskRepo          domain.TaskRepository
	taskSvc           TaskService
	authSvc           AuthService
	authRepo          domain.AuthRepository
	authTracker       *authservice.LoginTracker
	bootstrapPassword string
	cookieSecure      bool
	webRuntimeFS      fs.FS
	passwordVerifier  func(context.Context, string) (bool, error)
	readinessChecker  func(context.Context) error
	sessionGetter     func(context.Context, string) (sessionState, error)
	sessionCreator    func(context.Context, string, string, string, time.Time) error
	sessionDeleter    func(context.Context, string) error
	auditRecorder     func(context.Context, authAuditEvent) error
}
