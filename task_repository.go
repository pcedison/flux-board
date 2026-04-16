package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errTaskConflict      = errors.New("task conflict")
	errTaskInvalidAnchor = errors.New("task invalid anchor")
	errTaskNotFound      = errors.New("task not found")
	errStoredTaskInvalid = errors.New("stored task is invalid")
)

type taskReorderInput struct {
	Status       string
	AnchorTaskID string
	PlaceAfter   bool
}

type TaskRepository interface {
	ListTasks(context.Context) ([]Task, error)
	CreateTask(context.Context, Task) (Task, error)
	UpdateTask(context.Context, string, Task) (Task, error)
	ReorderTask(context.Context, string, taskReorderInput) (Task, error)
	ArchiveTask(context.Context, string) (ArchivedTask, error)
	ListArchived(context.Context) ([]ArchivedTask, error)
	RestoreTask(context.Context, string) (Task, error)
	DeleteArchived(context.Context, string) error
}

type postgresTaskRepository struct {
	db *pgxpool.Pool
}

func (a *App) taskRepository() TaskRepository {
	if a.taskRepo != nil {
		return a.taskRepo
	}
	return postgresTaskRepository{db: a.db}
}

func (r postgresTaskRepository) ListTasks(ctx context.Context) ([]Task, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, note, due, priority, status, sort_order, last_updated
		FROM tasks
		ORDER BY status, sort_order, last_updated DESC, id
	`)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r postgresTaskRepository) CreateTask(ctx context.Context, task Task) (Task, error) {
	task.Status = "queued"
	task.LastUpdated = time.Now().UnixMilli()

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)

	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return Task{}, err
	}

	task.SortOrder, err = nextLaneSortOrder(ctx, tx, task.Status)
	if err != nil {
		return Task{}, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO tasks (id, title, note, due, priority, status, sort_order, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.LastUpdated)
	if err != nil {
		return Task{}, err
	}
	if tag.RowsAffected() == 0 {
		return Task{}, errTaskConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, err
	}

	return task, nil
}

func (r postgresTaskRepository) UpdateTask(ctx context.Context, id string, task Task) (Task, error) {
	var currentStatus string
	var currentSortOrder int
	if err := r.db.QueryRow(ctx, `SELECT status, sort_order FROM tasks WHERE id=$1`, id).Scan(&currentStatus, &currentSortOrder); errors.Is(err, pgx.ErrNoRows) {
		return Task{}, errTaskNotFound
	} else if err != nil {
		return Task{}, err
	}

	task.LastUpdated = time.Now().UnixMilli()
	if _, err := r.db.Exec(ctx, `
		UPDATE tasks
		SET title=$1, note=$2, due=$3, priority=$4, last_updated=$5
		WHERE id=$6
	`, task.Title, task.Note, task.Due, task.Priority, task.LastUpdated, id); err != nil {
		return Task{}, err
	}

	task.ID = id
	task.Status = currentStatus
	task.SortOrder = currentSortOrder
	return task, nil
}

func (r postgresTaskRepository) ReorderTask(ctx context.Context, id string, reorder taskReorderInput) (Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)

	var current Task
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
		return Task{}, errTaskNotFound
	}
	if err != nil {
		return Task{}, err
	}

	if err := lockTaskLanes(ctx, tx, current.Status, reorder.Status); err != nil {
		return Task{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return Task{}, err
	}

	sourceIDs, err := laneTaskIDs(ctx, tx, current.Status)
	if err != nil {
		return Task{}, err
	}
	sourceIDs = removeTaskID(sourceIDs, current.ID)

	targetIDs := sourceIDs
	if current.Status != reorder.Status {
		targetIDs, err = laneTaskIDs(ctx, tx, reorder.Status)
		if err != nil {
			return Task{}, err
		}
	}

	insertIdx, err := reorderInsertIndex(ctx, tx, reorder.Status, reorder.AnchorTaskID, reorder.PlaceAfter, targetIDs)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, errTaskInvalidAnchor
	}
	if err != nil {
		return Task{}, err
	}
	targetIDs = insertAt(targetIDs, insertIdx, current.ID)

	now := time.Now().UnixMilli()
	if current.Status != reorder.Status {
		if err := applyLaneOrder(ctx, tx, current.Status, sourceIDs, "", 0); err != nil {
			return Task{}, err
		}
	}
	if err := applyLaneOrder(ctx, tx, reorder.Status, targetIDs, current.ID, now); err != nil {
		return Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Task{}, err
	}

	current.Status = reorder.Status
	current.SortOrder = insertIdx
	current.LastUpdated = now
	return current, nil
}

func (r postgresTaskRepository) ArchiveTask(ctx context.Context, id string) (ArchivedTask, error) {
	now := time.Now().UnixMilli()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return ArchivedTask{}, err
	}
	defer tx.Rollback(ctx)

	var task ArchivedTask
	err = tx.QueryRow(ctx, `
		SELECT id, title, note, due, priority, status, sort_order
		FROM tasks
		WHERE id=$1
		FOR UPDATE
	`, id).Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return ArchivedTask{}, errTaskNotFound
	}
	if err != nil {
		return ArchivedTask{}, err
	}

	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return ArchivedTask{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return ArchivedTask{}, err
	}

	if _, err := tx.Exec(ctx, `DELETE FROM tasks WHERE id=$1`, id); err != nil {
		return ArchivedTask{}, err
	}

	task.ArchivedAt = now
	if _, err := tx.Exec(ctx, `
		INSERT INTO archived_tasks (id, title, note, due, priority, status, sort_order, archived_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, task.ID, task.Title, task.Note, task.Due, task.Priority, task.Status, task.SortOrder, task.ArchivedAt); err != nil {
		return ArchivedTask{}, err
	}

	remainingIDs, err := laneTaskIDs(ctx, tx, task.Status)
	if err != nil {
		return ArchivedTask{}, err
	}
	if err := applyLaneOrder(ctx, tx, task.Status, remainingIDs, "", 0); err != nil {
		return ArchivedTask{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ArchivedTask{}, err
	}
	return task, nil
}

func (r postgresTaskRepository) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	cutoff := time.Now().Add(-archiveRetention).UnixMilli()
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
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tasks, nil
}

