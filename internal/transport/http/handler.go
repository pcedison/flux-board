package transporthttp

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"
	settingsservice "flux-board/internal/service/settings"
	taskservice "flux-board/internal/service/task"
)

const readinessTimeout = 2 * time.Second

type HandlerOptions struct {
	CookieSecure         bool
	AuthBodyLimit        int64
	SettingsBodyLimit    int64
	TaskBodyLimit        int64
	ReadinessChecker     func(context.Context) error
	AppEnvironment       string
	AppVersion           string
	ArchiveCleanupEvery  time.Duration
	SessionCleanupEvery  time.Duration
	RuntimeArtifact      string
	RuntimeOwnershipPath string
	LegacyRollbackPath   string
}

type Handler struct {
	taskService     taskservice.Service
	authService     authservice.Service
	settingsService settingsservice.Service
	options         HandlerOptions
}

type taskReorderRequest struct {
	Status       string `json:"status"`
	AnchorTaskID string `json:"anchorTaskId"`
	PlaceAfter   bool   `json:"placeAfter"`
}

type statusCheck struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type appStatusResponse struct {
	Status               string        `json:"status"`
	Version              string        `json:"version"`
	Environment          string        `json:"environment"`
	NeedsSetup           bool          `json:"needsSetup"`
	ArchiveRetentionDays *int          `json:"archiveRetentionDays"`
	RuntimeArtifact      string        `json:"runtimeArtifact"`
	RuntimeOwnershipPath string        `json:"runtimeOwnershipPath"`
	LegacyRollbackPath   string        `json:"legacyRollbackPath"`
	ArchiveCleanupEvery  string        `json:"archiveCleanupEvery"`
	SessionCleanupEvery  string        `json:"sessionCleanupEvery"`
	GeneratedAt          int64         `json:"generatedAt"`
	Checks               []statusCheck `json:"checks"`
}

func NewHandler(taskService taskservice.Service, authService authservice.Service, options HandlerOptions) *Handler {
	return &Handler{
		taskService: taskService,
		authService: authService,
		options:     options,
	}
}

func NewHandlerWithSettings(taskService taskservice.Service, authService authservice.Service, settingsService settingsservice.Service, options HandlerOptions) *Handler {
	return &Handler{
		taskService:     taskService,
		authService:     authService,
		settingsService: settingsService,
		options:         options,
	}
}

func (h *Handler) Auth(next stdhttp.HandlerFunc) stdhttp.HandlerFunc {
	return func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		cookie, err := r.Cookie(CookieName)
		if err != nil {
			WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
			return
		}

		session, err := h.authService.SessionFromToken(r.Context(), cookie.Value, ClientIDFromRequest(r))
		if err != nil {
			if errors.Is(err, authservice.ErrInvalidSession) {
				ClearSessionCookie(w, h.options.CookieSecure)
				WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
				return
			}
			WriteError(w, stdhttp.StatusInternalServerError, "db error")
			return
		}

		next(w, r.WithContext(withSession(r.Context(), session)))
	}
}

func (h *Handler) HandleLogin(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var payload struct {
		Password string `json:"password"`
	}
	if !DecodeJSON(w, r, h.options.AuthBodyLimit, &payload) {
		return
	}

	password := strings.TrimSpace(payload.Password)
	if password == "" {
		WriteError(w, stdhttp.StatusBadRequest, "password is required")
		return
	}

	authSession, err := h.authService.Authenticate(r.Context(), password, ClientIDFromRequest(r))
	if err != nil {
		switch {
		case errors.Is(err, authservice.ErrBlocked):
			WriteError(w, stdhttp.StatusTooManyRequests, "too many attempts, try again later")
		case errors.Is(err, authservice.ErrSetupRequired):
			WriteError(w, stdhttp.StatusConflict, "setup required")
		case errors.Is(err, authservice.ErrInvalidPassword):
			WriteError(w, stdhttp.StatusUnauthorized, "invalid password")
		default:
			WriteError(w, stdhttp.StatusInternalServerError, "db error")
		}
		return
	}

	SetSessionCookie(w, authSession.Token, authSession.ExpiresAt, h.options.CookieSecure)
	JSONResp(w, map[string]any{"ok": true, "expiresAt": authSession.ExpiresAt.UnixMilli()})
}

