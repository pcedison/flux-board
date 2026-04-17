package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flux-board/internal/config"
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
	authBodyLimit        int64 = 8 << 10
	taskBodyLimit        int64 = 64 << 10
	loginWindow                = 15 * time.Minute
	maxLoginFailures           = 10
	loginBlockDuration         = 15 * time.Minute
	bootstrapAdmin             = "admin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := connectDatabase(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	app := newApp(cfg, pool)

	if err := app.bootstrap(); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	runtimeCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app.startBackgroundLoops(runtimeCtx)

	mux, err := newMux(app)
	if err != nil {
		log.Fatalf("build mux: %v", err)
	}
	server := newHTTPServer(cfg.Port, app.securityHeaders(mux))
	installGracefulShutdown(server, runtimeCtx)

	log.Printf("Flux Board listening on :%s (cookieSecure=%v)", cfg.Port, cfg.CookieSecure)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}
