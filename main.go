package main

import (
	"context"
	"embed"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"flux-board/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
	CreatedAt int64
}

// App holds shared application dependencies.
type App struct {
	db                *pgxpool.Pool
	taskRepo          TaskRepository
	bootstrapPassword string
	cookieSecure      bool
	loginMu           sync.Mutex
	loginAttempts     map[string]loginAttemptState
	passwordVerifier  func(context.Context, string) (bool, error)
	sessionGetter     func(context.Context, string) (sessionState, error)
	sessionCreator    func(context.Context, string, string, string, time.Time) error
	sessionDeleter    func(context.Context, string) error
	auditRecorder     func(context.Context, authAuditEvent) error
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	app := &App{
		db:                pool,
		bootstrapPassword: cfg.AppPassword,
		cookieSecure:      cfg.CookieSecure,
		loginAttempts:     make(map[string]loginAttemptState),
	}

	if err := app.initSchema(); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	go app.archiveCleanupLoop()
	go app.sessionCleanupLoop()

	mux, err := newMux(app)
	if err != nil {
		log.Fatalf("build mux: %v", err)
	}
	server := newHTTPServer(cfg.Port, app.securityHeaders(mux))
	installGracefulShutdown(server)

	log.Printf("Flux Board listening on :%s (cookieSecure=%v)", cfg.Port, cfg.CookieSecure)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}

func (a *App) ensureBootstrapAdmin(ctx context.Context) error {
	var existingHash string
	err := a.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, bootstrapAdmin).Scan(&existingHash)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, pgx.ErrNoRows):
		hash, hashErr := bcrypt.GenerateFromPassword([]byte(a.bootstrapPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return hashErr
		}

		now := time.Now().UnixMilli()
		tx, txErr := a.db.Begin(ctx)
		if txErr != nil {
			return txErr
		}
		defer tx.Rollback(ctx)

		if _, txErr = tx.Exec(ctx, `
			INSERT INTO users (username, password_hash, role, created_at, updated_at)
			VALUES ($1, $2, 'admin', $3, $3)
		`, bootstrapAdmin, string(hash), now); txErr != nil {
			return txErr
		}

		return tx.Commit(ctx)
	default:
		return err
	}
}

// archiveCleanupLoop deletes archived tasks older than archiveRetention every hour.
func (a *App) archiveCleanupLoop() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().Add(-archiveRetention).UnixMilli()
		if _, err := a.db.Exec(context.Background(),
			`DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff); err != nil {
			log.Printf("archive cleanup error: %v", err)
		}
	}
}

func (a *App) sessionCleanupLoop() {
	ticker := time.NewTicker(sessionCleanupTicker)
	defer ticker.Stop()

	for range ticker.C {
		if _, err := a.db.Exec(context.Background(),
			`DELETE FROM sessions WHERE expires_at < $1 OR revoked_at IS NOT NULL`, time.Now().UnixMilli()); err != nil {
			log.Printf("session cleanup error: %v", err)
		}
	}
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Pragma", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
