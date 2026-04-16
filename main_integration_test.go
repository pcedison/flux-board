package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegrationAuthFlowWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration auth flow test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	badLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"wrong-password"}`))
	badLoginReq.RemoteAddr = "127.0.0.1:4567"
	badLoginRec := httptest.NewRecorder()
	app.handleLogin(badLoginRec, badLoginReq)
	if badLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected bad login status 401, got %d", badLoginRec.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"integration-secret"}`))
	loginReq.RemoteAddr = "127.0.0.1:4567"
	loginRec := httptest.NewRecorder()
	app.handleLogin(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRec.Code)
	}

	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login to issue a session cookie")
	}

	meHandler := app.auth(app.handleGetSession)
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.RemoteAddr = "127.0.0.1:4567"
	meReq.AddCookie(cookies[0])
	meRec := httptest.NewRecorder()
	meHandler(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("expected /api/auth/me status 200, got %d", meRec.Code)
	}

	var meBody struct {
		Authenticated bool  `json:"authenticated"`
		ExpiresAt     int64 `json:"expiresAt"`
	}
	if err := json.NewDecoder(meRec.Body).Decode(&meBody); err != nil {
		t.Fatalf("decode /api/auth/me response: %v", err)
	}
	if !meBody.Authenticated || meBody.ExpiresAt <= 0 {
		t.Fatalf("unexpected /api/auth/me body: %+v", meBody)
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.RemoteAddr = "127.0.0.1:4567"
	logoutReq.AddCookie(cookies[0])
	logoutRec := httptest.NewRecorder()
	app.handleLogout(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("expected logout status 200, got %d", logoutRec.Code)
	}

	postLogoutReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	postLogoutReq.RemoteAddr = "127.0.0.1:4567"
	postLogoutReq.AddCookie(cookies[0])
	postLogoutRec := httptest.NewRecorder()
	meHandler(postLogoutRec, postLogoutReq)
	if postLogoutRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected post-logout /api/auth/me status 401, got %d", postLogoutRec.Code)
	}

	var events []struct {
		EventType string
		Outcome   string
	}
	rows, err := app.db.Query(context.Background(), `
		SELECT event_type, outcome
		FROM auth_audit_logs
		ORDER BY id ASC
	`)
	if err != nil {
		t.Fatalf("query auth_audit_logs: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event struct {
			EventType string
			Outcome   string
		}
		if err := rows.Scan(&event.EventType, &event.Outcome); err != nil {
			t.Fatalf("scan auth_audit_logs: %v", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate auth_audit_logs: %v", err)
	}

	if len(events) < 4 {
		t.Fatalf("expected at least 4 auth audit events, got %+v", events)
	}
	if events[0].EventType != "login" || events[0].Outcome != "failed" {
		t.Fatalf("expected first event to be failed login, got %+v", events[0])
	}
	if events[1].EventType != "login" || events[1].Outcome != "success" {
		t.Fatalf("expected second event to be successful login, got %+v", events[1])
	}
	if events[2].EventType != "logout" || events[2].Outcome != "success" {
		t.Fatalf("expected third event to be successful logout, got %+v", events[2])
	}
	if events[3].EventType != "session" || events[3].Outcome != "invalid" {
		t.Fatalf("expected fourth event to be invalid session, got %+v", events[3])
	}
}

func TestIntegrationHealthAndReadinessWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration probe test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	app.handleHealthz(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("expected /healthz status 200, got %d", healthRec.Code)
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyRec := httptest.NewRecorder()
	app.handleReadyz(readyRec, readyReq)
	if readyRec.Code != http.StatusOK {
		t.Fatalf("expected /readyz status 200, got %d body=%s", readyRec.Code, readyRec.Body.String())
	}
}

func TestIntegrationInitSchemaAppliesMigrations(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration migration test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	if err := app.initSchema(); err != nil {
		t.Fatalf("re-run initSchema: %v", err)
	}

	rows, err := app.db.Query(context.Background(), `
		SELECT version, checksum
		FROM schema_migrations
		ORDER BY version ASC
	`)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer rows.Close()

	var versions []string
	var checksums []string
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			t.Fatalf("scan schema_migrations: %v", err)
		}
		versions = append(versions, version)
		checksums = append(checksums, checksum)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate schema_migrations: %v", err)
	}
	if len(versions) != 2 || versions[0] != "0001_initial" || versions[1] != "0002_task_order_constraints" {
		t.Fatalf("expected migration history [0001_initial], got %+v", versions)
	}
	if len(checksums) != 2 || checksums[0] == "" || checksums[1] == "" {
		t.Fatalf("expected non-empty migration checksum, got %+v", checksums)
	}

	for _, objectName := range requiredSchemaObjects {
		var resolved string
		if err := app.db.QueryRow(context.Background(),
			`SELECT COALESCE(to_regclass($1)::text, '')`,
			objectName,
		).Scan(&resolved); err != nil {
			t.Fatalf("check schema object %s: %v", objectName, err)
		}
		if resolved == "" {
			t.Fatalf("expected schema object %s to exist after initSchema", objectName)
		}
	}

	for _, constraintName := range requiredSchemaConstraints {
		var exists bool
		if err := app.db.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1
				FROM pg_constraint c
				JOIN pg_class r ON r.oid = c.conrelid
				JOIN pg_namespace n ON n.oid = r.relnamespace
				WHERE c.conname = $1
				  AND n.nspname = current_schema()
			)
		`, constraintName).Scan(&exists); err != nil {
			t.Fatalf("check schema constraint %s: %v", constraintName, err)
		}
		if !exists {
			t.Fatalf("expected schema constraint %s to exist after initSchema", constraintName)
		}
	}
}

func TestIntegrationTaskReorderAndArchiveRestore(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration reorder test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	createTaskForIntegration(t, app, Task{ID: "task-a", Title: "Task A", Due: "2026-04-20", Priority: "medium"})
	createTaskForIntegration(t, app, Task{ID: "task-b", Title: "Task B", Due: "2026-04-21", Priority: "high"})
	createTaskForIntegration(t, app, Task{ID: "task-c", Title: "Task C", Due: "2026-04-22", Priority: "critical"})

	reorderReq := httptest.NewRequest(http.MethodPost, "/api/tasks/task-c/reorder", strings.NewReader(`{"status":"queued","anchorTaskId":"task-b","placeAfter":false}`))
	reorderReq.SetPathValue("id", "task-c")
	reorderRec := httptest.NewRecorder()
	app.handleReorderTask(reorderRec, reorderReq)
	if reorderRec.Code != http.StatusOK {
		t.Fatalf("expected reorder status 200, got %d body=%s", reorderRec.Code, reorderRec.Body.String())
	}

	assertLaneOrder(t, app, "queued", []string{"task-a", "task-c", "task-b"})

	archiveReq := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-c", nil)
	archiveReq.SetPathValue("id", "task-c")
	archiveRec := httptest.NewRecorder()
	app.handleArchiveTask(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("expected archive status 200, got %d body=%s", archiveRec.Code, archiveRec.Body.String())
	}

	assertLaneOrder(t, app, "queued", []string{"task-a", "task-b"})

	var archivedOrder int
	if err := app.db.QueryRow(context.Background(), `SELECT sort_order FROM archived_tasks WHERE id='task-c'`).Scan(&archivedOrder); err != nil {
		t.Fatalf("query archived sort order: %v", err)
	}
	if archivedOrder != 1 {
		t.Fatalf("expected archived task sort order 1, got %d", archivedOrder)
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/archived/task-c/restore", nil)
	restoreReq.SetPathValue("id", "task-c")
	restoreRec := httptest.NewRecorder()
	app.handleRestoreTask(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d body=%s", restoreRec.Code, restoreRec.Body.String())
	}

	assertLaneOrder(t, app, "queued", []string{"task-a", "task-c", "task-b"})
}

func TestIntegrationUpdateTaskDoesNotClobberOrdering(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration update test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	createTaskForIntegration(t, app, Task{ID: "task-a", Title: "Task A", Due: "2026-04-20", Priority: "medium"})
	createTaskForIntegration(t, app, Task{ID: "task-b", Title: "Task B", Due: "2026-04-21", Priority: "medium"})

	reorderReq := httptest.NewRequest(http.MethodPost, "/api/tasks/task-b/reorder", strings.NewReader(`{"status":"active"}`))
	reorderReq.SetPathValue("id", "task-b")
	reorderRec := httptest.NewRecorder()
	app.handleReorderTask(reorderRec, reorderReq)
	if reorderRec.Code != http.StatusOK {
		t.Fatalf("expected reorder-to-active status 200, got %d body=%s", reorderRec.Code, reorderRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/api/tasks/task-b", strings.NewReader(`{
		"title":"Task B updated",
		"note":"changed",
		"due":"2026-04-30",
		"priority":"high",
		"status":"queued",
		"sort_order":99
	}`))
	updateReq.SetPathValue("id", "task-b")
	updateRec := httptest.NewRecorder()
	app.handleUpdateTask(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update status 200, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var task Task
	if err := json.NewDecoder(updateRec.Body).Decode(&task); err != nil {
		t.Fatalf("decode update response: %v", err)
	}
	if task.Status != "active" || task.SortOrder != 0 {
		t.Fatalf("expected preserved ordering metadata, got %+v", task)
	}
	if task.Title != "Task B updated" || task.Priority != "high" || task.Note != "changed" || task.Due != "2026-04-30" {
		t.Fatalf("expected content update to persist, got %+v", task)
	}

	assertLaneOrder(t, app, "queued", []string{"task-a"})
	assertLaneOrder(t, app, "active", []string{"task-b"})
}

func createTaskForIntegration(t *testing.T, app *App, task Task) {
	t.Helper()

	body := fmt.Sprintf(`{"id":"%s","title":"%s","note":"","due":"%s","priority":"%s"}`, task.ID, task.Title, task.Due, task.Priority)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
	rec := httptest.NewRecorder()
	app.handleCreateTask(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func assertLaneOrder(t *testing.T, app *App, status string, expectedIDs []string) {
	t.Helper()

	rows, err := app.db.Query(context.Background(), `
		SELECT id, sort_order
		FROM tasks
		WHERE status=$1
		ORDER BY sort_order ASC, last_updated DESC, id ASC
	`, status)
	if err != nil {
		t.Fatalf("query lane order for %s: %v", status, err)
	}
	defer rows.Close()

	var ids []string
	idx := 0
	for rows.Next() {
		var id string
		var sortOrder int
		if err := rows.Scan(&id, &sortOrder); err != nil {
			t.Fatalf("scan lane order for %s: %v", status, err)
		}
		if sortOrder != idx {
			t.Fatalf("expected contiguous sort order for %s at %d, got %d", status, idx, sortOrder)
		}
		ids = append(ids, id)
		idx++
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate lane order for %s: %v", status, err)
	}
	if len(ids) != len(expectedIDs) {
		t.Fatalf("expected lane %s ids %+v, got %+v", status, expectedIDs, ids)
	}
	for i := range expectedIDs {
		if ids[i] != expectedIDs[i] {
			t.Fatalf("expected lane %s ids %+v, got %+v", status, expectedIDs, ids)
		}
	}
}

func newIntegrationTestApp(t *testing.T, databaseURL string) (*App, func()) {
	t.Helper()

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}

	schemaName := fmt.Sprintf("itest_auth_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA `+schemaName); err != nil {
		adminPool.Close()
		t.Fatalf("create schema: %v", err)
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("parse test pool config: %v", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName

	testPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("connect test pool: %v", err)
	}

	app := &App{
		db:                testPool,
		bootstrapPassword: "integration-secret",
		cookieSecure:      false,
		loginAttempts:     make(map[string]loginAttemptState),
	}
	if err := app.initSchema(); err != nil {
		testPool.Close()
		adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("init schema: %v", err)
	}

	cleanup := func() {
		testPool.Close()
		if _, err := adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`); err != nil {
			t.Fatalf("drop schema: %v", err)
		}
		adminPool.Close()
	}

	return app, cleanup
}