func (r postgresTaskRepository) RestoreTask(ctx context.Context, id string) (Task, error) {
	now := time.Now().UnixMilli()
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)

	var task Task
	err = tx.QueryRow(ctx, `
		DELETE FROM archived_tasks WHERE id=$1
		RETURNING id, title, note, due, priority, status, sort_order
	`, id).Scan(&task.ID, &task.Title, &task.Note, &task.Due, &task.Priority, &task.Status, &task.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, errTaskNotFound
	}
	if err != nil {
		return Task{}, err
	}

	if err := validateTaskPayload(&task, true, true); err != nil {
		return Task{}, errStoredTaskInvalid
	}

	task.LastUpdated = now
	if err := lockTaskLanes(ctx, tx, task.Status); err != nil {
		return Task{}, err
	}
	if err := deferTaskOrderConstraint(ctx, tx); err != nil {
		return Task{}, err
	}

	laneIDs, err := laneTaskIDs(ctx, tx, task.Status)
	if err != nil {
		return Task{}, err
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
		return Task{}, err
	}

	if err := applyLaneOrder(ctx, tx, task.Status, laneIDs, task.ID, task.LastUpdated); err != nil {
		return Task{}, err
	}

	task.SortOrder = insertIdx
	if err := tx.Commit(ctx); err != nil {
		return Task{}, err
	}
	return task, nil
}

func (r postgresTaskRepository) DeleteArchived(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM archived_tasks WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errTaskNotFound
	}
	return nil
}
