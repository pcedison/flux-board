package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed static
var staticFiles embed.FS

const (
	archiveRetention = 3 * 24 * time.Hour
	sessionDuration  = 30 * 24 * time.Hour
	cookieName       = "flux_session"
)

// Task represents an active task in the kanban board.
// JSON tags match what the frontend expects (camelCase for time fields).
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Note        string `json:"note"`
	Due         string `json:"due"` // YYYY-MM-DD
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	LastUpdated int64  `json:"lastUpdated"` // Unix milliseconds
}

// ArchivedTask represents a soft-deleted task.
type ArchivedTask struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Note       string `json:"note"`
	Due        string `json:"due"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	ArchivedAt int64  `json:"archivedAt"` // Unix milliseconds
}

// App holds shared application dependencies.
type App struct {
	db           *pgxpool.Pool
	sessions     sync.Map // token(string) → expiry(time.Time)
	password     string
	cookieSecure bool
}

func main() {
	dbURL := mustEnv("DATABASE_URL")
	password := mustEnv("APP_PASSWORD")
	port := getEnv("PORT", "8080")
	// Set APP_ENV=development to disable Secure cookie (for local HTTP testing)
	cookieSecure := getEnv("APP_ENV", "production") != "development"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	app := &App{
		db:           pool,
		password:     password,
		cookieSecure: cookieSecure,
	}

	if err := app.initSchema(); err != nil {
		log.Fatalf("init schema: %v", err)
	}

	// Background goroutine: purge expired archived tasks every hour
	go app.archiveCleanupLoop()

	mux := http.NewServeMux()

	// Auth endpoints (public)
	mux.HandleFunc("POST /api/auth/login", app.handleLogin)
	mux.HandleFunc("POST /api/auth/logout", app.handleLogout)

	// Task endpoints (require valid session cookie)
	mux.HandleFunc("GET /api/tasks", app.auth(app.handleGetTasks))
	mux.HandleFunc("POST /api/tasks", app.auth(app.handleCreateTask))
	mux.HandleFunc("PUT /api/tasks/{id}", app.auth(app.handleUpdateTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", app.auth(app.handleArchiveTask))

	// Archive endpoints (require valid session cookie)
	mux.HandleFunc("GET /api/archived", app.auth(app.handleGetArchived))
	mux.HandleFunc("POST /api/archived/{id}/restore", app.auth(app.handleRestoreTask))
	mux.HandleFunc("DELETE /api/archived/{id}", app.auth(app.handleDeleteArchived))

	// Serve embedded static files (index.html, etc.)
	stripped, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(stripped)))

	log.Printf("Flux Board listening on :%s  (cookieSecure=%v)", port, cookieSecure)
	log.Fatal(http.ListenAndServe(":"+port, mux))
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
	`)
	return err
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

// ─── Auth ─────────────────────────────────────────────────────────────────────

// auth is a middleware that validates the session cookie before calling next.
func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		exp, ok := a.sessions.Load(c.Value)
		if !ok || time.Now().After(exp.(time.Time)) {
			a.sessions.Delete(c.Value)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if body.Password != a.password {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	token := newToken()
	exp := time.Now().Add(sessionDuration)
	a.sessions.Store(token, exp)

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Expires:  exp,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   a.cookieSecure,
		Path:     "/",
	})
	w.WriteHeader(http.StatusOK)
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(cookieName); err == nil {
		a.sessions.Delete(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, MaxAge: -1, Path: "/"})
	w.WriteHeader(http.StatusOK)
}

// ─── Tasks ────────────────────────────────────────────────────────────────────

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(context.Background(),
		`SELECT id, title, note, due, priority, status, sort_order, last_updated
		 FROM tasks
		 ORDER BY status, sort_order, last_updated DESC`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Note, &t.Due,
			&t.Priority, &t.Status, &t.SortOrder, &t.LastUpdated); err == nil {
			tasks = append(tasks, t)
		}
	}
	jsonResp(w, map[string]any{"tasks": tasks})
}

