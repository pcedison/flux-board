package main

import (
	"context"
	"io/fs"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"
	transporthttp "flux-board/internal/transport/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type sessionState = domain.Session
type authAuditEvent = domain.AuthAuditEvent

// App holds shared application dependencies.
type App struct {
	db                *pgxpool.Pool
	taskRepo          domain.TaskRepository
	taskSvc           TaskService
	authSvc           AuthService
	settingsRepo      domain.SettingsRepository
	settingsSvc       SettingsService
	authRepo          domain.AuthRepository
	authTracker       *authservice.LoginTracker
	metricsRegistry   *prometheus.Registry
	observability     *transporthttp.Observability
	appEnv            string
	version           string
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
