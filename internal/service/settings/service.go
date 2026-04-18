package settings

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength      = 10
	maxArchiveRetentionDay = 3650
)

var (
	ErrAlreadyConfigured = errors.New("already configured")
	ErrSessionNotFound   = errors.New("session not found")
)

type validationError struct {
	message string
}

func (e validationError) Error() string {
	return e.message
}

func NewValidationError(message string) error {
	return validationError{message: message}
}

func IsValidationError(err error) bool {
	var validationErr validationError
	return errors.As(err, &validationErr)
}

type Service interface {
	BootstrapStatus(context.Context) (BootstrapStatus, error)
	Bootstrap(context.Context, string, string) (authservice.LoginResult, error)
	GetSettings(context.Context) (domain.AppSettings, error)
	UpdateSettings(context.Context, UpdateSettingsInput) (domain.AppSettings, error)
	ChangePassword(context.Context, string, string, string, string) error
	ListSessions(context.Context, string) ([]domain.SessionInfo, error)
	RevokeSession(context.Context, string, string, string) error
	Export(context.Context) (domain.ExportBundle, error)
	Import(context.Context, domain.ExportBundle) error
}

type BootstrapStatus struct {
	NeedsSetup bool `json:"needsSetup"`
}

type UpdateSettingsInput struct {
	ArchiveRetentionDays *int `json:"archiveRetentionDays"`
}

type Options struct {
	Clock func() time.Time
}

type service struct {
	authRepo     domain.AuthRepository
	settingsRepo domain.SettingsRepository
	taskRepo     domain.TaskRepository
	authService  authservice.Service
	version      string
	options      Options
}

func New(
	authRepo domain.AuthRepository,
	settingsRepo domain.SettingsRepository,
	taskRepo domain.TaskRepository,
	authSvc authservice.Service,
	version string,
	options Options,
) Service {
	return service{
		authRepo:     authRepo,
		settingsRepo: settingsRepo,
		taskRepo:     taskRepo,
		authService:  authSvc,
		version:      strings.TrimSpace(version),
		options:      options,
	}
}

func (s service) BootstrapStatus(ctx context.Context) (BootstrapStatus, error) {
	exists, err := s.settingsRepo.BootstrapAdminExists(ctx, authservice.BootstrapAdmin)
	if err != nil {
		return BootstrapStatus{}, err
	}
	return BootstrapStatus{NeedsSetup: !exists}, nil
}

func (s service) Bootstrap(ctx context.Context, password, clientIP string) (authservice.LoginResult, error) {
	password = strings.TrimSpace(password)
	if err := validatePassword(password); err != nil {
		return authservice.LoginResult{}, err
	}

	exists, err := s.settingsRepo.BootstrapAdminExists(ctx, authservice.BootstrapAdmin)
	if err != nil {
		return authservice.LoginResult{}, err
	}
	if exists {
		return authservice.LoginResult{}, ErrAlreadyConfigured
	}

	if err := s.authRepo.EnsureBootstrapAdmin(ctx, authservice.BootstrapAdmin, password); err != nil {
		return authservice.LoginResult{}, err
	}

	s.recordAuthEvent(ctx, domain.AuthAuditEvent{
		Username:  authservice.BootstrapAdmin,
		EventType: "bootstrap",
		Outcome:   "success",
		ClientIP:  clientIP,
	})
	return s.authService.Authenticate(ctx, password, clientIP)
}

func (s service) GetSettings(ctx context.Context) (domain.AppSettings, error) {
	days, err := s.settingsRepo.GetArchiveRetentionDays(ctx)
	if err != nil {
		return domain.AppSettings{}, err
	}
	return domain.AppSettings{ArchiveRetentionDays: days}, nil
}

func (s service) UpdateSettings(ctx context.Context, input UpdateSettingsInput) (domain.AppSettings, error) {
	if err := validateArchiveRetentionDays(input.ArchiveRetentionDays); err != nil {
		return domain.AppSettings{}, err
	}

	if err := s.settingsRepo.SetArchiveRetentionDays(ctx, input.ArchiveRetentionDays, s.now().UnixMilli()); err != nil {
		return domain.AppSettings{}, err
	}
	return domain.AppSettings{ArchiveRetentionDays: cloneIntPointer(input.ArchiveRetentionDays)}, nil
}

