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

	"flux-board/internal/domain"

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
	if len(versions) != 3 ||
		versions[0] != "0001_initial" ||
		versions[1] != "0002_task_order_constraints" ||
		versions[2] != "0003_single_user_settings" {
		t.Fatalf("expected migration history [0001_initial 0002_task_order_constraints 0003_single_user_settings], got %+v", versions)
	}
	if len(checksums) != 3 || checksums[0] == "" || checksums[1] == "" || checksums[2] == "" {
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

	createTaskForIntegration(t, app, domain.Task{ID: "task-a", Title: "Task A", Due: "2026-04-20", Priority: "medium"})
	createTaskForIntegration(t, app, domain.Task{ID: "task-b", Title: "Task B", Due: "2026-04-21", Priority: "high"})
	createTaskForIntegration(t, app, domain.Task{ID: "task-c", Title: "Task C", Due: "2026-04-22", Priority: "critical"})

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

	createTaskForIntegration(t, app, domain.Task{ID: "task-a", Title: "Task A", Due: "2026-04-20", Priority: "medium"})
	createTaskForIntegration(t, app, domain.Task{ID: "task-b", Title: "Task B", Due: "2026-04-21", Priority: "medium"})

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

	var task domain.Task
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

func TestIntegrationSettingsUpdateExportAndImportWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration settings/export/import test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	handler := app.transportHandler()
	sessionCookie := loginForIntegration(t, app, "integration-secret", "127.0.0.1:4567")

	updateReq := httptest.NewRequest(http.MethodPatch, "/api/settings", strings.NewReader(`{"archiveRetentionDays":30}`))
	updateReq.RemoteAddr = "127.0.0.1:4567"
	updateReq.AddCookie(sessionCookie)
	updateRec := httptest.NewRecorder()
	handler.Auth(handler.HandleUpdateSettings)(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected settings update status 200, got %d body=%s", updateRec.Code, updateRec.Body.String())
	}

	var updatedSettings domain.AppSettings
	if err := json.NewDecoder(updateRec.Body).Decode(&updatedSettings); err != nil {
		t.Fatalf("decode settings update response: %v", err)
	}
	if updatedSettings.ArchiveRetentionDays == nil || *updatedSettings.ArchiveRetentionDays != 30 {
		t.Fatalf("expected archive retention 30 days, got %+v", updatedSettings)
	}

	var storedRetention string
	if err := app.db.QueryRow(context.Background(), `
		SELECT value
		FROM app_settings
		WHERE key='archive_retention_days'
	`).Scan(&storedRetention); err != nil {
		t.Fatalf("query updated archive retention: %v", err)
	}
	if storedRetention != "30" {
		t.Fatalf("expected stored archive retention value 30, got %q", storedRetention)
	}

	createTaskForIntegration(t, app, domain.Task{ID: "task-export-active", Title: "Task export active", Due: "2026-05-01", Priority: "medium"})
	createTaskForIntegration(t, app, domain.Task{ID: "task-export-archived", Title: "Task export archived", Due: "2026-05-02", Priority: "high"})

	archiveReq := httptest.NewRequest(http.MethodDelete, "/api/tasks/task-export-archived", nil)
	archiveReq.SetPathValue("id", "task-export-archived")
	archiveRec := httptest.NewRecorder()
	app.handleArchiveTask(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("expected archive-before-export status 200, got %d body=%s", archiveRec.Code, archiveRec.Body.String())
	}

	exportReq := httptest.NewRequest(http.MethodGet, "/api/export", nil)
	exportReq.RemoteAddr = "127.0.0.1:4567"
	exportReq.AddCookie(sessionCookie)
	exportRec := httptest.NewRecorder()
	handler.Auth(handler.HandleExport)(exportRec, exportReq)
	if exportRec.Code != http.StatusOK {
		t.Fatalf("expected export status 200, got %d body=%s", exportRec.Code, exportRec.Body.String())
	}
	if exportRec.Header().Get("Content-Disposition") != `attachment; filename="flux-board-export.json"` {
		t.Fatalf("expected export attachment header, got %q", exportRec.Header().Get("Content-Disposition"))
	}

	var exportBundle domain.ExportBundle
	if err := json.NewDecoder(exportRec.Body).Decode(&exportBundle); err != nil {
		t.Fatalf("decode export response: %v", err)
	}
	if exportBundle.Version == "" || exportBundle.ExportedAt <= 0 {
		t.Fatalf("expected export metadata to be populated, got %+v", exportBundle)
	}
	if exportBundle.Settings.ArchiveRetentionDays == nil || *exportBundle.Settings.ArchiveRetentionDays != 30 {
		t.Fatalf("expected exported settings to include retention 30 days, got %+v", exportBundle.Settings)
	}
	if len(exportBundle.Tasks) != 1 || exportBundle.Tasks[0].ID != "task-export-active" {
		t.Fatalf("expected exported active task snapshot, got %+v", exportBundle.Tasks)
	}
	if len(exportBundle.Archived) != 1 || exportBundle.Archived[0].ID != "task-export-archived" {
		t.Fatalf("expected exported archived task snapshot, got %+v", exportBundle.Archived)
	}

	importRetention := 14
	importBundle := domain.ExportBundle{
		Version:    "integration-import",
		ExportedAt: time.Now().UnixMilli(),
		Settings: domain.AppSettings{
			ArchiveRetentionDays: &importRetention,
		},
		Tasks: []domain.Task{
			{
				ID:          "queued-second",
				Title:       "Queued second",
				Note:        "",
				Due:         "2026-05-10",
				Priority:    "high",
				Status:      "queued",
				SortOrder:   9,
				LastUpdated: 100,
			},
			{
				ID:          "queued-first",
				Title:       "Queued first",
				Note:        "",
				Due:         "2026-05-09",
				Priority:    "medium",
				Status:      "queued",
				SortOrder:   2,
				LastUpdated: 200,
			},
			{
				ID:          "active-only",
				Title:       "Active only",
				Note:        "",
				Due:         "2026-05-11",
				Priority:    "critical",
				Status:      "active",
				SortOrder:   4,
				LastUpdated: 300,
			},
		},
		Archived: []domain.ArchivedTask{
			{
				ID:         "archived-imported",
				Title:      "Archived imported",
				Note:       "",
				Due:        "2026-05-12",
				Priority:   "medium",
				Status:     "done",
				SortOrder:  6,
				ArchivedAt: time.Now().UnixMilli(),
			},
		},
	}

	importReq := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(mustJSONForIntegration(t, importBundle)))
	importReq.RemoteAddr = "127.0.0.1:4567"
	importReq.AddCookie(sessionCookie)
	importRec := httptest.NewRecorder()
	handler.Auth(handler.HandleImport)(importRec, importReq)
	if importRec.Code != http.StatusNoContent {
		t.Fatalf("expected import status 204, got %d body=%s", importRec.Code, importRec.Body.String())
	}

	assertLaneOrder(t, app, "active", []string{"active-only"})
	assertLaneOrder(t, app, "queued", []string{"queued-first", "queued-second"})
	assertArchivedIDs(t, app, []string{"archived-imported"})

	var legacyActiveExists bool
	if err := app.db.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM tasks WHERE id='task-export-active')
	`).Scan(&legacyActiveExists); err != nil {
		t.Fatalf("check imported task replacement for active task: %v", err)
	}
	if legacyActiveExists {
		t.Fatal("expected import to replace previously exported active task")
	}

	var legacyArchivedExists bool
	if err := app.db.QueryRow(context.Background(), `
		SELECT EXISTS(SELECT 1 FROM archived_tasks WHERE id='task-export-archived')
	`).Scan(&legacyArchivedExists); err != nil {
		t.Fatalf("check imported task replacement for archived task: %v", err)
	}
	if legacyArchivedExists {
		t.Fatal("expected import to replace previously exported archived task")
	}

	if err := app.db.QueryRow(context.Background(), `
		SELECT value
		FROM app_settings
		WHERE key='archive_retention_days'
	`).Scan(&storedRetention); err != nil {
		t.Fatalf("query imported archive retention: %v", err)
	}
	if storedRetention != "14" {
		t.Fatalf("expected imported archive retention value 14, got %q", storedRetention)
	}
}

func TestIntegrationListAndRevokeSessionsWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration session list/revoke test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	handler := app.transportHandler()
	currentCookie := loginForIntegration(t, app, "integration-secret", "127.0.0.1:4567")
	otherCookie := loginForIntegration(t, app, "integration-secret", "127.0.0.2:4567")

	listReq := httptest.NewRequest(http.MethodGet, "/api/settings/sessions", nil)
	listReq.RemoteAddr = "127.0.0.1:4567"
	listReq.AddCookie(currentCookie)
	listRec := httptest.NewRecorder()
	handler.Auth(handler.HandleGetSessions)(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list sessions status 200, got %d body=%s", listRec.Code, listRec.Body.String())
	}

	var listBody struct {
		Sessions []domain.SessionInfo `json:"sessions"`
	}
	if err := json.NewDecoder(listRec.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list sessions response: %v", err)
	}
	if len(listBody.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %+v", listBody.Sessions)
	}

	currentCount := 0
	foundOther := false
	for _, session := range listBody.Sessions {
		if session.Token == currentCookie.Value {
			if !session.Current {
				t.Fatalf("expected current session to be marked current, got %+v", session)
			}
			if session.ClientIP != "127.0.0.1" {
				t.Fatalf("expected current session client IP 127.0.0.1, got %+v", session)
			}
			currentCount++
		}
		if session.Token == otherCookie.Value {
			foundOther = true
			if session.Current {
				t.Fatalf("expected non-current session to be marked not current, got %+v", session)
			}
			if session.ClientIP != "127.0.0.2" {
				t.Fatalf("expected other session client IP 127.0.0.2, got %+v", session)
			}
		}
	}
	if currentCount != 1 {
		t.Fatalf("expected exactly one current session, got %+v", listBody.Sessions)
	}
	if !foundOther {
		t.Fatalf("expected other session %q in %+v", otherCookie.Value, listBody.Sessions)
	}

	revokeReq := httptest.NewRequest(http.MethodDelete, "/api/settings/sessions/"+otherCookie.Value, nil)
	revokeReq.RemoteAddr = "127.0.0.1:4567"
	revokeReq.SetPathValue("token", otherCookie.Value)
	revokeReq.AddCookie(currentCookie)
	revokeRec := httptest.NewRecorder()
	handler.Auth(handler.HandleDeleteSession)(revokeRec, revokeReq)
	if revokeRec.Code != http.StatusNoContent {
		t.Fatalf("expected revoke session status 204, got %d body=%s", revokeRec.Code, revokeRec.Body.String())
	}

	var sessionCount int
	if err := app.db.QueryRow(context.Background(), `SELECT COUNT(*) FROM sessions`).Scan(&sessionCount); err != nil {
		t.Fatalf("count sessions after revoke: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("expected exactly one session after revoke, got %d", sessionCount)
	}

	meHandler := handler.Auth(handler.HandleGetSession)

	revokedReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	revokedReq.RemoteAddr = "127.0.0.2:4567"
	revokedReq.AddCookie(otherCookie)
	revokedRec := httptest.NewRecorder()
	meHandler(revokedRec, revokedReq)
	if revokedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked session to be unauthorized, got %d body=%s", revokedRec.Code, revokedRec.Body.String())
	}

	currentReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	currentReq.RemoteAddr = "127.0.0.1:4567"
	currentReq.AddCookie(currentCookie)
	currentRec := httptest.NewRecorder()
	meHandler(currentRec, currentReq)
	if currentRec.Code != http.StatusOK {
		t.Fatalf("expected current session to stay valid, got %d body=%s", currentRec.Code, currentRec.Body.String())
	}

	var eventType, outcome, clientIP, details string
	if err := app.db.QueryRow(context.Background(), `
		SELECT event_type, outcome, client_ip, details
		FROM auth_audit_logs
		WHERE event_type='session_revoke'
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&eventType, &outcome, &clientIP, &details); err != nil {
		t.Fatalf("query session revoke audit log: %v", err)
	}
	if eventType != "session_revoke" || outcome != "success" || clientIP != "127.0.0.1" || !strings.Contains(details, otherCookie.Value) {
		t.Fatalf("expected successful session revoke audit event, got type=%q outcome=%q clientIP=%q details=%q", eventType, outcome, clientIP, details)
	}
}

