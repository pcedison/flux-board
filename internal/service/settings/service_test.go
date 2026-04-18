package settings

import (
	"context"
	"errors"
	"testing"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"

	"golang.org/x/crypto/bcrypt"
)

type stubAuthRepository struct {
	bootstrapAdminExistsFn func(context.Context, string) (bool, error)
	bootstrapPasswordHash  string
	recordedEvents         []domain.AuthAuditEvent
}

func (s *stubAuthRepository) BootstrapPasswordHash(context.Context, string) (string, error) {
	if s.bootstrapPasswordHash == "" {
		return "", errors.New("missing password hash")
	}
	return s.bootstrapPasswordHash, nil
}

func (s *stubAuthRepository) EnsureBootstrapAdmin(context.Context, string, string) error {
	return nil
}

func (s *stubAuthRepository) BootstrapAdminExists(ctx context.Context, username string) (bool, error) {
	if s.bootstrapAdminExistsFn != nil {
		return s.bootstrapAdminExistsFn(ctx, username)
	}
	return false, nil
}

func (s *stubAuthRepository) UpdatePasswordHash(context.Context, string, string, int64) error {
	return nil
}

func (s *stubAuthRepository) GetActiveSession(context.Context, string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (s *stubAuthRepository) CreateSession(context.Context, string, string, string, time.Time) error {
	return nil
}

func (s *stubAuthRepository) DeleteSession(context.Context, string) error {
	return nil
}

func (s *stubAuthRepository) ListSessions(context.Context, string) ([]domain.SessionInfo, error) {
	return nil, nil
}

func (s *stubAuthRepository) DeleteSessionsExcept(context.Context, string, []string) error {
	return nil
}

func (s *stubAuthRepository) RecordAuthEvent(_ context.Context, event domain.AuthAuditEvent) error {
	s.recordedEvents = append(s.recordedEvents, event)
	return nil
}

type stubSettingsRepository struct {
	bootstrapAdminExistsFn func(context.Context, string) (bool, error)
	updatePasswordHashFn   func(context.Context, string, string, int64) error
	deleteSessionsExceptFn func(context.Context, string, []string) error
	setArchiveRetentionFn  func(context.Context, *int, int64) error
}

func (s *stubSettingsRepository) BootstrapAdminExists(ctx context.Context, username string) (bool, error) {
	if s.bootstrapAdminExistsFn != nil {
		return s.bootstrapAdminExistsFn(ctx, username)
	}
	return false, nil
}

func (s *stubSettingsRepository) UpdatePasswordHash(ctx context.Context, username, passwordHash string, updatedAt int64) error {
	if s.updatePasswordHashFn != nil {
		return s.updatePasswordHashFn(ctx, username, passwordHash, updatedAt)
	}
	return nil
}

func (s *stubSettingsRepository) ListSessions(context.Context, string) ([]domain.SessionInfo, error) {
	return nil, nil
}

func (s *stubSettingsRepository) DeleteSessionsExcept(ctx context.Context, username string, keepTokens []string) error {
	if s.deleteSessionsExceptFn != nil {
		return s.deleteSessionsExceptFn(ctx, username, keepTokens)
	}
	return nil
}

func (s *stubSettingsRepository) GetArchiveRetentionDays(context.Context) (*int, error) {
	return nil, nil
}

func (s *stubSettingsRepository) SetArchiveRetentionDays(ctx context.Context, days *int, updatedAt int64) error {
	if s.setArchiveRetentionFn != nil {
		return s.setArchiveRetentionFn(ctx, days, updatedAt)
	}
	return nil
}

func (s *stubSettingsRepository) ReplaceBoardSnapshot(context.Context, []domain.Task, []domain.ArchivedTask) error {
	return nil
}

type stubTaskRepository struct {
	tasks     []domain.Task
	archived  []domain.ArchivedTask
}

func (s stubTaskRepository) ListTasks(context.Context) ([]domain.Task, error) {
	return s.tasks, nil
}

func (s stubTaskRepository) CreateTask(context.Context, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}

func (s stubTaskRepository) UpdateTask(context.Context, string, domain.Task) (domain.Task, error) {
	return domain.Task{}, nil
}

func (s stubTaskRepository) ReorderTask(context.Context, string, domain.TaskReorderInput) (domain.Task, error) {
	return domain.Task{}, nil
}

func (s stubTaskRepository) ArchiveTask(context.Context, string) (domain.ArchivedTask, error) {
	return domain.ArchivedTask{}, nil
}

func (s stubTaskRepository) ListArchived(context.Context) ([]domain.ArchivedTask, error) {
	return s.archived, nil
}

func (s stubTaskRepository) RestoreTask(context.Context, string) (domain.Task, error) {
	return domain.Task{}, nil
}

func (s stubTaskRepository) DeleteArchived(context.Context, string) error {
	return nil
}

func TestBootstrapStatusReportsSetupRequired(t *testing.T) {
	t.Parallel()

	service := New(
		&stubAuthRepository{},
		&stubSettingsRepository{
			bootstrapAdminExistsFn: func(context.Context, string) (bool, error) {
				return false, nil
			},
		},
		stubTaskRepository{},
		nil,
		"",
		Options{},
	)

	status, err := service.BootstrapStatus(context.Background())
	if err != nil {
		t.Fatalf("BootstrapStatus returned error: %v", err)
	}
	if !status.NeedsSetup {
		t.Fatal("expected setup to be required")
	}
}

func TestChangePasswordKeepsCurrentSessionOnly(t *testing.T) {
	t.Parallel()

	currentHash, err := bcrypt.GenerateFromPassword([]byte("current-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	var updatedHash string
	var keptTokens []string
	authRepo := &stubAuthRepository{
		bootstrapPasswordHash: string(currentHash),
	}
	settingsRepo := &stubSettingsRepository{
		updatePasswordHashFn: func(_ context.Context, username, passwordHash string, _ int64) error {
			if username != authservice.BootstrapAdmin {
				t.Fatalf("unexpected username: %s", username)
			}
			updatedHash = passwordHash
			return nil
		},
		deleteSessionsExceptFn: func(_ context.Context, username string, keepTokens []string) error {
			if username != authservice.BootstrapAdmin {
				t.Fatalf("unexpected username: %s", username)
			}
			keptTokens = append([]string{}, keepTokens...)
			return nil
		},
	}

	service := New(authRepo, settingsRepo, stubTaskRepository{}, nil, "dev", Options{})
	if err := service.ChangePassword(
		context.Background(),
		"current-password",
		"new-password-123",
		"keep-token",
		"127.0.0.1",
	); err != nil {
		t.Fatalf("ChangePassword returned error: %v", err)
	}

	if updatedHash == "" {
		t.Fatal("expected password hash update to be recorded")
	}
	if bcrypt.CompareHashAndPassword([]byte(updatedHash), []byte("new-password-123")) != nil {
		t.Fatal("expected new password hash to match the new password")
	}
	if len(keptTokens) != 1 || keptTokens[0] != "keep-token" {
		t.Fatalf("expected current session token to be preserved, got %+v", keptTokens)
	}
	if len(authRepo.recordedEvents) != 1 || authRepo.recordedEvents[0].Outcome != "success" {
		t.Fatalf("expected one successful auth audit event, got %+v", authRepo.recordedEvents)
	}
}

func TestUpdateSettingsRejectsInvalidRetention(t *testing.T) {
	t.Parallel()

	service := New(&stubAuthRepository{}, &stubSettingsRepository{}, stubTaskRepository{}, nil, "", Options{})
	days := 0

	_, err := service.UpdateSettings(context.Background(), UpdateSettingsInput{ArchiveRetentionDays: &days})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestImportRejectsDuplicateIDsAcrossTasksAndArchived(t *testing.T) {
	t.Parallel()

	service := New(&stubAuthRepository{}, &stubSettingsRepository{}, stubTaskRepository{}, nil, "", Options{})
	days := 30

	err := service.Import(context.Background(), domain.ExportBundle{
		Version:    "test",
		ExportedAt: time.Now().UnixMilli(),
		Settings: domain.AppSettings{
			ArchiveRetentionDays: &days,
		},
		Tasks: []domain.Task{
			{
				ID:        "task-1",
				Title:     "Queued",
				Note:      "",
				Due:       "2026-04-20",
				Priority:  "medium",
				Status:    "queued",
				SortOrder: 0,
			},
		},
		Archived: []domain.ArchivedTask{
			{
				ID:         "task-1",
				Title:      "Archived",
				Note:       "",
				Due:        "2026-04-21",
				Priority:   "high",
				Status:     "done",
				SortOrder:  0,
				ArchivedAt: time.Now().UnixMilli(),
			},
		},
	})
	if !IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
