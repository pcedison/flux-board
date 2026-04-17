package main

import (
	"testing"

	"flux-board/internal/domain"
)

func TestValidateTaskPayloadTrimsAndDefaultsTask(t *testing.T) {
	task := domain.Task{
		ID:       " task-1 ",
		Title:    " Ship tests ",
		Note:     " note ",
		Due:      "2026-04-20",
		Priority: "",
	}

	if err := domain.ValidateTaskPayload(&task, true, false); err != nil {
		t.Fatalf("expected valid payload, got %v", err)
	}
	if task.ID != "task-1" {
		t.Fatalf("expected trimmed id, got %q", task.ID)
	}
	if task.Title != "Ship tests" {
		t.Fatalf("expected trimmed title, got %q", task.Title)
	}
	if task.Note != "note" {
		t.Fatalf("expected trimmed note, got %q", task.Note)
	}
	if task.Priority != "medium" {
		t.Fatalf("expected default priority, got %q", task.Priority)
	}
	if task.Status != "queued" {
		t.Fatalf("expected default status queued, got %q", task.Status)
	}
}

func TestValidateTaskPayloadRejectsInvalidDueDateFormat(t *testing.T) {
	task := domain.Task{
		ID:       "task-1",
		Title:    "Ship tests",
		Due:      "2026/04/20",
		Priority: "medium",
	}

	err := domain.ValidateTaskPayload(&task, true, false)
	if err == nil || err.Error() != "due must be in YYYY-MM-DD format" {
		t.Fatalf("expected due date validation error, got %v", err)
	}
}

func TestValidateTaskPayloadRejectsInvalidPriority(t *testing.T) {
	task := domain.Task{
		ID:       "task-1",
		Title:    "Ship tests",
		Due:      "2026-04-20",
		Priority: "low",
	}

	err := domain.ValidateTaskPayload(&task, true, false)
	if err == nil || err.Error() != "invalid priority" {
		t.Fatalf("expected invalid priority error, got %v", err)
	}
}

func TestNormalizeTaskIDRejectsBlankID(t *testing.T) {
	_, err := domain.NormalizeTaskID(" \t")
	if err == nil || err.Error() != "missing task id" {
		t.Fatalf("expected missing task id error, got %v", err)
	}
}
