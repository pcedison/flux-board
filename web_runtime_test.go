package main

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewMuxServesReactRuntimeRoutes(t *testing.T) {
	app := &App{
		webRuntimeFS: newWebRuntimeFSTestHelper(t),
		passwordVerifier: func(context.Context, string) (bool, error) {
			return false, nil
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

	cases := []struct {
		name       string
		target     string
		wantBody   string
		wantStatus int
	}{
		{
			name:       "runtime root",
			target:     "/",
			wantBody:   "Flux Board Web Runtime",
			wantStatus: http.StatusOK,
		},
		{
			name:       "runtime login fallback",
			target:     "/login",
			wantBody:   "Flux Board Web Runtime",
			wantStatus: http.StatusOK,
		},
		{
			name:       "runtime board fallback",
			target:     "/board",
			wantBody:   "Flux Board Web Runtime",
			wantStatus: http.StatusOK,
		},
		{
			name:       "runtime asset",
			target:     "/assets/app.js",
			wantBody:   "runtime bundle",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d", tc.wantStatus, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("expected body to contain %q, got %q", tc.wantBody, rec.Body.String())
			}
		})
	}
}

func TestNewMuxServesLegacyRollbackRoute(t *testing.T) {
	app := &App{
		webRuntimeFS: newWebRuntimeFSTestHelper(t),
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

	redirectReq := httptest.NewRequest(http.MethodGet, "/legacy", nil)
	redirectRec := httptest.NewRecorder()
	mux.ServeHTTP(redirectRec, redirectReq)

	if redirectRec.Code != http.StatusPermanentRedirect {
		t.Fatalf("expected legacy redirect status 308, got %d", redirectRec.Code)
	}
	if location := redirectRec.Header().Get("Location"); location != "/legacy/" {
		t.Fatalf("expected /legacy redirect location, got %q", location)
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/legacy/", nil)
	legacyRec := httptest.NewRecorder()
	mux.ServeHTTP(legacyRec, legacyReq)

	if legacyRec.Code != http.StatusOK {
		t.Fatalf("expected legacy status 200, got %d", legacyRec.Code)
	}
	if !strings.Contains(strings.ToLower(legacyRec.Body.String()), "<!doctype html>") {
		t.Fatalf("expected legacy html, got %q", legacyRec.Body.String())
	}
}

func TestNewMuxRedirectsNextAliasToRootRuntime(t *testing.T) {
	app := &App{
		webRuntimeFS: newWebRuntimeFSTestHelper(t),
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
		name         string
		target       string
		wantLocation string
	}{
		{
			name:         "next root",
			target:       "/next",
			wantLocation: "/",
		},
		{
			name:         "next login",
			target:       "/next/login",
			wantLocation: "/login",
		},
		{
			name:         "next board",
			target:       "/next/board",
			wantLocation: "/board",
		},
		{
			name:         "next asset",
			target:       "/next/assets/app.js",
			wantLocation: "/assets/app.js",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.target, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusPermanentRedirect {
				t.Fatalf("expected status 308, got %d", rec.Code)
			}
			if location := rec.Header().Get("Location"); location != tc.wantLocation {
				t.Fatalf("expected redirect location %q, got %q", tc.wantLocation, location)
			}
		})
	}
}

func newWebRuntimeFSTestHelper(t *testing.T) fs.FS {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<!doctype html><html><body>Flux Board Web Runtime</body></html>`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte(`console.log("runtime bundle");`), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), 0o644); err != nil {
		t.Fatalf("write favicon: %v", err)
	}

	return os.DirFS(dir)
}
