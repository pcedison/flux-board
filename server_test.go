package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewMuxRegistersCoreRoutes(t *testing.T) {
	app := &App{
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, nil
		},
		readinessChecker: func(context.Context) error {
			return nil
		},
		auditRecorder: func(context.Context, authAuditEvent) error {
			return nil
		},
	}

	mux, err := newMux(app)
	if err != nil {
		t.Fatalf("newMux returned error: %v", err)
	}

	cases := []struct {
		name       string
		method     string
		target     string
		body       string
		wantStatus int
	}{
		{
			name:       "healthz route",
			method:     http.MethodGet,
			target:     "/healthz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "readyz route",
			method:     http.MethodGet,
			target:     "/readyz",
			wantStatus: http.StatusOK,
		},
		{
			name:       "login route",
			method:     http.MethodPost,
			target:     "/api/auth/login",
			body:       `{"password":"wrong"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "tasks route",
			method:     http.MethodGet,
			target:     "/api/tasks",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "reorder route",
			method:     http.MethodPost,
			target:     "/api/tasks/task-1/reorder",
			body:       `{"status":"queued"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "archived route",
			method:     http.MethodGet,
			target:     "/api/archived",
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, strings.NewReader(tc.body))
			req.RemoteAddr = "127.0.0.1:1234"
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
		})
	}
}

func TestNewMuxServesReactRuntimeRoot(t *testing.T) {
	app := &App{
		webRuntimeFS: newWebRuntimeFSTestHelper(t),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return true, nil
		},
		readinessChecker: func(context.Context) error {
			return nil
		},
		sessionCreator: func(context.Context, string, string, string, time.Time) error {
			return nil
		},
		auditRecorder: func(context.Context, authAuditEvent) error {
			return nil
		},
	}

	mux, err := newMux(app)
	if err != nil {
		t.Fatalf("newMux returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for root document, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Flux Board Web Runtime") {
		t.Fatalf("expected react runtime html, got %q", rec.Body.String())
	}
}

func TestHandleReadyzReturnsServiceUnavailableWhenDBIsUnavailable(t *testing.T) {
	app := &App{
		readinessChecker: func(context.Context) error {
			return context.DeadlineExceeded
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	app.handleReadyz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}
}

func TestHandleHealthzReturnsProbeContract(t *testing.T) {
	app := &App{}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	app.handleHealthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", cacheControl)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok body, got %+v", body)
	}
}

func TestHandleReadyzReturnsReadyProbeContract(t *testing.T) {
	app := &App{
		readinessChecker: func(context.Context) error {
			return nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	app.handleReadyz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if cacheControl := rec.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("expected Cache-Control no-store, got %q", cacheControl)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("expected status ready body, got %+v", body)
	}
}
