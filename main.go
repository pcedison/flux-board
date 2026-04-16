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
	maxTitleLength             = 120
	maxNoteLength              = 4000
	loginWindow                = 15 * time.Minute
	maxLoginFailures           = 10
	loginBlockDuration         = 15 * time.Minute
	bootstrapAdmin             = "admin"
)

// Task represents an active task in the kanban board.
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Note        string `json:"note"`
	Due         string `json:"due"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	LastUpdated int64  `json:"lastUpdated"`
}

// ArchivedTask represents a soft-deleted task.
type ArchivedTask struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Note       string `json:"note"`
	Due        string `json:"due"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	SortOrder  int    `json:"sort_order"`
	ArchivedAt int64  `json:"archivedAt"`
}

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
