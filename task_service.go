package main

import (
	"context"
	"strings"
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
	id, err := normalizeTaskID(id)
	if err != nil {
		return Task{}, err
	}
	if err := validateTaskPayload(&task, false, false); err != nil {
		return Task{}, err
	}
	return s.repo.UpdateTask(ctx, id, task)
}

func (s defaultTaskService) ReorderTask(ctx context.Context, id string, reorder taskReorderInput) (Task, error) {
	var err error
	id, err = normalizeTaskID(id)
	if err != nil {
		return Task{}, err
	}
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
	id, err := normalizeTaskID(id)
	if err != nil {
		return ArchivedTask{}, err
	}
	return s.repo.ArchiveTask(ctx, id)
}

func (s defaultTaskService) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	return s.repo.ListArchived(ctx)
}

func (s defaultTaskService) RestoreTask(ctx context.Context, id string) (Task, error) {
	id, err := normalizeTaskID(id)
	if err != nil {
		return Task{}, err
	}
	return s.repo.RestoreTask(ctx, id)
}

func (s defaultTaskService) DeleteArchived(ctx context.Context, id string) error {
	id, err := normalizeTaskID(id)
	if err != nil {
		return err
	}
	return s.repo.DeleteArchived(ctx, id)
}
