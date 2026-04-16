package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

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
	dbURL := mustEnv("DATABASE_URL")
	password := mustEnv("APP_PASSWORD")
	port := getEnv("PORT", "8080")
	cookieSecure := getEnv("APP_ENV", "production") != "development"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	app := &App{
		db:                pool,
		bootstrapPassword: password,
		cookieSecure:      cookieSecure,
		loginAttempts:     make(map[string]loginAttemptState),
	}

	if err := app.initSchema(); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	go app.archiveCleanupLoop()
	go app.sessionCleanupLoop()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", app.handleLogout)
	mux.HandleFunc("GET /api/auth/me", app.auth(app.handleGetSession))

	mux.HandleFunc("GET /api/tasks", app.auth(app.handleGetTasks))
	mux.HandleFunc("POST /api/tasks", app.auth(app.handleCreateTask))
	mux.HandleFunc("PUT /api/tasks/{id}", app.auth(app.handleUpdateTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", app.auth(app.handleArchiveTask))

	mux.HandleFunc("GET /api/archived", app.auth(app.handleGetArchived))
	mux.HandleFunc("POST /api/archived/{id}/restore", app.auth(app.handleRestoreTask))
	mux.HandleFunc("DELETE /api/archived/{id}", app.auth(app.handleDeleteArchived))

	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(stripped)))

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           app.securityHeaders(mux),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	shutdownSignals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-shutdownSignals.Done()
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
	}()

	log.Printf("Flux Board listening on :%s (cookieSecure=%v)", port, cookieSecure)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
}

