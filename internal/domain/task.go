package domain

import (
	"errors"
	"strings"
	"time"
)

const (
	maxTitleLength = 120
	maxNoteLength  = 4000
)

// Task represents an active task in the kanban board.
type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Note        string `json:"note"`
	Due         string `json:"due"`
	Priority    string `json:"priority"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	LastUpdated int64  `json:"lastUpdated"`
}

// ArchivedTask represents a soft-deleted task.
type ArchivedTask struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Note       string `json:"note"`
	Due        string `json:"due"`
	Priority   string `json:"priority"`
	Status     string `json:"status"`
	SortOrder  int    `json:"sort_order"`
	ArchivedAt int64  `json:"archivedAt"`
}

// TaskReorderInput captures the target lane and anchor for a reorder.
type TaskReorderInput struct {
	Status       string
	AnchorTaskID string
	PlaceAfter   bool
}

var (
	ErrTaskConflict      = errors.New("task conflict")
	ErrTaskInvalidAnchor = errors.New("task invalid anchor")
	ErrTaskNotFound      = errors.New("task not found")
	ErrStoredTaskInvalid = errors.New("stored task is invalid")
)

type taskValidationError struct {
	message string
}

func (e taskValidationError) Error() string {
	return e.message
}

func NewTaskValidationError(message string) error {
	return taskValidationError{message: message}
}

func ValidateTaskPayload(task *Task, requireID bool, allowStatus bool) error {
	if requireID {
		task.ID = strings.TrimSpace(task.ID)
		if task.ID == "" {
			return NewTaskValidationError("id is required")
		}
	}

	task.Title = strings.TrimSpace(task.Title)
	task.Note = strings.TrimSpace(task.Note)

	if task.Title == "" {
		return NewTaskValidationError("title is required")
	}
	if len(task.Title) > maxTitleLength {
		return NewTaskValidationError("title is too long")
	}
	if len(task.Note) > maxNoteLength {
		return NewTaskValidationError("note is too long")
	}
	if !validDueDate(task.Due) {
		return NewTaskValidationError("due must be in YYYY-MM-DD format")
	}

	if task.Priority == "" {
		task.Priority = "medium"
	}
	if !validPriority(task.Priority) {
		return NewTaskValidationError("invalid priority")
	}

	if allowStatus {
		if !ValidStatus(task.Status) {
			return NewTaskValidationError("invalid status")
		}
	} else {
		task.Status = "queued"
	}

	return nil
}

func ValidStatus(status string) bool {
	return status == "queued" || status == "active" || status == "done"
}

func IsTaskValidationError(err error) bool {
	var validationErr taskValidationError
	return errors.As(err, &validationErr)
}

func NormalizeTaskID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", NewTaskValidationError("missing task id")
	}
	return id, nil
}

func validPriority(priority string) bool {
	return priority == "critical" || priority == "high" || priority == "medium"
}

func validDueDate(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}
