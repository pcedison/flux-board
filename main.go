package main

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flux-board/internal/config"
	"flux-board/internal/observability/tracing"
	transporthttp "flux-board/internal/transport/http"
)

//go:embed static
var staticFiles embed.FS

const (
	archiveRetention           = 3 * 24 * time.Hour
	sessionDuration            = 14 * 24 * time.Hour
	sessionCleanupTicker       = 15 * time.Minute
	cookieName                 = "flux_session"
	readHeaderTimeout          = 5 * time.Second
	readTimeout                = 15 * time.Second
	writeTimeout               = 15 * time.Second
	idleTimeout                = 60 * time.Second
	shutdownTimeout            = 10 * time.Second
	traceShutdownTimeout       = 5 * time.Second
	authBodyLimit        int64 = 8 << 10
	taskBodyLimit        int64 = 64 << 10
	loginWindow                = 15 * time.Minute
	maxLoginFailures           = 10
	loginBlockDuration         = 15 * time.Minute
	bootstrapAdmin             = "admin"
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

	pool, err := connectDatabase(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect db", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	app := newApp(cfg, pool)
	app.observability = transporthttp.NewObservability(transporthttp.ObservabilityOptions{
		Logger: logger,
	})

	if err := app.bootstrap(); err != nil {
		logger.Error("init schema", slog.Any("err", err))
		os.Exit(1)
	}

	runtimeCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.startBackgroundLoops(runtimeCtx)

	mux, err := newMux(app)
	if err != nil {
		logger.Error("build mux", slog.Any("err", err))
		os.Exit(1)
	}
	server := newHTTPServer(cfg.Port, app.securityHeaders(mux), app.observabilityRuntime())
	installGracefulShutdown(server, runtimeCtx)

	logger.Info("Flux Board listening", slog.String("addr", ":"+cfg.Port), slog.Bool("cookie_secure", cfg.CookieSecure))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("listen", slog.Any("err", err))
		os.Exit(1)
	}
}