// initSchema creates tables on first run; safe to call repeatedly.
func (a *App) initSchema() error {
	_, err := a.db.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS tasks (
			id           TEXT    PRIMARY KEY,
			title        TEXT    NOT NULL,
			note         TEXT    NOT NULL DEFAULT '',
			due          TEXT    NOT NULL,
			priority     TEXT    NOT NULL DEFAULT 'medium',
			status       TEXT    NOT NULL DEFAULT 'queued',
			sort_order   INTEGER NOT NULL DEFAULT 0,
			last_updated BIGINT  NOT NULL
		);
		CREATE TABLE IF NOT EXISTS archived_tasks (
			id          TEXT   PRIMARY KEY,
			title       TEXT   NOT NULL,
			note        TEXT   NOT NULL DEFAULT '',
			due         TEXT   NOT NULL,
			priority    TEXT   NOT NULL,
			status      TEXT   NOT NULL,
			archived_at BIGINT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS users (
			username      TEXT PRIMARY KEY,
			password_hash TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'admin',
			created_at    BIGINT NOT NULL,
			updated_at    BIGINT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS sessions (
			token        TEXT PRIMARY KEY,
			username     TEXT NOT NULL REFERENCES users(username) ON DELETE CASCADE,
			created_at   BIGINT NOT NULL,
			expires_at   BIGINT NOT NULL,
			revoked_at   BIGINT,
			last_seen_at BIGINT NOT NULL,
			client_ip    TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
		CREATE TABLE IF NOT EXISTS auth_audit_logs (
			id         BIGSERIAL PRIMARY KEY,
			username   TEXT NOT NULL DEFAULT '',
			event_type TEXT NOT NULL,
			outcome    TEXT NOT NULL,
			client_ip  TEXT NOT NULL DEFAULT '',
			details    TEXT NOT NULL DEFAULT '',
			created_at BIGINT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_auth_audit_logs_created_at ON auth_audit_logs(created_at);
	`)
	if err != nil {
		return err
	}
	return a.ensureBootstrapAdmin(context.Background())
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

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		session, err := a.getActiveSession(r.Context(), cookie.Value)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				a.recordAuthEvent(r.Context(), authAuditEvent{
					EventType: "session",
					Outcome:   "error",
					ClientIP:  clientIDFromRequest(r),
					Details:   "session lookup error",
				})
				writeError(w, http.StatusInternalServerError, "db error")
				return
			}
			a.recordAuthEvent(r.Context(), authAuditEvent{
				EventType: "session",
				Outcome:   "invalid",
				ClientIP:  clientIDFromRequest(r),
				Details:   "session lookup failed",
			})
			clearSessionCookie(w, a.cookieSecure)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next(w, r.WithContext(ctx))
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

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFromRequest(r)
	if !a.allowLoginAttempt(clientID) {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  bootstrapAdmin,
			EventType: "login",
			Outcome:   "blocked",
			ClientIP:  clientID,
			Details:   "too many login attempts",
		})
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, authBodyLimit, &body) {
		return
	}

	passwordValid, err := a.verifyLoginPassword(r.Context(), body.Password)
	if err != nil {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  bootstrapAdmin,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientID,
			Details:   "password verification failed",
		})
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if !passwordValid {
		a.recordFailedLogin(clientID)
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  bootstrapAdmin,
			EventType: "login",
			Outcome:   "failed",
			ClientIP:  clientID,
			Details:   "invalid credentials",
		})
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	a.clearLoginAttempts(clientID)

	token := newToken()
	now := time.Now()
	exp := now.Add(sessionDuration)
	if err := a.createSession(r.Context(), token, bootstrapAdmin, clientID, exp); err != nil {
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  bootstrapAdmin,
			EventType: "login",
			Outcome:   "error",
			ClientIP:  clientID,
			Details:   "session creation failed",
		})
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	setSessionCookie(w, token, exp, a.cookieSecure)
	a.recordAuthEvent(r.Context(), authAuditEvent{
		Username:  bootstrapAdmin,
		EventType: "login",
		Outcome:   "success",
		ClientIP:  clientID,
	})
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(cookieName); err == nil {
		clientID := clientIDFromRequest(r)
		session, sessionErr := a.getActiveSession(r.Context(), cookie.Value)
		if err := a.deleteSession(r.Context(), cookie.Value); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		username := bootstrapAdmin
		if sessionErr == nil && session.Username != "" {
			username = session.Username
		}
		a.recordAuthEvent(r.Context(), authAuditEvent{
			Username:  username,
			EventType: "logout",
			Outcome:   "success",
			ClientIP:  clientID,
		})
	}
	clearSessionCookie(w, a.cookieSecure)
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleGetSession(w http.ResponseWriter, r *http.Request) {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	jsonResp(w, map[string]any{
		"authenticated": true,
		"expiresAt":     session.ExpiresAt.UnixMilli(),
	})
}

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(),
		`SELECT id, title, note, due, priority, status, sort_order, last_updated
		 FROM tasks
		 ORDER BY status, sort_order, last_updated DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Note,
			&task.Due,
			&task.Priority,
			&task.Status,
			&task.SortOrder,
			&task.LastUpdated,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, map[string]any{"tasks": tasks})
}

func (a *App) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if !decodeJSON(w, r, taskBodyLimit, &task) {
		return
	}

	if err := validateTaskPayload(&task, true, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task.Status = "queued"
	task.LastUpdated = time.Now().UnixMilli()

	var maxOrd int
	if err := a.db.QueryRow(r.Context(),
		`SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE status = 'queued'`).Scan(&maxOrd); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	task.SortOrder = maxOrd + 1

	tag, err := a.db.Exec(r.Context(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO NOTHING`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "task id already exists")
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResp(w, task)
}

func (a *App) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	var task Task
	if !decodeJSON(w, r, taskBodyLimit, &task) {
		return
	}

	if err := validateTaskPayload(&task, false, true); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if task.SortOrder < 0 {
		writeError(w, http.StatusBadRequest, "invalid sort order")
		return
	}

	task.LastUpdated = time.Now().UnixMilli()

	tag, err := a.db.Exec(r.Context(),
		`UPDATE tasks
		 SET title=$1, note=$2, due=$3, priority=$4, status=$5, sort_order=$6, last_updated=$7
		 WHERE id=$8`,
		task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	task.ID = id
	jsonResp(w, task)
}

// handleArchiveTask moves a task from tasks to archived_tasks.
func (a *App) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	now := time.Now().UnixMilli()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	var task ArchivedTask
	err = tx.QueryRow(r.Context(),
		`DELETE FROM tasks WHERE id=$1
		 RETURNING id, title, note, due, priority, status`, id).
		Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.ArchivedAt = now
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO archived_tasks (id, title, note, due, priority, status, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.ArchivedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
}

func (a *App) handleGetArchived(w http.ResponseWriter, r *http.Request) {
	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
	if _, err := a.db.Exec(r.Context(),
		`DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	rows, err := a.db.Query(r.Context(),
		`SELECT id, title, note, due, priority, status, archived_at
		 FROM archived_tasks
		 ORDER BY archived_at DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	tasks := make([]ArchivedTask, 0)
	for rows.Next() {
		var task ArchivedTask
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Note,
			&task.Due,
			&task.Priority,
			&task.Status,
			&task.ArchivedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, map[string]any{"tasks": tasks})
}

// handleRestoreTask moves a task from archived_tasks to tasks.
func (a *App) handleRestoreTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	now := time.Now().UnixMilli()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	var task Task
	err = tx.QueryRow(r.Context(),
		`DELETE FROM archived_tasks WHERE id=$1
		 RETURNING id, title, note, due, priority, status`, id).
		Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := validateTaskPayload(&task, true, true); err != nil {
		writeError(w, http.StatusInternalServerError, "stored task is invalid")
		return
	}

	task.LastUpdated = now

	var maxOrd int
	if err := tx.QueryRow(r.Context(),
		`SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE status=$1`, task.Status).Scan(&maxOrd); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	task.SortOrder = maxOrd + 1

	if _, err := tx.Exec(r.Context(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
}

func (a *App) handleDeleteArchived(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	tag, err := a.db.Exec(r.Context(),
		`DELETE FROM archived_tasks WHERE id=$1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func jsonResp(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": message}); err != nil {
		log.Printf("encode error response: %v", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return false
	}

	return true
}

func setSessionCookie(w http.ResponseWriter, token string, expiry time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Expires:  expiry,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func clearSessionCookie(w http.ResponseWriter, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
		Path:     "/",
	})
}

func validateTaskPayload(task *Task, requireID bool, allowStatus bool) error {
	if requireID {
		task.ID = strings.TrimSpace(task.ID)
		if task.ID == "" {
			return errors.New("id is required")
		}
	}

	task.Title = strings.TrimSpace(task.Title)
	task.Note = strings.TrimSpace(task.Note)

	if task.Title == "" {
		return errors.New("title is required")
	}
	if len(task.Title) > maxTitleLength {
		return errors.New("title is too long")
	}
	if len(task.Note) > maxNoteLength {
		return errors.New("note is too long")
	}
	if !validDueDate(task.Due) {
		return errors.New("due must be in YYYY-MM-DD format")
	}

	if task.Priority == "" {
		task.Priority = "medium"
	}
	if !validPriority(task.Priority) {
		return errors.New("invalid priority")
	}

	if allowStatus {
		if !validStatus(task.Status) {
			return errors.New("invalid status")
		}
	} else {
		task.Status = "queued"
	}

	return nil
}

func (a *App) verifyLoginPassword(ctx context.Context, given string) (bool, error) {
	if a.passwordVerifier != nil {
		return a.passwordVerifier(ctx, given)
	}
	return a.verifyBootstrapPassword(ctx, given)
}

func (a *App) verifyBootstrapPassword(ctx context.Context, given string) (bool, error) {
	var hash string
	if err := a.db.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username=$1`, bootstrapAdmin).Scan(&hash); err != nil {
		return false, err
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(given)) == nil, nil
}

func (a *App) getActiveSession(ctx context.Context, token string) (sessionState, error) {
	if a.sessionGetter != nil {
		return a.sessionGetter(ctx, token)
	}

	var session sessionState
	var expiresAt int64
	if err := a.db.QueryRow(ctx, `
		SELECT username, expires_at
		FROM sessions
		WHERE token=$1 AND revoked_at IS NULL AND expires_at > $2
	`, token, time.Now().UnixMilli()).Scan(&session.Username, &expiresAt); err != nil {
		return sessionState{}, err
	}

	session.Token = token
	session.ExpiresAt = time.UnixMilli(expiresAt)
	return session, nil
}

func (a *App) createSession(ctx context.Context, token, username, clientIP string, expiresAt time.Time) error {
	if a.sessionCreator != nil {
		return a.sessionCreator(ctx, token, username, clientIP, expiresAt)
	}

	now := time.Now().UnixMilli()
	_, err := a.db.Exec(ctx, `
		INSERT INTO sessions (token, username, created_at, expires_at, revoked_at, last_seen_at, client_ip)
		VALUES ($1, $2, $3, $4, NULL, $3, $5)
	`, token, username, now, expiresAt.UnixMilli(), clientIP)
	return err
}

func (a *App) deleteSession(ctx context.Context, token string) error {
	if a.sessionDeleter != nil {
		return a.sessionDeleter(ctx, token)
	}

	_, err := a.db.Exec(ctx, `DELETE FROM sessions WHERE token=$1`, token)
	return err
}

func (a *App) recordAuthEvent(ctx context.Context, event authAuditEvent) {
	event.CreatedAt = time.Now().UnixMilli()
	if a.auditRecorder != nil {
		if err := a.auditRecorder(ctx, event); err != nil {
			log.Printf("auth audit recorder error: %v", err)
		}
		return
	}
	if a.db == nil {
		return
	}

	if _, err := a.db.Exec(ctx, `
		INSERT INTO auth_audit_logs (username, event_type, outcome, client_ip, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, event.Username, event.EventType, event.Outcome, event.ClientIP, event.Details, event.CreatedAt); err != nil {
		log.Printf("auth audit insert error: %v", err)
	}
}

func sessionFromContext(ctx context.Context) (sessionState, bool) {
	session, ok := ctx.Value(sessionContextKey).(sessionState)
	return session, ok
}

func clientIDFromRequest(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func (a *App) allowLoginAttempt(clientID string) bool {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	return !now.Before(state.BlockedUntil)
}

func (a *App) recordFailedLogin(clientID string) {
	now := time.Now()
	a.loginMu.Lock()
	defer a.loginMu.Unlock()

	state := a.loginAttempts[clientID]
	if state.WindowStart.IsZero() || now.Sub(state.WindowStart) > loginWindow {
		state = loginAttemptState{WindowStart: now}
	}

	state.Failures++
	if state.Failures >= maxLoginFailures {
		state.BlockedUntil = now.Add(loginBlockDuration)
		state.Failures = 0
		state.WindowStart = now
	}

	a.loginAttempts[clientID] = state
}

func (a *App) clearLoginAttempts(clientID string) {
	a.loginMu.Lock()
	defer a.loginMu.Unlock()
	delete(a.loginAttempts, clientID)
}

func newToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("generate token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("required env var %s is not set", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func validPriority(priority string) bool {
	return priority == "critical" || priority == "high" || priority == "medium"
}

func validStatus(status string) bool {
	return status == "queued" || status == "active" || status == "done"
}

func validDueDate(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
