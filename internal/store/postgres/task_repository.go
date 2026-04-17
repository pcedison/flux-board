package postgres

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"flux-board/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const taskOrderConstraintName = "tasks_status_sort_order_unique"

var taskLaneLockIDs = map[string]int64{
	"active": 31_002,
	"done":   31_003,
	"queued": 31_001,
}

type TaskRepository struct {
	db               *pgxpool.Pool
	archiveRetention time.Duration
}

func NewTaskRepository(db *pgxpool.Pool, archiveRetention time.Duration) *TaskRepository {
	return &TaskRepository{
		db:               db,
		archiveRetention: archiveRetention,
	}
}

func (r *TaskRepository) ListTasks(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, note, due, priority, status, sort_order, last_updated
		FROM tasks
		ORDER BY status, sort_order, last_updated DESC, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0)
	for rows.Next() {
		var task domain.Task
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
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	task.Status = "queued"
	task.LastUpdated = time.Now().UnixMilli()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return domain.Task{}, err
	}

	task.SortOrder, err = nextLaneSortOrder(ctx, tx, task.Status)
	if err != nil {
		return domain.Task{}, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated)
	if err != nil {
		return domain.Task{}, err
	}
	if tag.RowsAffected() == 0 {
		return domain.Task{}, domain.ErrTaskConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, err
	}

	return task, nil
}

func (r *TaskRepository) UpdateTask(ctx context.Context, id string, task domain.Task) (domain.Task, error) {
	var currentStatus string
	var currentSortOrder int
	if err := r.db.QueryRow(ctx, `SELECT status, sort_order FROM tasks WHERE id=$1`, id).Scan(&currentStatus, &currentSortOrder); errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	} else if err != nil {
		return domain.Task{}, err
	}

	task.LastUpdated = time.Now().UnixMilli()
	if _, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET title=$1, note=$2, due=$3, priority=$4, last_updated=$5
		WHERE id=$6
	`, task.Title, task.Note, task.Due, task.Priority, task.LastUpdated, id); err != nil {
		return domain.Task{}, err
	}

	task.ID = id
	task.Status = currentStatus
	task.SortOrder = currentSortOrder
	return task, nil
}

func (r *TaskRepository) ReorderTask(ctx context.Context, id string, reorder domain.TaskReorderInput) (domain.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	defer tx.Rollback(ctx)

	var current domain.Task
	err = tx.QueryRow(ctx, `
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
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, err
	}

	if err := lockTaskLanes(ctx, tx, current.Status, reorder.Status); err != nil {
		return domain.Task{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return domain.Task{}, err
	}

	sourceIDs, err := laneTaskIDs(ctx, tx, current.Status)
	if err != nil {
		return domain.Task{}, err
	}
	sourceIDs = removeTaskID(sourceIDs, current.ID)

	targetIDs := sourceIDs
	if current.Status != reorder.Status {
		targetIDs, err = laneTaskIDs(ctx, tx, reorder.Status)
		if err != nil {
			return domain.Task{}, err
		}
	}

	insertIdx, err := reorderInsertIndex(ctx, tx, reorder.Status, reorder.AnchorTaskID, reorder.PlaceAfter, targetIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskInvalidAnchor
	}
	if err != nil {
		return domain.Task{}, err
	}
	targetIDs = insertAt(targetIDs, insertIdx, current.ID)

	now := time.Now().UnixMilli()
	if current.Status != reorder.Status {
		if err := applyLaneOrder(ctx, tx, current.Status, sourceIDs, "", 0); err != nil {
			return domain.Task{}, err
		}
	}
	if err := applyLaneOrder(ctx, tx, reorder.Status, targetIDs, current.ID, now); err != nil {
		return domain.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, err
	}

	current.Status = reorder.Status
	current.SortOrder = insertIdx
	current.LastUpdated = now
	return current, nil
}

func (r *TaskRepository) ArchiveTask(ctx context.Context, id string) (domain.ArchivedTask, error) {
	now := time.Now().UnixMilli()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.ArchivedTask{}, err
	}
	defer tx.Rollback(ctx)

	var task domain.ArchivedTask
	err = tx.QueryRow(ctx, `
		SELECT id, title, note, due, priority, status, sort_order
		FROM tasks
		WHERE id=$1
		FOR UPDATE
	`, id).Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ArchivedTask{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.ArchivedTask{}, err
	}

	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return domain.ArchivedTask{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return domain.ArchivedTask{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id=$1`, id); err != nil {
		return domain.ArchivedTask{}, err
	}

	task.ArchivedAt = now
	if _, err := tx.Exec(ctx, `
		INSERT INTO archived_tasks (id, title, note, due, priority, status, sort_order, archived_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.ArchivedAt); err != nil {
		return domain.ArchivedTask{}, err
	}

	remainingIDs, err := laneTaskIDs(ctx, tx, task.Status)
	if err != nil {
		return domain.ArchivedTask{}, err
	}
	if err := applyLaneOrder(ctx, tx, task.Status, remainingIDs, "", 0); err != nil {
		return domain.ArchivedTask{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ArchivedTask{}, err
	}
	return task, nil
}

func (r *TaskRepository) ListArchived(ctx context.Context) ([]domain.ArchivedTask, error) {
	cutoff := time.Now().Add(-r.archiveRetention).UnixMilli()
	if _, err := r.db.Exec(ctx, `DELETE FROM archived_tasks WHERE archived_at < $1`, cutoff); err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, title, note, due, priority, status, sort_order, archived_at
		FROM archived_tasks
		ORDER BY archived_at DESC, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]domain.ArchivedTask, 0)
	for rows.Next() {
		var task domain.ArchivedTask
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
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r *TaskRepository) RestoreTask(ctx context.Context, id string) (domain.Task, error) {
	now := time.Now().UnixMilli()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Task{}, err
	}
	defer tx.Rollback(ctx)

	var task domain.Task
	err = tx.QueryRow(ctx, `
		DELETE FROM archived_tasks WHERE id=$1
		RETURNING id, title, note, due, priority, status, sort_order
	`, id).Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Task{}, domain.ErrTaskNotFound
	}
	if err != nil {
		return domain.Task{}, err
	}

	if err := domain.ValidateTaskPayload(&task, true, true); err != nil {
		return domain.Task{}, domain.ErrStoredTaskInvalid
	}

	task.LastUpdated = now
	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return domain.Task{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return domain.Task{}, err
	}

	laneIDs, err := laneTaskIDs(ctx, tx, task.Status)
	if err != nil {
		return domain.Task{}, err
	}

	insertIdx := task.SortOrder
	if insertIdx < 0 {
		insertIdx = 0
	}
	if insertIdx > len(laneIDs) {
		insertIdx = len(laneIDs)
	}
	laneIDs = insertAt(laneIDs, insertIdx, task.ID)

	if _, err := tx.Exec(ctx, `
		INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, insertIdx, task.LastUpdated); err != nil {
		return domain.Task{}, err
	}

	if err := applyLaneOrder(ctx, tx, task.Status, laneIDs, task.ID, task.LastUpdated); err != nil {
		return domain.Task{}, err
	}

	task.SortOrder = insertIdx
	if err := tx.Commit(ctx); err != nil {
		return domain.Task{}, err
	}
	return task, nil
}

func (r *TaskRepository) DeleteArchived(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM archived_tasks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTaskNotFound
	}
	return nil
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
