package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewMuxRegistersCoreRoutes(t *testing.T) {
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, nil
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

func TestNewMuxServesEmbeddedFrontendRoot(t *testing.T) {
	app := &App{
		loginAttempts: make(map[string]loginAttemptState),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return true, nil
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
	if !strings.Contains(strings.ToLower(rec.Body.String()), "<!doctype html>") {
		t.Fatalf("expected embedded frontend html, got %q", rec.Body.String())
	}
}
