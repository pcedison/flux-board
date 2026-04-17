package domain

import "context"

type TaskRepository interface {
	ListTasks(context.Context) ([]Task, error)
	CreateTask(context.Context, Task) (Task, error)
	UpdateTask(context.Context, string, Task) (Task, error)
	ReorderTask(context.Context, string, TaskReorderInput) (Task, error)
	ArchiveTask(context.Context, string) (ArchivedTask, error)
	ListArchived(context.Context) ([]ArchivedTask, error)
	RestoreTask(context.Context, string) (Task, error)
	DeleteArchived(context.Context, string) error
}
