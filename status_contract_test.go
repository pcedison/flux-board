package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"
	settingsservice "flux-board/internal/service/settings"
)

type statusContractSettingsService struct {
	bootstrapStatus settingsservice.BootstrapStatus
	bootstrapErr    error
	settings        domain.AppSettings
	settingsErr     error
}

type statusCheckResponse struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type statusContractResponse struct {
	Status               string                `json:"status"`
	Version              string                `json:"version"`
	Environment          string                `json:"environment"`
	NeedsSetup           bool                  `json:"needsSetup"`
	RuntimeArtifact      string                `json:"runtimeArtifact"`
	RuntimeOwnershipPath string                `json:"runtimeOwnershipPath"`
	LegacyRollbackPath   string                `json:"legacyRollbackPath"`
	ArchiveCleanupEvery  string                `json:"archiveCleanupEvery"`
	SessionCleanupEvery  string                `json:"sessionCleanupEvery"`
	GeneratedAt          int64                 `json:"generatedAt"`
	ArchiveRetentionDays *int                  `json:"archiveRetentionDays"`
	Checks               []statusCheckResponse `json:"checks"`
}

func (s statusContractSettingsService) BootstrapStatus(context.Context) (settingsservice.BootstrapStatus, error) {
	if s.bootstrapErr != nil {
		return settingsservice.BootstrapStatus{}, s.bootstrapErr
	}
	return s.bootstrapStatus, nil
}

func (s statusContractSettingsService) Bootstrap(context.Context, string, string) (authservice.LoginResult, error) {
	return authservice.LoginResult{}, nil
}

func (s statusContractSettingsService) GetSettings(context.Context) (domain.AppSettings, error) {
	if s.settingsErr != nil {
		return domain.AppSettings{}, s.settingsErr
	}
	return s.settings, nil
}

func (s statusContractSettingsService) UpdateSettings(context.Context, settingsservice.UpdateSettingsInput) (domain.AppSettings, error) {
	return domain.AppSettings{}, nil
}

func (s statusContractSettingsService) ChangePassword(context.Context, string, string, string, string) error {
	return nil
}

func (s statusContractSettingsService) ListSessions(context.Context, string) ([]domain.SessionInfo, error) {
	return nil, nil
}

func (s statusContractSettingsService) RevokeSession(context.Context, string, string, string) error {
	return nil
}

func (s statusContractSettingsService) Export(context.Context) (domain.ExportBundle, error) {
	return domain.ExportBundle{}, nil
}

func (s statusContractSettingsService) Import(context.Context, domain.ExportBundle) error {
	return nil
}

func TestStatusEndpointReturnsReadyOperatorContract(t *testing.T) {
	retentionDays := 30
	app := &App{
		appEnv:  "production",
		version: "1.2.3",
		settingsSvc: statusContractSettingsService{
			bootstrapStatus: settingsservice.BootstrapStatus{NeedsSetup: false},
			settings: domain.AppSettings{
				ArchiveRetentionDays: &retentionDays,
			},
		},
		readinessChecker: func(context.Context) error {
			return nil
		},
	}

	resp := requestStatusContract(t, app)

	if resp.Status != "ready" {
		t.Fatalf("expected status ready, got %+v", resp)
	}
	if resp.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %+v", resp)
	}
	if resp.Environment != "production" {
		t.Fatalf("expected environment production, got %+v", resp)
	}
	if resp.NeedsSetup {
		t.Fatalf("expected needsSetup false, got %+v", resp)
	}
	if resp.RuntimeArtifact != "self-contained-root-runtime" {
		t.Fatalf("expected runtimeArtifact self-contained-root-runtime, got %+v", resp)
	}
	if resp.RuntimeOwnershipPath != "/" {
		t.Fatalf("expected runtimeOwnershipPath /, got %+v", resp)
	}
	if resp.LegacyRollbackPath != "/legacy/" {
		t.Fatalf("expected legacyRollbackPath /legacy/, got %+v", resp)
	}
	if resp.ArchiveCleanupEvery != "1h0m0s" {
		t.Fatalf("expected archiveCleanupEvery 1h0m0s, got %+v", resp)
	}
	if resp.SessionCleanupEvery == "" || resp.SessionCleanupEvery == "disabled" {
		t.Fatalf("expected sessionCleanupEvery to be populated, got %+v", resp)
	}
	if resp.GeneratedAt <= 0 || resp.GeneratedAt > time.Now().Add(time.Minute).UnixMilli() {
		t.Fatalf("expected generatedAt to be a recent timestamp, got %+v", resp)
	}
	if resp.ArchiveRetentionDays == nil || *resp.ArchiveRetentionDays != retentionDays {
		t.Fatalf("expected archiveRetentionDays %d, got %+v", retentionDays, resp)
	}
	if len(resp.Checks) != 3 {
		t.Fatalf("expected three status checks, got %+v", resp.Checks)
	}

	assertStatusCheck(t, resp.Checks[0], "database", true, "database reachable")
	assertStatusCheck(t, resp.Checks[1], "bootstrap", true, "admin password already configured")
	assertStatusCheck(t, resp.Checks[2], "archive-retention", true, "archived cards auto-delete after 30 days")
}

func TestStatusEndpointReturnsDegradedContractWhenChecksFail(t *testing.T) {
	app := &App{
		appEnv:  "development",
		version: "dev-build",
		settingsSvc: statusContractSettingsService{
			bootstrapErr: errors.New("bootstrap query failed"),
			settingsErr:  errors.New("settings query failed"),
		},
		readinessChecker: func(context.Context) error {
			return errors.New("dial tcp timeout")
		},
	}

	rec := httptest.NewRecorder()
	mux, err := newMux(app)
	if err != nil {
		t.Fatalf("newMux returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected /api/status to return 503, got %d", rec.Code)
	}

	var resp statusContractResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}

	if resp.Status != "degraded" {
		t.Fatalf("expected degraded status, got %+v", resp)
	}
	if resp.Environment != "development" || resp.Version != "dev-build" {
		t.Fatalf("expected environment/version to round-trip, got %+v", resp)
	}
	if len(resp.Checks) != 3 {
		t.Fatalf("expected three degraded checks, got %+v", resp.Checks)
	}

	assertStatusCheck(t, resp.Checks[0], "database", false, "dial tcp timeout")
	assertStatusCheck(t, resp.Checks[1], "bootstrap", false, "bootstrap status unavailable: bootstrap query failed")
	assertStatusCheck(t, resp.Checks[2], "archive-retention", false, "archive retention unavailable: settings query failed")
}

func requestStatusContract(t *testing.T, app *App) statusContractResponse {
	t.Helper()

	rec := httptest.NewRecorder()
	mux, err := newMux(app)
	if err != nil {
		t.Fatalf("newMux returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected /api/status to return 200, got %d", rec.Code)
	}

	var resp statusContractResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode status response: %v", err)
	}
	return resp
}

func assertStatusCheck(t *testing.T, got statusCheckResponse, wantName string, wantOK bool, wantMessage string) {
	t.Helper()
	if got.Name != wantName || got.OK != wantOK || got.Message != wantMessage {
		t.Fatalf("expected check (%s,%t,%s), got %+v", wantName, wantOK, wantMessage, got)
	}
}