func (s service) ChangePassword(ctx context.Context, currentPassword, newPassword, currentToken, clientIP string) error {
	currentPassword = strings.TrimSpace(currentPassword)
	newPassword = strings.TrimSpace(newPassword)
	if currentPassword == "" {
		return NewValidationError("current password is required")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	hash, err := s.authRepo.BootstrapPasswordHash(ctx, authservice.BootstrapAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		return authservice.ErrSetupRequired
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(currentPassword)) != nil {
		s.recordAuthEvent(ctx, domain.AuthAuditEvent{
			Username:  authservice.BootstrapAdmin,
			EventType: "password_change",
			Outcome:   "failed",
			ClientIP:  clientIP,
			Details:   "current password mismatch",
		})
		return authservice.ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.settingsRepo.UpdatePasswordHash(ctx, authservice.BootstrapAdmin, string(newHash), s.now().UnixMilli()); err != nil {
		return err
	}
	keepTokens := []string{}
	if strings.TrimSpace(currentToken) != "" {
		keepTokens = append(keepTokens, currentToken)
	}
	if err := s.settingsRepo.DeleteSessionsExcept(ctx, authservice.BootstrapAdmin, keepTokens); err != nil {
		return err
	}

	s.recordAuthEvent(ctx, domain.AuthAuditEvent{
		Username:  authservice.BootstrapAdmin,
		EventType: "password_change",
		Outcome:   "success",
		ClientIP:  clientIP,
	})
	return nil
}

func (s service) ListSessions(ctx context.Context, currentToken string) ([]domain.SessionInfo, error) {
	sessions, err := s.settingsRepo.ListSessions(ctx, authservice.BootstrapAdmin)
	if err != nil {
		return nil, err
	}
	for idx := range sessions {
		sessions[idx].Current = sessions[idx].Token == currentToken
	}
	return sessions, nil
}

func (s service) RevokeSession(ctx context.Context, token, currentToken, clientIP string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return NewValidationError("session token is required")
	}

	sessions, err := s.settingsRepo.ListSessions(ctx, authservice.BootstrapAdmin)
	if err != nil {
		return err
	}
	found := false
	for _, session := range sessions {
		if session.Token == token {
			found = true
			break
		}
	}
	if !found {
		return ErrSessionNotFound
	}
	if err := s.authRepo.DeleteSession(ctx, token); err != nil {
		return err
	}
	outcome := "success"
	if token == currentToken {
		outcome = "self-revoked"
	}
	s.recordAuthEvent(ctx, domain.AuthAuditEvent{
		Username:  authservice.BootstrapAdmin,
		EventType: "session_revoke",
		Outcome:   outcome,
		ClientIP:  clientIP,
		Details:   fmt.Sprintf("token=%s", token),
	})
	return nil
}

func (s service) Export(ctx context.Context) (domain.ExportBundle, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	tasks, err := s.taskRepo.ListTasks(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	archived, err := s.taskRepo.ListArchived(ctx)
	if err != nil {
		return domain.ExportBundle{}, err
	}
	return domain.ExportBundle{
		Version:    defaultVersion(s.version),
		ExportedAt: s.now().UnixMilli(),
		Settings:   settings,
		Tasks:      tasks,
		Archived:   archived,
	}, nil
}

func (s service) Import(ctx context.Context, bundle domain.ExportBundle) error {
	if err := validateArchiveRetentionDays(bundle.Settings.ArchiveRetentionDays); err != nil {
		return err
	}
	if err := validateImportTasks(bundle.Tasks, bundle.Archived); err != nil {
		return err
	}
	if err := s.settingsRepo.ReplaceBoardSnapshot(ctx, bundle.Tasks, bundle.Archived); err != nil {
		return err
	}
	return s.settingsRepo.SetArchiveRetentionDays(ctx, bundle.Settings.ArchiveRetentionDays, s.now().UnixMilli())
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return NewValidationError(fmt.Sprintf("password must be at least %d characters", minPasswordLength))
	}
	return nil
}

func validateArchiveRetentionDays(days *int) error {
	if days == nil {
		return nil
	}
	if *days < 1 {
		return NewValidationError("archive retention must be at least 1 day")
	}
	if *days > maxArchiveRetentionDay {
		return NewValidationError(fmt.Sprintf("archive retention must be %d days or fewer", maxArchiveRetentionDay))
	}
	return nil
}

func validateImportTasks(tasks []domain.Task, archived []domain.ArchivedTask) error {
	seenIDs := make(map[string]string, len(tasks)+len(archived))
	for _, task := range tasks {
		if _, exists := seenIDs[task.ID]; exists {
			return NewValidationError("task ids must be unique across the import bundle")
		}
		seenIDs[task.ID] = "task"

		candidate := task
		if err := domain.ValidateTaskPayload(&candidate, true, true); err != nil {
			return NewValidationError(err.Error())
		}
		if task.SortOrder < 0 {
			return NewValidationError("task sort order cannot be negative")
		}
	}
	for _, task := range archived {
		if _, exists := seenIDs[task.ID]; exists {
			return NewValidationError("task ids must be unique across the import bundle")
		}
		seenIDs[task.ID] = "archived"

		candidate := domain.Task{
			ID:        task.ID,
			Title:     task.Title,
			Note:      task.Note,
			Due:       task.Due,
			Priority:  task.Priority,
			Status:    task.Status,
			SortOrder: task.SortOrder,
		}
		if err := domain.ValidateTaskPayload(&candidate, true, true); err != nil {
			return NewValidationError(err.Error())
		}
		if task.SortOrder < 0 {
			return NewValidationError("archived task sort order cannot be negative")
		}
		if task.ArchivedAt <= 0 {
			return NewValidationError("archived tasks must include archivedAt")
		}
	}
	return nil
}

func (s service) now() time.Time {
	if s.options.Clock != nil {
		return s.options.Clock()
	}
	return time.Now()
}

func (s service) recordAuthEvent(ctx context.Context, event domain.AuthAuditEvent) {
	if s.authRepo == nil {
		return
	}
	event.CreatedAt = s.now().UnixMilli()
	if err := s.authRepo.RecordAuthEvent(ctx, event); err != nil {
		slog.Default().Error("settings auth audit insert error", slog.Any("err", err))
	}
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func defaultVersion(version string) string {
	if version == "" {
		return "local"
	}
	return version
}
