package main

import (
	"context"
	"testing"

	"flux-board/internal/domain"
	taskservice "flux-board/internal/service/task"
)

func TestDefaultTaskServiceCreateTaskValidatesPayloadBeforeRepository(t *testing.T) {
	called := false
	service := taskservice.New(stubTaskRepository{
		createTaskFn: func(context.Context, domain.Task) (domain.Task, error) {
			called = true
			return domain.Task{}, nil
		},
	})

	_, err := service.CreateTask(context.Background(), domain.Task{
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
	service := taskservice.New(stubTaskRepository{})

	_, err := service.ReorderTask(context.Background(), "task-1", domain.TaskReorderInput{
		Status:       "queued",
		AnchorTaskID: "task-1",
		PlaceAfter:   false,
	})
	if err == nil || err.Error() != "anchor task cannot match task id" {
		t.Fatalf("expected self-anchor validation error, got %v", err)
	}
}

func TestDefaultTaskServiceUpdateTaskRejectsMissingID(t *testing.T) {
	service := taskservice.New(stubTaskRepository{})

	_, err := service.UpdateTask(context.Background(), "   ", domain.Task{
		Title:    "Ship tests",
		Due:      "2026-04-20",
		Priority: "medium",
	})
	if err == nil || err.Error() != "missing task id" {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}

func TestDefaultTaskServiceArchiveTaskRejectsMissingID(t *testing.T) {
	service := taskservice.New(stubTaskRepository{})

	_, err := service.ArchiveTask(context.Background(), " ")
	if err == nil || err.Error() != "missing task id" {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}