func TestIntegrationChangePasswordRevokesOtherSessionsWithDatabase(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration password rotation test")
	}

	app, cleanup := newIntegrationTestApp(t, databaseURL)
	defer cleanup()

	handler := app.transportHandler()
	currentCookie := loginForIntegration(t, app, "integration-secret", "127.0.0.1:4567")
	otherCookie := loginForIntegration(t, app, "integration-secret", "127.0.0.2:4567")

	changeReq := httptest.NewRequest(http.MethodPost, "/api/settings/password", strings.NewReader(`{
		"currentPassword":"integration-secret",
		"newPassword":"integration-secret-rotated"
	}`))
	changeReq.RemoteAddr = "127.0.0.1:4567"
	changeReq.AddCookie(currentCookie)
	changeRec := httptest.NewRecorder()
	handler.Auth(handler.HandleChangePassword)(changeRec, changeReq)
	if changeRec.Code != http.StatusNoContent {
		t.Fatalf("expected change password status 204, got %d body=%s", changeRec.Code, changeRec.Body.String())
	}

	rows, err := app.db.Query(context.Background(), `
		SELECT token
		FROM sessions
		ORDER BY token ASC
	`)
	if err != nil {
		t.Fatalf("query sessions after password change: %v", err)
	}
	defer rows.Close()

	var remainingTokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			t.Fatalf("scan session after password change: %v", err)
		}
		remainingTokens = append(remainingTokens, token)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sessions after password change: %v", err)
	}
	if len(remainingTokens) != 1 || remainingTokens[0] != currentCookie.Value {
		t.Fatalf("expected only the current session to remain after password change, got %+v", remainingTokens)
	}

	meHandler := handler.Auth(handler.HandleGetSession)

	currentReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	currentReq.RemoteAddr = "127.0.0.1:4567"
	currentReq.AddCookie(currentCookie)
	currentRec := httptest.NewRecorder()
	meHandler(currentRec, currentReq)
	if currentRec.Code != http.StatusOK {
		t.Fatalf("expected current session to remain valid after password change, got %d body=%s", currentRec.Code, currentRec.Body.String())
	}

	otherReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	otherReq.RemoteAddr = "127.0.0.2:4567"
	otherReq.AddCookie(otherCookie)
	otherRec := httptest.NewRecorder()
	meHandler(otherRec, otherReq)
	if otherRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected other session to be revoked after password change, got %d body=%s", otherRec.Code, otherRec.Body.String())
	}

	oldLoginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"integration-secret"}`))
	oldLoginReq.RemoteAddr = "127.0.0.3:4567"
	oldLoginRec := httptest.NewRecorder()
	app.handleLogin(oldLoginRec, oldLoginReq)
	if oldLoginRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected old password login status 401 after rotation, got %d body=%s", oldLoginRec.Code, oldLoginRec.Body.String())
	}

	newCookie := loginForIntegration(t, app, "integration-secret-rotated", "127.0.0.3:4567")
	if newCookie == nil || newCookie.Value == "" {
		t.Fatal("expected rotated password login to issue a session cookie")
	}

	var outcome, clientIP string
	if err := app.db.QueryRow(context.Background(), `
		SELECT outcome, client_ip
		FROM auth_audit_logs
		WHERE event_type='password_change'
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&outcome, &clientIP); err != nil {
		t.Fatalf("query password change audit log: %v", err)
	}
	if outcome != "success" || clientIP != "127.0.0.1" {
		t.Fatalf("expected successful password change audit event, got outcome=%q clientIP=%q", outcome, clientIP)
	}
}

