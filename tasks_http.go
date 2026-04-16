package main

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.taskRepository().ListTasks(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, map[string]any{"tasks": tasks})
}

func (a *App) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var task Task
	if !decodeJSON(w, r, taskBodyLimit, &task) {
		return
	}

	if err := validateTaskPayload(&task, true, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	createdTask, err := a.taskRepository().CreateTask(r.Context(), task)
	if errors.Is(err, errTaskConflict) {
		writeError(w, http.StatusConflict, "task id already exists")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResp(w, createdTask)
}

func (a *App) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	var task Task
	if !decodeJSON(w, r, taskBodyLimit, &task) {
		return
	}

	if err := validateTaskPayload(&task, false, false); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updatedTask, err := a.taskRepository().UpdateTask(r.Context(), id, task)
	if errors.Is(err, errTaskNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	jsonResp(w, updatedTask)
}

// handleArchiveTask moves a task from tasks to archived_tasks.
func (a *App) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	task, err := a.taskRepository().ArchiveTask(r.Context(), id)
	if errors.Is(err, errTaskNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
}

func (a *App) handleGetArchived(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.taskRepository().ListArchived(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, map[string]any{"tasks": tasks})
}

// handleRestoreTask moves a task from archived_tasks to tasks.
func (a *App) handleRestoreTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	task, err := a.taskRepository().RestoreTask(r.Context(), id)
	if errors.Is(err, errTaskNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, errStoredTaskInvalid) {
		writeError(w, http.StatusInternalServerError, "stored task is invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
}

func (a *App) handleDeleteArchived(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	err := a.taskRepository().DeleteArchived(r.Context(), id)
	if errors.Is(err, errTaskNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateTaskPayload(task *Task, requireID bool, allowStatus bool) error {
	if requireID {
		task.ID = strings.TrimSpace(task.ID)
		if task.ID == "" {
			return errors.New("id is required")
		}
	}

	task.Title = strings.TrimSpace(task.Title)
	task.Note = strings.TrimSpace(task.Note)

	if task.Title == "" {
		return errors.New("title is required")
	}
	if len(task.Title) > maxTitleLength {
		return errors.New("title is too long")
	}
	if len(task.Note) > maxNoteLength {
		return errors.New("note is too long")
	}
	if !validDueDate(task.Due) {
		return errors.New("due must be in YYYY-MM-DD format")
	}

	if task.Priority == "" {
		task.Priority = "medium"
	}
	if !validPriority(task.Priority) {
		return errors.New("invalid priority")
	}

	if allowStatus {
		if !validStatus(task.Status) {
			return errors.New("invalid status")
		}
	} else {
		task.Status = "queued"
	}

	return nil
}

func validPriority(priority string) bool {
	return priority == "critical" || priority == "high" || priority == "medium"
}

func validStatus(status string) bool {
	return status == "queued" || status == "active" || status == "done"
}

func validDueDate(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
