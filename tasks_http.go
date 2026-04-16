package main

import (
	"errors"
	"net/http"
	"strings"
)

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := a.taskService().ListTasks(r.Context())
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

	createdTask, err := a.taskService().CreateTask(r.Context(), task)
	if isTaskValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
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

	updatedTask, err := a.taskService().UpdateTask(r.Context(), id, task)
	if isTaskValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
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

	task, err := a.taskService().ArchiveTask(r.Context(), id)
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
	tasks, err := a.taskService().ListArchived(r.Context())
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

	task, err := a.taskService().RestoreTask(r.Context(), id)
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

	err := a.taskService().DeleteArchived(r.Context(), id)
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