func createTaskForIntegration(t *testing.T, app *App, task domain.Task) {
	t.Helper()

	body := fmt.Sprintf(`{"id":"%s","title":"%s","note":"","due":"%s","priority":"%s"}`, task.ID, task.Title, task.Due, task.Priority)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(body))
	rec := httptest.NewRecorder()
	app.handleCreateTask(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func loginForIntegration(t *testing.T, app *App, password, remoteAddr string) *http.Cookie {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(fmt.Sprintf(`{"password":"%s"}`, password)))
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	app.handleLogin(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected login to issue a session cookie")
	}
	return cookies[0]
}

func mustJSONForIntegration(t *testing.T, value any) string {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal integration payload: %v", err)
	}
	return string(payload)
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

func assertArchivedIDs(t *testing.T, app *App, expectedIDs []string) {
	t.Helper()

	rows, err := app.db.Query(context.Background(), `
		SELECT id
		FROM archived_tasks
		ORDER BY archived_at DESC, id ASC
	`)
	if err != nil {
		t.Fatalf("query archived ids: %v", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan archived ids: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate archived ids: %v", err)
	}
	if len(ids) != len(expectedIDs) {
		t.Fatalf("expected archived ids %+v, got %+v", expectedIDs, ids)
	}
	for i := range expectedIDs {
		if ids[i] != expectedIDs[i] {
			t.Fatalf("expected archived ids %+v, got %+v", expectedIDs, ids)
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
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("parse test pool config: %v", err)
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = map[string]string{}
	}
	config.ConnConfig.RuntimeParams["search_path"] = schemaName

	testPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
		adminPool.Close()
		t.Fatalf("connect test pool: %v", err)
	}

	app := &App{
		db:                testPool,
		bootstrapPassword: "integration-secret",
		cookieSecure:      false,
	}
	if err := app.initSchema(); err != nil {
		testPool.Close()
		_, _ = adminPool.Exec(ctx, `DROP SCHEMA `+schemaName+` CASCADE`)
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
