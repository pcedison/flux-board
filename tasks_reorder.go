package main

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const taskOrderConstraintName = "tasks_status_sort_order_unique"

var taskLaneLockIDs = map[string]int64{
	"active": 31_002,
	"done":   31_003,
	"queued": 31_001,
}

type taskReorderRequest struct {
	Status       string `json:"status"`
	AnchorTaskID string `json:"anchorTaskId"`
	PlaceAfter   bool   `json:"placeAfter"`
}

func (a *App) handleReorderTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	var req taskReorderRequest
	if !decodeJSON(w, r, taskBodyLimit, &req) {
		return
	}
	if !validStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if req.AnchorTaskID == id {
		writeError(w, http.StatusBadRequest, "anchor task cannot match task id")
		return
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer tx.Rollback(r.Context())

	var current Task
	err = tx.QueryRow(r.Context(), `
		SELECT id, title, note, due, priority, status, sort_order, last_updated
		FROM tasks
		WHERE id=$1
	`, id).Scan(
		&current.ID,
		&current.Title,
		&current.Note,
		&current.Due,
		&current.Priority,
		&current.Status,
		&current.SortOrder,
		&current.LastUpdated,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	if err := lockTaskLanes(r.Context(), tx, current.Status, req.Status); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := deferTaskOrderConstraint(r.Context(), tx); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	sourceIDs, err := laneTaskIDs(r.Context(), tx, current.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	sourceIDs = removeTaskID(sourceIDs, current.ID)

	targetIDs := sourceIDs
	if current.Status != req.Status {
		targetIDs, err = laneTaskIDs(r.Context(), tx, req.Status)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}

	insertIdx, err := reorderInsertIndex(r.Context(), tx, req.Status, req.AnchorTaskID, req.PlaceAfter, targetIDs)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusBadRequest, "anchor task is invalid")
			return
		}
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	targetIDs = insertAt(targetIDs, insertIdx, current.ID)

	now := time.Now().UnixMilli()
	if current.Status != req.Status {
		if err := applyLaneOrder(r.Context(), tx, current.Status, sourceIDs, "", 0); err != nil {
			writeError(w, http.StatusInternalServerError, "db error")
			return
		}
	}
	if err := applyLaneOrder(r.Context(), tx, req.Status, targetIDs, current.ID, now); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	current.Status = req.Status
	current.SortOrder = insertIdx
	current.LastUpdated = now
	jsonResp(w, current)
}

func lockTaskLanes(ctx context.Context, tx pgx.Tx, statuses ...string) error {
	unique := uniqueStatuses(statuses)
	for _, status := range unique {
		lockID, ok := taskLaneLockIDs[status]
		if !ok {
			return errors.New("invalid status")
		}
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, lockID); err != nil {
			return err
		}
	}
	return nil
}

func uniqueStatuses(statuses []string) []string {
	seen := make(map[string]struct{}, len(statuses))
	unique := make([]string, 0, len(statuses))
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status == "" {
			continue
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		unique = append(unique, status)
	}
	sort.Strings(unique)
	return unique
}

func deferTaskOrderConstraint(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SET CONSTRAINTS `+taskOrderConstraintName+` DEFERRED`)
	return err
}

func nextLaneSortOrder(ctx context.Context, tx pgx.Tx, status string) (int, error) {
	var count int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE status=$1`, status).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func laneTaskIDs(ctx context.Context, tx pgx.Tx, status string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT id
		FROM tasks
		WHERE status=$1
		ORDER BY sort_order ASC, last_updated DESC, id ASC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func reorderInsertIndex(ctx context.Context, tx pgx.Tx, targetStatus, anchorTaskID string, placeAfter bool, currentTargetIDs []string) (int, error) {
	if strings.TrimSpace(anchorTaskID) == "" {
		return len(currentTargetIDs), nil
	}

	var anchorStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM tasks WHERE id=$1`, anchorTaskID).Scan(&anchorStatus); err != nil {
		return 0, err
	}
	if anchorStatus != targetStatus {
		return 0, pgx.ErrNoRows
	}

	for idx, id := range currentTargetIDs {
		if id != anchorTaskID {
			continue
		}
		if placeAfter {
			return idx + 1, nil
		}
		return idx, nil
	}

	return len(currentTargetIDs), nil
}

func applyLaneOrder(ctx context.Context, tx pgx.Tx, status string, ids []string, touchedID string, touchedAt int64) error {
	for idx, id := range ids {
		if id == touchedID && touchedAt > 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE tasks
				SET status=$1, sort_order=$2, last_updated=$3
				WHERE id=$4
			`, status, idx, touchedAt, id); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tasks
			SET status=$1, sort_order=$2
			WHERE id=$3
		`, status, idx, id); err != nil {
			return err
		}
	}
	return nil
}

func insertAt(ids []string, index int, value string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(ids) {
		index = len(ids)
	}
	ids = append(ids, "")
	copy(ids[index+1:], ids[index:])
	ids[index] = value
	return ids
}

func removeTaskID(ids []string, taskID string) []string {
	filtered := ids[:0]
	for _, id := range ids {
		if id == taskID {
			continue
		}
		filtered = append(filtered, id)
	}
	return filtered
}
