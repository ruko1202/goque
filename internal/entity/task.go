// Package entity contains domain entities for the task queue system.
package entity

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/utils/xtime"
)

// TaskType represents the type of task to be executed.
type TaskType = string

// TaskStatus represents the current status of a task.
type TaskStatus = string

// Task status constants define the possible states of a task in the queue.
const (
	// TaskStatusNew is a new task.
	TaskStatusNew = "new"
	// TaskStatusPending is a task waiting to be processed.
	TaskStatusPending = "pending"
	// TaskStatusProcessing is a task in progress.
	TaskStatusProcessing = "processing"
	// TaskStatusDone is a task that was processed successfully.
	TaskStatusDone = "done"
	// TaskStatusCanceled is a canceled task.
	TaskStatusCanceled = "canceled"
	// TaskStatusError is a task is processed with an error and HAS attempts.
	TaskStatusError = "error"
	// TaskStatusAttemptsLeft is a task that was processed with an error and NO attempts.
	TaskStatusAttemptsLeft = "attempts_left"
)

// NoTaskPayload is an empty JSON object payload for tasks without input data.
const NoTaskPayload = "{}"

// Task represents a unit of work in the queue system.
type Task struct {
	ID            uuid.UUID
	Type          TaskType
	ExternalID    string
	Payload       string
	Status        TaskStatus
	Attempts      int32
	Errors        *string
	Metadata      Metadata
	CreatedAt     time.Time
	UpdatedAt     *time.Time
	NextAttemptAt time.Time
}

// NewTask creates a new task with the specified type and payload.
func NewTask(taskType TaskType, payload string) *Task {
	return NewTaskWithExternalID(taskType, payload, "")
}

// NewTaskWithExternalID creates a new task with a custom external ID.
func NewTaskWithExternalID(taskType, payload, externalID string) *Task {
	if externalID == "" {
		externalID = "internal-" + uuid.NewString()
	}

	now := xtime.Now()
	task := &Task{
		ID:            newUUID(),
		Type:          taskType,
		ExternalID:    externalID,
		Payload:       payload,
		Status:        TaskStatusNew,
		CreatedAt:     now,
		NextAttemptAt: now,
	}

	return task
}

// AddError appends an error message to the task's error log.
func (t *Task) AddError(err error) {
	if err == nil {
		return
	}
	taskErr := lo.FromPtr(t.Errors)
	taskErr += fmt.Sprintf("attempt %d: %v\n", t.Attempts, err)
	t.Errors = &taskErr
}

// IsInTerminalState reports whether the task is in a terminal status.
func (t *Task) IsInTerminalState() bool {
	switch t.Status {
	case TaskStatusDone, TaskStatusCanceled, TaskStatusAttemptsLeft:
		return true
	default:
		return false
	}
}

func newUUID() uuid.UUID {
	uuidV7, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}

	return uuidV7
}