func (a *App) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil || t.ID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validPriority(t.Priority) {
		t.Priority = "medium"
	}
	t.Status = "queued"
	t.LastUpdated = time.Now().UnixMilli()

	// Place at end of queued lane
	var maxOrd int
	_ = a.db.QueryRow(context.Background(),
		`SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE status = 'queued'`).Scan(&maxOrd)
	t.SortOrder = maxOrd + 1

	_, err := a.db.Exec(context.Background(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO NOTHING`,
		t.ID, t.Title, t.Note, t.Due, t.Priority, t.Status, t.SortOrder, t.LastUpdated)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResp(w, t)
}

func (a *App) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if !validPriority(t.Priority) {
		t.Priority = "medium"
	}
	if !validStatus(t.Status) {
		t.Status = "queued"
	}
	t.LastUpdated = time.Now().UnixMilli()

	tag, err := a.db.Exec(context.Background(),
		`UPDATE tasks
		 SET title=$1, note=$2, due=$3, priority=$4, status=$5, sort_order=$6, last_updated=$7
		 WHERE id=$8`,
		t.Title, t.Note, t.Due, t.Priority, t.Status, t.SortOrder, t.LastUpdated, id)
	if err != nil || tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	t.ID = id
	jsonResp(w, t)
}

// handleArchiveTask moves a task from tasks → archived_tasks (soft delete).
func (a *App) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UnixMilli()

	tx, err := a.db.Begin(context.Background())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	var t ArchivedTask
	err = tx.QueryRow(context.Background(),
		`DELETE FROM tasks WHERE id=$1
		 RETURNING id, title, note, due, priority, status`, id).
		Scan(&t.ID, &t.Title, &t.Note, &t.Due, &t.Priority, &t.Status)
	if err == pgx.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	t.ArchivedAt = now
	if _, err = tx.Exec(context.Background(),
		`INSERT INTO archived_tasks (id, title, note, due, priority, status, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		t.ID, t.Title, t.Note, t.Due, t.Priority, t.Status, t.ArchivedAt); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(context.Background()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonResp(w, t)
}

// ─── Archived ─────────────────────────────────────────────────────────────────

func (a *App) handleGetArchived(w http.ResponseWriter, r *http.Request) {
	// Inline purge on every GET so the list is always fresh
	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
	_, _ = a.db.Exec(context.Background(),
		`DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff)

	rows, err := a.db.Query(context.Background(),
		`SELECT id, title, note, due, priority, status, archived_at
		 FROM archived_tasks
		 ORDER BY archived_at DESC`)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tasks := make([]ArchivedTask, 0)
	for rows.Next() {
		var t ArchivedTask
		if err := rows.Scan(&t.ID, &t.Title, &t.Note, &t.Due,
			&t.Priority, &t.Status, &t.ArchivedAt); err == nil {
			tasks = append(tasks, t)
		}
	}
	jsonResp(w, map[string]any{"tasks": tasks})
}

// handleRestoreTask moves a task from archived_tasks → tasks.
func (a *App) handleRestoreTask(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	now := time.Now().UnixMilli()

	tx, err := a.db.Begin(context.Background())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(context.Background())

	var t Task
	err = tx.QueryRow(context.Background(),
		`DELETE FROM archived_tasks WHERE id=$1
		 RETURNING id, title, note, due, priority, status`, id).
		Scan(&t.ID, &t.Title, &t.Note, &t.Due, &t.Priority, &t.Status)
	if err == pgx.ErrNoRows {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	t.LastUpdated = now

	var maxOrd int
	_ = tx.QueryRow(context.Background(),
		`SELECT COALESCE(MAX(sort_order), 0) FROM tasks WHERE status=$1`, t.Status).Scan(&maxOrd)
	t.SortOrder = maxOrd + 1

	if _, err = tx.Exec(context.Background(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Title, t.Note, t.Due, t.Priority, t.Status, t.SortOrder, t.LastUpdated); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(context.Background()); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonResp(w, t)
}

func (a *App) handleDeleteArchived(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tag, err := a.db.Exec(context.Background(),
		`DELETE FROM archived_tasks WHERE id=$1`, id)
	if err != nil || tag.RowsAffected() == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func jsonResp(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func newToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func mustEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("required env var %s is not set", k)
	}
	return v
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func validPriority(p string) bool { return p == "critical" || p == "high" || p == "medium" }
func validStatus(s string) bool   { return s == "queued" || s == "active" || s == "done" }
