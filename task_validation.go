package main

import (
	"errors"
	"strings"
	"time"
)

type taskValidationError struct {
	message string
}

func (e taskValidationError) Error() string {
	return e.message
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

func normalizeTaskID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", taskValidationError{message: "missing task id"}
	}
	return id, nil
}
