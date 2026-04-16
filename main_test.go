package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestValidateTaskPayloadDefaultsQueuedTask(t *testing.T) {
	task := Task{
		ID:    "task-1",
		Title: "  Ship MVP safely  ",
		Note:  "  Keep this trimmed  ",
		Due:   "2026-04-20",
	}

	if err := validateTaskPayload(&task, true, false); err != nil {
		t.Fatalf("validateTaskPayload returned error: %v", err)
	}

	if task.Title != "Ship MVP safely" {
		t.Fatalf("expected trimmed title, got %q", task.Title)
	}
	if task.Note != "Keep this trimmed" {
		t.Fatalf("expected trimmed note, got %q", task.Note)
	}
	if task.Priority != "medium" {
		t.Fatalf("expected default priority medium, got %q", task.Priority)
	}
	if task.Status != "queued" {
		t.Fatalf("expected default status queued, got %q", task.Status)
	}
}

func TestValidateTaskPayloadRejectsInvalidDueDate(t *testing.T) {
	task := Task{
		ID:       "task-2",
		Title:    "Broken due date",
		Note:     "",
		Due:      "20-04-2026",
		Priority: "high",
		Status:   "queued",
	}

	if err := validateTaskPayload(&task, true, true); err == nil {
		t.Fatal("expected invalid due date to fail validation")
	}
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"password":"secret","extra":true}`))
	rec := httptest.NewRecorder()

	var body struct {
		Password string `json:"password"`
	}

	if decodeJSON(rec, req, authBodyLimit, &body) {
		t.Fatal("expected decodeJSON to reject unknown fields")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestDecodeJSONRejectsMultipleObjects(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{"password":"secret"}{"another":true}`))
	rec := httptest.NewRecorder()

	var body struct {
		Password string `json:"password"`
	}

	if decodeJSON(rec, req, authBodyLimit, &body) {
		t.Fatal("expected decodeJSON to reject multiple JSON objects")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestDecodeJSONRejectsOversizedBody(t *testing.T) {
	payload := `{"password":"` + strings.Repeat("a", int(authBodyLimit)) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(payload))
	rec := httptest.NewRecorder()

	var body struct {
		Password string `json:"password"`
	}

	if decodeJSON(rec, req, authBodyLimit, &body) {
		t.Fatal("expected decodeJSON to reject oversized payload")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rec.Code)
	}
}

func TestLoginRateLimiterBlocksAfterThreshold(t *testing.T) {
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
	}
	clientID := "127.0.0.1"

	if !app.allowLoginAttempt(clientID) {
		t.Fatal("expected new client to be allowed")
	}

	for i := 0; i < maxLoginFailures; i++ {
		app.recordFailedLogin(clientID)
	}

	if app.allowLoginAttempt(clientID) {
		t.Fatal("expected client to be blocked after repeated failures")
	}

	app.clearLoginAttempts(clientID)
	if !app.allowLoginAttempt(clientID) {
		t.Fatal("expected cleared client to be allowed again")
	}
}

func TestClientIDFromRequestPrefersForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.RemoteAddr = "10.0.0.10:4321"
	req.Header.Set("X-Forwarded-For", "203.0.113.8, 10.0.0.10")

	if got := clientIDFromRequest(req); got != "203.0.113.8" {
		t.Fatalf("expected forwarded client IP, got %q", got)
	}
}

func TestSecurityHeadersAddsAPIHeaders(t *testing.T) {
	app := &App{}
	handler := app.securityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Fatalf("expected Referrer-Policy header, got %q", got)
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected no-store cache control for API route, got %q", got)
	}
}

