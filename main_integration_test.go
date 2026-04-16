package main

import (
	"context"
	"encoding/json"
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

func newIntegrationTestApp(t *testing.T, databaseURL string) (*App, func()) {
	t.Helper()

	ctx := context.Background()
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect admin pool: %v", err)
	}

	schemaName := "itest_auth_" + time.Now().UTC().Format("20060102150405")
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