func (h *Handler) HandleLogout(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		ClearSessionCookie(w, h.options.CookieSecure)
		w.WriteHeader(stdhttp.StatusOK)
		return
	}

	if err := h.authService.Logout(r.Context(), cookie.Value, ClientIDFromRequest(r)); err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	ClearSessionCookie(w, h.options.CookieSecure)
	w.WriteHeader(stdhttp.StatusOK)
}

func (h *Handler) HandleGetSession(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	session, ok := SessionFromContext(r.Context())
	if !ok {
		WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	JSONResp(w, map[string]any{
		"authenticated": true,
		"expiresAt":     session.ExpiresAt.UnixMilli(),
	})
}

func (h *Handler) HandleGetTasks(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	tasks, err := h.taskService.ListTasks(r.Context())
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, map[string]any{"tasks": tasks})
}

func (h *Handler) HandleCreateTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var task domain.Task
	if !DecodeJSON(w, r, h.options.TaskBodyLimit, &task) {
		return
	}

	createdTask, err := h.taskService.CreateTask(r.Context(), task)
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskConflict) {
		WriteError(w, stdhttp.StatusConflict, "task id already exists")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(stdhttp.StatusCreated)
	JSONResp(w, createdTask)
}

func (h *Handler) HandleUpdateTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var task domain.Task
	if !DecodeJSON(w, r, h.options.TaskBodyLimit, &task) {
		return
	}

	updatedTask, err := h.taskService.UpdateTask(r.Context(), r.PathValue("id"), task)
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		WriteError(w, stdhttp.StatusNotFound, "not found")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, updatedTask)
}

func (h *Handler) HandleReorderTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var req taskReorderRequest
	if !DecodeJSON(w, r, h.options.TaskBodyLimit, &req) {
		return
	}

	task, err := h.taskService.ReorderTask(r.Context(), r.PathValue("id"), domain.TaskReorderInput{
		Status:       req.Status,
		AnchorTaskID: req.AnchorTaskID,
		PlaceAfter:   req.PlaceAfter,
	})
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		WriteError(w, stdhttp.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, domain.ErrTaskInvalidAnchor) {
		WriteError(w, stdhttp.StatusBadRequest, "anchor task is invalid")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, task)
}

func (h *Handler) HandleArchiveTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	task, err := h.taskService.ArchiveTask(r.Context(), r.PathValue("id"))
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		WriteError(w, stdhttp.StatusNotFound, "not found")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, task)
}

func (h *Handler) HandleGetArchived(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	tasks, err := h.taskService.ListArchived(r.Context())
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, map[string]any{"tasks": tasks})
}

