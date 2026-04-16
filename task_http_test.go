package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubTaskRepository struct {
	archiveTaskFn   func(context.Context, string) (ArchivedTask, error)
	createTaskFn    func(context.Context, Task) (Task, error)
	deleteArchiveFn func(context.Context, string) error
	listArchivedFn  func(context.Context) ([]ArchivedTask, error)
	listTasksFn     func(context.Context) ([]Task, error)
	reorderTaskFn   func(context.Context, string, taskReorderInput) (Task, error)
	restoreTaskFn   func(context.Context, string) (Task, error)
	updateTaskFn    func(context.Context, string, Task) (Task, error)
}

type stubTaskService struct {
	archiveTaskFn   func(context.Context, string) (ArchivedTask, error)
	createTaskFn    func(context.Context, Task) (Task, error)
	deleteArchiveFn func(context.Context, string) error
	listArchivedFn  func(context.Context) ([]ArchivedTask, error)
	listTasksFn     func(context.Context) ([]Task, error)
	reorderTaskFn   func(context.Context, string, taskReorderInput) (Task, error)
	restoreTaskFn   func(context.Context, string) (Task, error)
	updateTaskFn    func(context.Context, string, Task) (Task, error)
}

func (s stubTaskRepository) ListTasks(ctx context.Context) ([]Task, error) {
	if s.listTasksFn != nil {
		return s.listTasksFn(ctx)
	}
	return nil, nil
}

func (s stubTaskRepository) CreateTask(ctx context.Context, task Task) (Task, error) {
	if s.createTaskFn != nil {
		return s.createTaskFn(ctx, task)
	}
	return task, nil
}

func (s stubTaskRepository) UpdateTask(ctx context.Context, id string, task Task) (Task, error) {
	if s.updateTaskFn != nil {
		return s.updateTaskFn(ctx, id, task)
	}
	return task, nil
}

func (s stubTaskRepository) ReorderTask(ctx context.Context, id string, reorder taskReorderInput) (Task, error) {
	if s.reorderTaskFn != nil {
		return s.reorderTaskFn(ctx, id, reorder)
	}
	return Task{}, nil
}

func (s stubTaskRepository) ArchiveTask(ctx context.Context, id string) (ArchivedTask, error) {
	if s.archiveTaskFn != nil {
		return s.archiveTaskFn(ctx, id)
	}
	return ArchivedTask{}, nil
}

func (s stubTaskRepository) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	if s.listArchivedFn != nil {
		return s.listArchivedFn(ctx)
	}
	return nil, nil
}

func (s stubTaskRepository) RestoreTask(ctx context.Context, id string) (Task, error) {
	if s.restoreTaskFn != nil {
		return s.restoreTaskFn(ctx, id)
	}
	return Task{}, nil
}

func (s stubTaskRepository) DeleteArchived(ctx context.Context, id string) error {
	if s.deleteArchiveFn != nil {
		return s.deleteArchiveFn(ctx, id)
	}
	return nil
}

func (s stubTaskService) ListTasks(ctx context.Context) ([]Task, error) {
	if s.listTasksFn != nil {
		return s.listTasksFn(ctx)
	}
	return nil, nil
}

func (s stubTaskService) CreateTask(ctx context.Context, task Task) (Task, error) {
	if s.createTaskFn != nil {
		return s.createTaskFn(ctx, task)
	}
	return task, nil
}

func (s stubTaskService) UpdateTask(ctx context.Context, id string, task Task) (Task, error) {
	if s.updateTaskFn != nil {
		return s.updateTaskFn(ctx, id, task)
	}
	return task, nil
}

func (s stubTaskService) ReorderTask(ctx context.Context, id string, reorder taskReorderInput) (Task, error) {
	if s.reorderTaskFn != nil {
		return s.reorderTaskFn(ctx, id, reorder)
	}
	return Task{}, nil
}

func (s stubTaskService) ArchiveTask(ctx context.Context, id string) (ArchivedTask, error) {
	if s.archiveTaskFn != nil {
		return s.archiveTaskFn(ctx, id)
	}
	return ArchivedTask{}, nil
}

func (s stubTaskService) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	if s.listArchivedFn != nil {
		return s.listArchivedFn(ctx)
	}
	return nil, nil
}

