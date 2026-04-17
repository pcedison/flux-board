package transporthttp

import (
	"context"
	"errors"
	stdhttp "net/http"
	"strings"
	"time"

	"flux-board/internal/domain"
	authservice "flux-board/internal/service/auth"
	taskservice "flux-board/internal/service/task"
)

const readinessTimeout = 2 * time.Second

type HandlerOptions struct {
	CookieSecure     bool
	AuthBodyLimit    int64
	TaskBodyLimit    int64
	ReadinessChecker func(context.Context) error
}

type Handler struct {
	taskService taskservice.Service
	authService authservice.Service
	options     HandlerOptions
}

type taskReorderRequest struct {
	Status       string `json:"status"`
	AnchorTaskID string `json:"anchorTaskId"`
	PlaceAfter   bool   `json:"placeAfter"`
}

func NewHandler(taskService taskservice.Service, authService authservice.Service, options HandlerOptions) *Handler {
	return &Handler{
		taskService: taskService,
		authService: authService,
		options:     options,
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

func (h *Handler) HandleHealthz(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	setProbeHeaders(w)
	JSONResp(w, map[string]string{"status": "ok"})
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
