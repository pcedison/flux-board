package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flux-board/internal/config"
	"flux-board/internal/observability/tracing"
	authservice "flux-board/internal/service/auth"
	settingsservice "flux-board/internal/service/settings"
	taskservice "flux-board/internal/service/task"
	storepostgres "flux-board/internal/store/postgres"
	transporthttp "flux-board/internal/transport/http"
)

const (
	readHeaderTimeout          = 5 * time.Second
	readTimeout                = 15 * time.Second
	writeTimeout               = 15 * time.Second
	idleTimeout                = 60 * time.Second
	shutdownTimeout            = 10 * time.Second
	traceShutdownTimeout       = 5 * time.Second
	authBodyLimit        int64 = 8 << 10
	settingsBodyLimit    int64 = 2 << 20
	taskBodyLimit        int64 = 64 << 10
	readinessTimeout           = 2 * time.Second
)

func main() {
	logger := transporthttp.NewLogger(os.Getenv("APP_ENV"), os.Stdout)
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", slog.Any("err", err))
		os.Exit(1)
	}

	traceShutdown, err := tracing.Configure(context.Background(), tracing.Config{
		ServiceName:    "flux-board",
		ServiceVersion: os.Getenv("APP_VERSION"),
		Environment:    cfg.AppEnv,
		Endpoint:       os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Logger:         logger,
	})
	if err != nil {
		logger.Error("configure tracing", slog.Any("err", err))
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), traceShutdownTimeout)
		defer cancel()
		if shutdownErr := traceShutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("shutdown tracing", slog.Any("err", shutdownErr))
		}
	}()

	pool, err := storepostgres.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := storepostgres.InitializeSchema(
		context.Background(),
		pool,
		os.DirFS("."),
		"migrations/*.sql",
		authservice.BootstrapAdmin,
		cfg.AppPassword,
	); err != nil {
		logger.Error("init schema", slog.Any("err", err))
		os.Exit(1)
	}

	taskRepo := storepostgres.NewTaskRepository(pool)
	authRepo := storepostgres.NewAuthRepository(pool)
	settingsRepo := storepostgres.NewSettingsRepository(pool)
	taskSvc := taskservice.New(taskRepo)
	authSvc := authservice.New(authRepo, authservice.Options{
		RequestIDFromContext: transporthttp.RequestIDFromContext,
	})
	settingsSvc := settingsservice.New(authRepo, settingsRepo, taskRepo, authSvc, appVersion(), settingsservice.Options{})
	observability := transporthttp.NewObservability(transporthttp.ObservabilityOptions{
		Logger: logger,
	})

	runtimeCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go runCleanupLoop(runtimeCtx, time.Hour, "archive cleanup", logger, func(ctx context.Context) error {
		return storepostgres.CleanupArchivedTasks(ctx, pool)
	})
	go runCleanupLoop(runtimeCtx, 15*time.Minute, "session cleanup", logger, func(ctx context.Context) error {
		return storepostgres.CleanupExpiredSessions(ctx, pool, time.Now())
	})

	mux, err := transporthttp.NewMux(
		transporthttp.NewHandlerWithSettings(taskSvc, authSvc, settingsSvc, transporthttp.HandlerOptions{
			CookieSecure:         cfg.CookieSecure,
			AuthBodyLimit:        authBodyLimit,
			SettingsBodyLimit:    settingsBodyLimit,
			TaskBodyLimit:        taskBodyLimit,
			AppEnvironment:       cfg.AppEnv,
			AppVersion:           appVersion(),
			ArchiveCleanupEvery:  time.Hour,
			SessionCleanupEvery:  15 * time.Minute,
			RuntimeArtifact:      "filesystem-root-runtime",
			RuntimeOwnershipPath: "/",
			LegacyRollbackPath:   "/legacy/",
			ReadinessChecker: func(ctx context.Context) error {
				readinessCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
				defer cancel()
				return pool.Ping(readinessCtx)
			},
		}),
		transporthttp.MuxOptions{
			LegacyFS:      os.DirFS("static"),
			Observability: observability,
		},
	)
	if err != nil {
		logger.Error("build mux", slog.Any("err", err))
		os.Exit(1)
	}

	server := transporthttp.NewServer(cfg.Port, transporthttp.SecurityHeaders(mux), transporthttp.ServerOptions{
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		Observability:     observability,
	})
	transporthttp.InstallGracefulShutdown(server, runtimeCtx, shutdownTimeout)

	logger.Info("Flux Board listening", slog.String("addr", ":"+cfg.Port), slog.Bool("cookie_secure", cfg.CookieSecure))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("listen", slog.Any("err", err))
		os.Exit(1)
	}
}

func runCleanupLoop(ctx context.Context, interval time.Duration, label string, logger *slog.Logger, cleanup func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := cleanup(cleanupCtx)
			cancel()
			if err != nil && ctx.Err() == nil && !errors.Is(err, context.Canceled) {
				logger.Error("background cleanup failed", slog.String("cleanup", label), slog.Any("err", err))
			}
		}
	}
}
