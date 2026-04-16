package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (a *App) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(),
		`SELECT id, title, note, due, priority, status, sort_order, last_updated
		 FROM tasks
		 ORDER BY status, sort_order, last_updated DESC, id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	tasks := make([]Task, 0)
	for rows.Next() {
		var task Task
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Note,
			&task.Due,
			&task.Priority,
			&task.Status,
			&task.SortOrder,
			&task.LastUpdated,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
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

	task.Status = "queued"
	task.LastUpdated = time.Now().UnixMilli()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	if err := lockTaskLanes(r.Context(), tx, task.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.SortOrder, err = nextLaneSortOrder(r.Context(), tx, task.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	tag, err := tx.Exec(r.Context(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (id) DO NOTHING`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "task id already exists")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonResp(w, task)
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

	var currentStatus string
	var currentSortOrder int
	if err := a.db.QueryRow(r.Context(),
		`SELECT status, sort_order FROM tasks WHERE id=$1`, id,
	).Scan(&currentStatus, &currentSortOrder); errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.LastUpdated = time.Now().UnixMilli()
	if _, err := a.db.Exec(r.Context(),
		`UPDATE tasks
		 SET title=$1, note=$2, due=$3, priority=$4, last_updated=$5
		 WHERE id=$6`,
		task.Title, task.Note, task.Due, task.Priority, task.LastUpdated, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.ID = id
	task.Status = currentStatus
	task.SortOrder = currentSortOrder
	jsonResp(w, task)
}

// handleArchiveTask moves a task from tasks to archived_tasks.
func (a *App) handleArchiveTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	now := time.Now().UnixMilli()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	var task ArchivedTask
	err = tx.QueryRow(r.Context(),
		`SELECT id, title, note, due, priority, status, sort_order
		 FROM tasks
		 WHERE id=$1
		 FOR UPDATE`, id).
		Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := lockTaskLanes(r.Context(), tx, task.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := deferTaskOrderConstraint(r.Context(), tx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if _, err := tx.Exec(r.Context(), `DELETE FROM tasks WHERE id=$1`, id); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.ArchivedAt = now
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO archived_tasks (id, title, note, due, priority, status, sort_order, archived_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.ArchivedAt); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	remainingIDs, err := laneTaskIDs(r.Context(), tx, task.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := applyLaneOrder(r.Context(), tx, task.Status, remainingIDs, "", 0); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
}

func (a *App) handleGetArchived(w http.ResponseWriter, r *http.Request) {
	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
	if _, err := a.db.Exec(r.Context(),
		`DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	rows, err := a.db.Query(r.Context(),
		`SELECT id, title, note, due, priority, status, sort_order, archived_at
		 FROM archived_tasks
		 ORDER BY archived_at DESC, id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	tasks := make([]ArchivedTask, 0)
	for rows.Next() {
		var task ArchivedTask
		if err := rows.Scan(
			&task.ID,
			&task.Title,
			&task.Note,
			&task.Due,
			&task.Priority,
			&task.Status,
			&task.SortOrder,
			&task.ArchivedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
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

	now := time.Now().UnixMilli()
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	var task Task
	err = tx.QueryRow(r.Context(),
		`DELETE FROM archived_tasks WHERE id=$1
		 RETURNING id, title, note, due, priority, status, sort_order`, id).
		Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if err == pgx.ErrNoRows {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := validateTaskPayload(&task, true, true); err != nil {
		writeError(w, http.StatusInternalServerError, "stored task is invalid")
		return
	}

	task.LastUpdated = now
	if err := lockTaskLanes(r.Context(), tx, task.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := deferTaskOrderConstraint(r.Context(), tx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	laneIDs, err := laneTaskIDs(r.Context(), tx, task.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	insertIdx := task.SortOrder
	if insertIdx < 0 {
		insertIdx = 0
	}
	if insertIdx > len(laneIDs) {
		insertIdx = len(laneIDs)
	}
	laneIDs = insertAt(laneIDs, insertIdx, task.ID)

	if _, err := tx.Exec(r.Context(),
		`INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, insertIdx, task.LastUpdated); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := applyLaneOrder(r.Context(), tx, task.Status, laneIDs, task.ID, task.LastUpdated); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	task.SortOrder = insertIdx

	if err := tx.Commit(r.Context()); err != nil {
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

	tag, err := a.db.Exec(r.Context(),
		`DELETE FROM archived_tasks WHERE id=$1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not found")
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
