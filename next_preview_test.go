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

func TestNewMuxServesReactPreviewRoutes(t *testing.T) {
	app := &App{
		webPreviewFS:  newPreviewFSTestHelper(t),
		loginAttempts: make(map[string]loginAttemptState),
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
		name         string
		target       string
		wantStatus   int
		wantBody     string
		wantLocation string
	}{
		{
			name:       "preview root",
			target:     "/next/",
			wantStatus: http.StatusOK,
			wantBody:   "Flux Board Next Preview",
		},
		{
			name:       "preview login fallback",
			target:     "/next/login",
			wantStatus: http.StatusOK,
			wantBody:   "Flux Board Next Preview",
		},
		{
			name:       "preview board fallback",
			target:     "/next/board",
			wantStatus: http.StatusOK,
			wantBody:   "Flux Board Next Preview",
		},
		{
			name:       "preview asset",
			target:     "/next/assets/app.js",
			wantStatus: http.StatusOK,
			wantBody:   "preview bundle",
		},
		{
			name:         "preview redirect",
			target:       "/next",
			wantStatus:   http.StatusPermanentRedirect,
			wantLocation: "/next/",
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
			if tc.wantBody != "" && !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("expected body to contain %q, got %q", tc.wantBody, rec.Body.String())
			}
			if tc.wantLocation != "" && rec.Header().Get("Location") != tc.wantLocation {
				t.Fatalf("expected redirect location %q, got %q", tc.wantLocation, rec.Header().Get("Location"))
			}
		})
	}
}

func newPreviewFSTestHelper(t *testing.T) fs.FS {
	t.Helper()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<!doctype html><html><body>Flux Board Next Preview</body></html>`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte(`console.log("preview bundle");`), 0o644); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), 0o644); err != nil {
		t.Fatalf("write favicon: %v", err)
	}

	return os.DirFS(dir)
}