func (h *Handler) HandleRestoreTask(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	task, err := h.taskService.RestoreTask(r.Context(), r.PathValue("id"))
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		WriteError(w, stdhttp.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, domain.ErrStoredTaskInvalid) {
		WriteError(w, stdhttp.StatusInternalServerError, "stored task is invalid")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	JSONResp(w, task)
}

func (h *Handler) HandleDeleteArchived(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	err := h.taskService.DeleteArchived(r.Context(), r.PathValue("id"))
	if domain.IsTaskValidationError(err) {
		WriteError(w, stdhttp.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrTaskNotFound) {
		WriteError(w, stdhttp.StatusNotFound, "not found")
		return
	}
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(stdhttp.StatusNoContent)
}

func (h *Handler) HandleStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	status := appStatusResponse{
		Status:               "ready",
		Version:              defaultString(h.options.AppVersion, "dev"),
		Environment:          defaultString(h.options.AppEnvironment, "production"),
		RuntimeArtifact:      defaultString(h.options.RuntimeArtifact, "self-contained-root-runtime"),
		RuntimeOwnershipPath: defaultString(h.options.RuntimeOwnershipPath, "/"),
		LegacyRollbackPath:   defaultString(h.options.LegacyRollbackPath, "/legacy/"),
		ArchiveCleanupEvery:  formatDuration(h.options.ArchiveCleanupEvery),
		SessionCleanupEvery:  formatDuration(h.options.SessionCleanupEvery),
		GeneratedAt:          time.Now().UnixMilli(),
	}

	if h.options.ReadinessChecker != nil {
		readinessCtx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		err := h.options.ReadinessChecker(readinessCtx)
		cancel()
		if err != nil {
			status.Status = "degraded"
			status.Checks = append(status.Checks, statusCheck{
				Name:    "database",
				OK:      false,
				Message: err.Error(),
			})
		} else {
			status.Checks = append(status.Checks, statusCheck{
				Name:    "database",
				OK:      true,
				Message: "database reachable",
			})
		}
	}

	bootstrapStatus, err := h.settingsService.BootstrapStatus(r.Context())
	if err != nil {
		status.Status = "degraded"
		status.Checks = append(status.Checks, statusCheck{
			Name:    "bootstrap",
			OK:      false,
			Message: fmt.Sprintf("bootstrap status unavailable: %v", err),
		})
	} else {
		status.NeedsSetup = bootstrapStatus.NeedsSetup
		status.Checks = append(status.Checks, statusCheck{
			Name:    "bootstrap",
			OK:      !bootstrapStatus.NeedsSetup,
			Message: bootstrapMessage(bootstrapStatus.NeedsSetup),
		})
	}

	settings, err := h.settingsService.GetSettings(r.Context())
	if err != nil {
		status.Status = "degraded"
		status.Checks = append(status.Checks, statusCheck{
			Name:    "archive-retention",
			OK:      false,
			Message: fmt.Sprintf("archive retention unavailable: %v", err),
		})
	} else {
		status.ArchiveRetentionDays = settings.ArchiveRetentionDays
		status.Checks = append(status.Checks, statusCheck{
			Name:    "archive-retention",
			OK:      true,
			Message: archiveRetentionMessage(settings.ArchiveRetentionDays),
		})
	}

	if status.Status == "degraded" {
		w.WriteHeader(stdhttp.StatusServiceUnavailable)
	}
	JSONResp(w, status)
}

func (h *Handler) HandleBootstrapStatus(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	status, err := h.settingsService.BootstrapStatus(r.Context())
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}
	JSONResp(w, status)
}

func (h *Handler) HandleBootstrapSetup(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	var payload struct {
		Password string `json:"password"`
	}
	if !DecodeJSON(w, r, h.options.SettingsBodyLimit, &payload) {
		return
	}

	result, err := h.settingsService.Bootstrap(r.Context(), payload.Password, ClientIDFromRequest(r))
	if err != nil {
		switch {
		case errors.Is(err, settingsservice.ErrAlreadyConfigured):
			WriteError(w, stdhttp.StatusConflict, "already configured")
		case settingsservice.IsValidationError(err):
			WriteError(w, stdhttp.StatusBadRequest, err.Error())
		default:
			WriteError(w, stdhttp.StatusInternalServerError, "db error")
		}
		return
	}

	SetSessionCookie(w, result.Token, result.ExpiresAt, h.options.CookieSecure)
	JSONResp(w, map[string]any{"ok": true, "expiresAt": result.ExpiresAt.UnixMilli()})
}

func (h *Handler) HandleGetSettings(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	settings, err := h.settingsService.GetSettings(r.Context())
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}
	JSONResp(w, settings)
}

func (h *Handler) HandleUpdateSettings(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	var payload struct {
		ArchiveRetentionDays *int `json:"archiveRetentionDays"`
	}
	if !DecodeJSON(w, r, h.options.SettingsBodyLimit, &payload) {
		return
	}

	settings, err := h.settingsService.UpdateSettings(r.Context(), settingsservice.UpdateSettingsInput{
		ArchiveRetentionDays: payload.ArchiveRetentionDays,
	})
	if err != nil {
		if settingsservice.IsValidationError(err) {
			WriteError(w, stdhttp.StatusBadRequest, err.Error())
			return
		}
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}
	JSONResp(w, settings)
}