func TestAuthRejectsMissingCookie(t *testing.T) {
	app := &App{}
	handler := app.auth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called without session cookie")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestAuthReturnsInternalServerErrorOnSessionLookupFailure(t *testing.T) {
	var events []authAuditEvent
	app := &App{
		sessionGetter: func(context.Context, string) (sessionState, error) {
			return sessionState{}, context.DeadlineExceeded
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}
	handler := app.auth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called when session lookup fails")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.AddCookie(&http.Cookie{Name: cookieName, Value: "dead-session"})
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
	if len(events) != 1 || events[0].EventType != "session" || events[0].Outcome != "error" {
		t.Fatalf("expected session error audit event, got %+v", events)
	}
	if setCookie := rec.Header().Values("Set-Cookie"); len(setCookie) != 0 {
		t.Fatalf("did not expect session cookie to be cleared on store failure, got %+v", setCookie)
	}
}

func TestHandleLoginReturnsTooManyRequestsWhenBlocked(t *testing.T) {
	var events []authAuditEvent
	app := &App{
		loginAttempts: map[string]loginAttemptState{
			"127.0.0.1": {BlockedUntil: time.Now().Add(5 * time.Minute)},
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	req.RemoteAddr = "127.0.0.1:4567"
	rec := httptest.NewRecorder()

	app.handleLogin(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", rec.Code)
	}
	if len(events) != 1 || events[0].Outcome != "blocked" {
		t.Fatalf("expected blocked audit event, got %+v", events)
	}
}

func TestHandleLoginReturnsInternalServerErrorWhenVerifierFails(t *testing.T) {
	var events []authAuditEvent
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, context.DeadlineExceeded
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	req.RemoteAddr = "127.0.0.1:4567"
	rec := httptest.NewRecorder()

	app.handleLogin(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
	if len(events) != 1 || events[0].Outcome != "error" {
		t.Fatalf("expected error audit event, got %+v", events)
	}
}

func TestHandleLoginReturnsUnauthorizedOnBadPassword(t *testing.T) {
	var events []authAuditEvent
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, nil
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"wrong"}`))
	req.RemoteAddr = "127.0.0.1:4567"
	rec := httptest.NewRecorder()

	app.handleLogin(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	state := app.loginAttempts["127.0.0.1"]
	if state.Failures != 1 {
		t.Fatalf("expected failed login counter to increment, got %+v", state)
	}
	if len(events) != 1 || events[0].Outcome != "failed" {
		t.Fatalf("expected failed audit event, got %+v", events)
	}
}

func TestObservedLoginAuditEventCarriesRequestID(t *testing.T) {
	var events []authAuditEvent
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, nil
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}

	server := newHTTPServer("8080", app.securityHeaders(http.HandlerFunc(app.handleLogin)))
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"wrong"}`))
	req.RemoteAddr = "127.0.0.1:4567"
	rec := httptest.NewRecorder()

	server.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
	if len(events) != 1 {
		t.Fatalf("expected one audit event, got %+v", events)
	}
	if got := rec.Header().Get(requestIDHeader); got == "" {
		t.Fatal("expected request id response header")
	} else if events[0].RequestID != got {
		t.Fatalf("expected audit event request id %q, got %q", got, events[0].RequestID)
	}
}

func TestAuthFlowLoginSessionLogout(t *testing.T) {
	sessions := make(map[string]sessionState)
	var events []authAuditEvent

	app := &App{
		cookieSecure:  false,
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return true, nil
		},
		sessionCreator: func(_ context.Context, token, username, clientIP string, expiresAt time.Time) error {
			sessions[token] = sessionState{
				Token:     token,
				Username:  username,
				ExpiresAt: expiresAt,
			}
			return nil
		},
		sessionGetter: func(_ context.Context, token string) (sessionState, error) {
			session, ok := sessions[token]
			if !ok {
				return sessionState{}, pgx.ErrNoRows
			}
			return session, nil
		},
		sessionDeleter: func(_ context.Context, token string) error {
			delete(sessions, token)
			return nil
		},
		auditRecorder: func(_ context.Context, event authAuditEvent) error {
			events = append(events, event)
			return nil
		},
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"password":"secret"}`))
	loginReq.RemoteAddr = "127.0.0.1:4567"
	loginRec := httptest.NewRecorder()
	app.handleLogin(loginRec, loginReq)

	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRec.Code)
	}
	cookies := loginRec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie on login")
	}

	meHandler := app.auth(app.handleGetSession)
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
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

	if len(events) < 3 {
		t.Fatalf("expected auth audit events, got %+v", events)
	}
	if events[0].EventType != "login" || events[0].Outcome != "success" {
		t.Fatalf("expected first audit event to be login success, got %+v", events[0])
	}
	if events[1].EventType != "logout" || events[1].Outcome != "success" {
		t.Fatalf("expected second audit event to be logout success, got %+v", events[1])
	}
	if events[2].EventType != "session" || events[2].Outcome != "invalid" {
		t.Fatalf("expected third audit event to be invalid session, got %+v", events[2])
	}
}