func (s stubTaskService) RestoreTask(ctx context.Context, id string) (Task, error) {
	if s.restoreTaskFn != nil {
		return s.restoreTaskFn(ctx, id)
	}
	return Task{}, nil
}

func (s stubTaskService) DeleteArchived(ctx context.Context, id string) error {
	if s.deleteArchiveFn != nil {
		return s.deleteArchiveFn(ctx, id)
	}
	return nil
}

func TestHandleGetTasksUsesRepositoryResult(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			listTasksFn: func(context.Context) ([]Task, error) {
				return []Task{{ID: "task-1", Title: "Ship tests", Due: "2026-04-20", Priority: "medium", Status: "queued", SortOrder: 0}}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	app.handleGetTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Tasks) != 1 || body.Tasks[0].ID != "task-1" {
		t.Fatalf("unexpected tasks payload: %+v", body.Tasks)
	}
}

func TestHandleCreateTaskMapsConflictError(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			createTaskFn: func(context.Context, Task) (Task, error) {
				return Task{}, errTaskConflict
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"id":"task-1",
		"title":"Ship tests",
		"note":"",
		"due":"2026-04-20",
		"priority":"medium"
	}`))
	rec := httptest.NewRecorder()
	app.handleCreateTask(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateTaskMapsValidationError(t *testing.T) {
	app := &App{}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"id":"task-1",
		"title":"   ",
		"note":"",
		"due":"2026-04-20",
		"priority":"medium"
	}`))
	rec := httptest.NewRecorder()
	app.handleCreateTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "title is required") {
		t.Fatalf("expected title validation error, got %s", rec.Body.String())
	}
}

func TestHandleCreateTaskUsesInjectedTaskService(t *testing.T) {
	app := &App{
		taskSvc: stubTaskService{
			createTaskFn: func(context.Context, Task) (Task, error) {
				return Task{}, taskValidationError{message: "service validation"}
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"id":"task-1",
		"title":"Ship tests",
		"note":"",
		"due":"2026-04-20",
		"priority":"medium"
	}`))
	rec := httptest.NewRecorder()
	app.handleCreateTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "service validation") {
		t.Fatalf("expected service validation error, got %s", rec.Body.String())
	}
}

func TestHandleRestoreTaskMapsStoredTaskInvalid(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			restoreTaskFn: func(context.Context, string) (Task, error) {
				return Task{}, errStoredTaskInvalid
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/archived/task-1/restore", nil)
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()
	app.handleRestoreTask(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleArchiveTaskMapsMissingIDFromService(t *testing.T) {
	app := &App{}

	req := httptest.NewRequest(http.MethodDelete, "/api/tasks/", nil)
	rec := httptest.NewRecorder()
	app.handleArchiveTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "missing task id") {
		t.Fatalf("expected missing task id error, got %s", rec.Body.String())
	}
}

func TestHandleDeleteArchivedMapsNotFound(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			deleteArchiveFn: func(context.Context, string) error {
				return errTaskNotFound
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/archived/task-1", nil)
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()
	app.handleDeleteArchived(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetArchivedMapsRepositoryFailure(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			listArchivedFn: func(context.Context) ([]ArchivedTask, error) {
				return nil, errors.New("db down")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/archived", nil)
	rec := httptest.NewRecorder()
	app.handleGetArchived(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestHandleReorderTaskMapsInvalidAnchor(t *testing.T) {
	app := &App{
		taskRepo: stubTaskRepository{
			reorderTaskFn: func(context.Context, string, taskReorderInput) (Task, error) {
				return Task{}, errTaskInvalidAnchor
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/reorder", strings.NewReader(`{
		"status":"queued",
		"anchorTaskId":"task-2",
		"placeAfter":false
	}`))
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()

	app.handleReorderTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleReorderTaskMapsValidationError(t *testing.T) {
	app := &App{}

	req := httptest.NewRequest(http.MethodPost, "/api/tasks/task-1/reorder", strings.NewReader(`{
		"status":"blocked"
	}`))
	req.SetPathValue("id", "task-1")
	rec := httptest.NewRecorder()

	app.handleReorderTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid status") {
		t.Fatalf("expected invalid status error, got %s", rec.Body.String())
	}
}
