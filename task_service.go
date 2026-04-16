package main

import (
	"context"
	"errors"
	"strings"
	"time"
)

type TaskService interface {
	ListTasks(context.Context) ([]Task, error)
	CreateTask(context.Context, Task) (Task, error)
	UpdateTask(context.Context, string, Task) (Task, error)
	ReorderTask(context.Context, string, taskReorderInput) (Task, error)
	ArchiveTask(context.Context, string) (ArchivedTask, error)
	ListArchived(context.Context) ([]ArchivedTask, error)
	RestoreTask(context.Context, string) (Task, error)
	DeleteArchived(context.Context, string) error
}

type defaultTaskService struct {
	repo TaskRepository
}

type taskValidationError struct {
	message string
}

func (e taskValidationError) Error() string {
	return e.message
}

func (a *App) taskService() TaskService {
	if a.taskSvc != nil {
		return a.taskSvc
	}
	return defaultTaskService{repo: a.taskRepository()}
}

func (s defaultTaskService) ListTasks(ctx context.Context) ([]Task, error) {
	return s.repo.ListTasks(ctx)
}

func (s defaultTaskService) CreateTask(ctx context.Context, task Task) (Task, error) {
	if err := validateTaskPayload(&task, true, false); err != nil {
		return Task{}, err
	}
	return s.repo.CreateTask(ctx, task)
}

func (s defaultTaskService) UpdateTask(ctx context.Context, id string, task Task) (Task, error) {
	if err := validateTaskPayload(&task, false, false); err != nil {
		return Task{}, err
	}
	return s.repo.UpdateTask(ctx, id, task)
}

func (s defaultTaskService) ReorderTask(ctx context.Context, id string, reorder taskReorderInput) (Task, error) {
	reorder.Status = strings.TrimSpace(reorder.Status)
	reorder.AnchorTaskID = strings.TrimSpace(reorder.AnchorTaskID)
	if !validStatus(reorder.Status) {
		return Task{}, taskValidationError{message: "invalid status"}
	}
	if reorder.AnchorTaskID == id {
		return Task{}, taskValidationError{message: "anchor task cannot match task id"}
	}
	return s.repo.ReorderTask(ctx, id, reorder)
}

func (s defaultTaskService) ArchiveTask(ctx context.Context, id string) (ArchivedTask, error) {
	return s.repo.ArchiveTask(ctx, id)
}

func (s defaultTaskService) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	return s.repo.ListArchived(ctx)
}

func (s defaultTaskService) RestoreTask(ctx context.Context, id string) (Task, error) {
	return s.repo.RestoreTask(ctx, id)
}

func (s defaultTaskService) DeleteArchived(ctx context.Context, id string) error {
	return s.repo.DeleteArchived(ctx, id)
}

func validateTaskPayload(task *Task, requireID bool, allowStatus bool) error {
	if requireID {
		task.ID = strings.TrimSpace(task.ID)
		if task.ID == "" {
			return taskValidationError{message: "id is required"}
		}
	}

	task.Title = strings.TrimSpace(task.Title)
	task.Note = strings.TrimSpace(task.Note)

	if task.Title == "" {
		return taskValidationError{message: "title is required"}
	}
	if len(task.Title) > maxTitleLength {
		return taskValidationError{message: "title is too long"}
	}
	if len(task.Note) > maxNoteLength {
		return taskValidationError{message: "note is too long"}
	}
	if !validDueDate(task.Due) {
		return taskValidationError{message: "due must be in YYYY-MM-DD format"}
	}

	if task.Priority == "" {
		task.Priority = "medium"
	}
	if !validPriority(task.Priority) {
		return taskValidationError{message: "invalid priority"}
	}

	if allowStatus {
		if !validStatus(task.Status) {
			return taskValidationError{message: "invalid status"}
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

func isTaskValidationError(err error) bool {
	var validationErr taskValidationError
	return errors.As(err, &validationErr)
}
