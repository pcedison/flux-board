package main

import (
	"context"
	"testing"
)

func TestDefaultTaskServiceCreateTaskValidatesPayloadBeforeRepository(t *testing.T) {
	called := false
	service := defaultTaskService{
		repo: stubTaskRepository{
			createTaskFn: func(context.Context, Task) (Task, error) {
				called = true
				return Task{}, nil
			},
		},
	}

	_, err := service.CreateTask(context.Background(), Task{
		ID:       "task-1",
		Title:    "",
		Due:      "2026-04-20",
		Priority: "medium",
	})
	if err == nil || err.Error() != "title is required" {
		t.Fatalf("expected title validation error, got %v", err)
	}
	if called {
		t.Fatal("repository should not be called when validation fails")
	}
}

func TestDefaultTaskServiceReorderTaskRejectsSelfAnchor(t *testing.T) {
	service := defaultTaskService{
		repo: stubTaskRepository{},
	}

	_, err := service.ReorderTask(context.Background(), "task-1", taskReorderInput{
		Status:       "queued",
		AnchorTaskID: "task-1",
		PlaceAfter:   false,
	})
	if err == nil || err.Error() != "anchor task cannot match task id" {
		t.Fatalf("expected self-anchor validation error, got %v", err)
	}
}

func TestDefaultTaskServiceUpdateTaskRejectsMissingID(t *testing.T) {
	service := defaultTaskService{
		repo: stubTaskRepository{},
	}

	_, err := service.UpdateTask(context.Background(), "   ", Task{
		Title:    "Ship tests",
		Due:      "2026-04-20",
		Priority: "medium",
	})
	if err == nil || err.Error() != "missing task id" {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}

func TestDefaultTaskServiceArchiveTaskRejectsMissingID(t *testing.T) {
	service := defaultTaskService{
		repo: stubTaskRepository{},
	}

	_, err := service.ArchiveTask(context.Background(), " ")
	if err == nil || err.Error() != "missing task id" {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}
