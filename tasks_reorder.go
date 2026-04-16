package main

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"

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
	var req taskReorderRequest
	if !decodeJSON(w, r, taskBodyLimit, &req) {
		return
	}
	task, err := a.taskService().ReorderTask(r.Context(), r.PathValue("id"), taskReorderInput{
		Status:       req.Status,
		AnchorTaskID: req.AnchorTaskID,
		PlaceAfter:   req.PlaceAfter,
	})
	if isTaskValidationError(err) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, errTaskNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if errors.Is(err, errTaskInvalidAnchor) {
		writeError(w, http.StatusBadRequest, "anchor task is invalid")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	jsonResp(w, task)
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