func (h *Handler) HandleChangePassword(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	session, ok := SessionFromContext(r.Context())
	if !ok {
		WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	var payload struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !DecodeJSON(w, r, h.options.SettingsBodyLimit, &payload) {
		return
	}

	err := h.settingsService.ChangePassword(
		r.Context(),
		payload.CurrentPassword,
		payload.NewPassword,
		session.Token,
		ClientIDFromRequest(r),
	)
	if err != nil {
		switch {
		case settingsservice.IsValidationError(err):
			WriteError(w, stdhttp.StatusBadRequest, err.Error())
		case errors.Is(err, authservice.ErrInvalidPassword):
			WriteError(w, stdhttp.StatusUnauthorized, "invalid password")
		case errors.Is(err, authservice.ErrSetupRequired):
			WriteError(w, stdhttp.StatusConflict, "setup required")
		default:
			WriteError(w, stdhttp.StatusInternalServerError, "db error")
		}
		return
	}

	w.WriteHeader(stdhttp.StatusNoContent)
}

func (h *Handler) HandleGetSessions(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	session, ok := SessionFromContext(r.Context())
	if !ok {
		WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	sessions, err := h.settingsService.ListSessions(r.Context(), session.Token)
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}
	JSONResp(w, map[string]any{"sessions": sessions})
}

func (h *Handler) HandleDeleteSession(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	session, ok := SessionFromContext(r.Context())
	if !ok {
		WriteError(w, stdhttp.StatusUnauthorized, "unauthorized")
		return
	}

	err := h.settingsService.RevokeSession(r.Context(), r.PathValue("token"), session.Token, ClientIDFromRequest(r))
	if err != nil {
		switch {
		case settingsservice.IsValidationError(err):
			WriteError(w, stdhttp.StatusBadRequest, err.Error())
		case errors.Is(err, settingsservice.ErrSessionNotFound):
			WriteError(w, stdhttp.StatusNotFound, "not found")
		default:
			WriteError(w, stdhttp.StatusInternalServerError, "db error")
		}
		return
	}

	if session.Token == r.PathValue("token") {
		ClearSessionCookie(w, h.options.CookieSecure)
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

func (h *Handler) HandleExport(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	bundle, err := h.settingsService.Export(r.Context())
	if err != nil {
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=\"flux-board-export.json\"")
	JSONResp(w, bundle)
}

func (h *Handler) HandleImport(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	if h.settingsService == nil {
		WriteError(w, stdhttp.StatusNotImplemented, "settings unavailable")
		return
	}

	var bundle domain.ExportBundle
	if !DecodeJSON(w, r, h.options.SettingsBodyLimit, &bundle) {
		return
	}

	if err := h.settingsService.Import(r.Context(), bundle); err != nil {
		if settingsservice.IsValidationError(err) {
			WriteError(w, stdhttp.StatusBadRequest, err.Error())
			return
		}
		WriteError(w, stdhttp.StatusInternalServerError, "db error")
		return
	}
	w.WriteHeader(stdhttp.StatusNoContent)
}

func (h *Handler) HandleHealthz(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	setProbeHeaders(w)
	JSONResp(w, map[string]string{"status": "ok"})
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func formatDuration(value time.Duration) string {
	if value <= 0 {
		return "disabled"
	}
	return value.String()
}

func bootstrapMessage(needsSetup bool) string {
	if needsSetup {
		return "browser setup still required"
	}
	return "admin password already configured"
}

func archiveRetentionMessage(days *int) string {
	if days == nil {
		return "archived cards are kept until you delete them"
	}
	return fmt.Sprintf("archived cards auto-delete after %d days", *days)
}

func (h *Handler) HandleReadyz(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	setProbeHeaders(w)
	if err := h.checkReadiness(r.Context()); err != nil {
		WriteError(w, stdhttp.StatusServiceUnavailable, "not ready")
		return
	}
	JSONResp(w, map[string]string{"status": "ready"})
}

func setProbeHeaders(w stdhttp.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func (h *Handler) checkReadiness(ctx context.Context) error {
	if h.options.ReadinessChecker == nil {
		return errors.New("readiness checker not configured")
	}

	readinessCtx, cancel := context.WithTimeout(ctx, readinessTimeout)
	defer cancel()
	return h.options.ReadinessChecker(readinessCtx)
}
