package task

import (
	"context"
	"strings"

	"flux-board/internal/domain"
)

type Service interface {
	ListTasks(context.Context) ([]domain.Task, error)
	CreateTask(context.Context, domain.Task) (domain.Task, error)
	UpdateTask(context.Context, string, domain.Task) (domain.Task, error)
	ReorderTask(context.Context, string, domain.TaskReorderInput) (domain.Task, error)
	ArchiveTask(context.Context, string) (domain.ArchivedTask, error)
	ListArchived(context.Context) ([]domain.ArchivedTask, error)
	RestoreTask(context.Context, string) (domain.Task, error)
	DeleteArchived(context.Context, string) error
}

type service struct {
	repo domain.TaskRepository
}

func New(repo domain.TaskRepository) Service {
	return service{repo: repo}
}

func (s service) ListTasks(ctx context.Context) ([]domain.Task, error) {
	return s.repo.ListTasks(ctx)
}

func (s service) CreateTask(ctx context.Context, task domain.Task) (domain.Task, error) {
	if err := domain.ValidateTaskPayload(&task, true, false); err != nil {
		return domain.Task{}, err
	}
	return s.repo.CreateTask(ctx, task)
}

func (s service) UpdateTask(ctx context.Context, id string, task domain.Task) (domain.Task, error) {
	var err error
	id, err = domain.NormalizeTaskID(id)
	if err != nil {
		return domain.Task{}, err
	}
	if err := domain.ValidateTaskPayload(&task, false, false); err != nil {
		return domain.Task{}, err
	}
	return s.repo.UpdateTask(ctx, id, task)
}

func (s service) ReorderTask(ctx context.Context, id string, reorder domain.TaskReorderInput) (domain.Task, error) {
	var err error
	id, err = domain.NormalizeTaskID(id)
	if err != nil {
		return domain.Task{}, err
	}
	reorder.Status = strings.TrimSpace(reorder.Status)
	reorder.AnchorTaskID = strings.TrimSpace(reorder.AnchorTaskID)
	if !domain.ValidStatus(reorder.Status) {
		return domain.Task{}, domain.NewTaskValidationError("invalid status")
	}
	if reorder.AnchorTaskID == id {
		return domain.Task{}, domain.NewTaskValidationError("anchor task cannot match task id")
	}
	return s.repo.ReorderTask(ctx, id, reorder)
}

func (s service) ArchiveTask(ctx context.Context, id string) (domain.ArchivedTask, error) {
	var err error
	id, err = domain.NormalizeTaskID(id)
	if err != nil {
		return domain.ArchivedTask{}, err
	}
	return s.repo.ArchiveTask(ctx, id)
}

func (s service) ListArchived(ctx context.Context) ([]domain.ArchivedTask, error) {
	return s.repo.ListArchived(ctx)
}

func (s service) RestoreTask(ctx context.Context, id string) (domain.Task, error) {
	var err error
	id, err = domain.NormalizeTaskID(id)
	if err != nil {
		return domain.Task{}, err
	}
	return s.repo.RestoreTask(ctx, id)
}

func (s service) DeleteArchived(ctx context.Context, id string) error {
	var err error
	id, err = domain.NormalizeTaskID(id)
	if err != nil {
		return err
	}
	return s.repo.DeleteArchived(ctx, id)
}
