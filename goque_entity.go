// Package goque provides a robust, PostgreSQL-backed task queue system for Go applications.
package goque

import "github.com/ruko1202/goque/internal/entity"

// TaskType represents the type identifier for tasks in the queue.
type TaskType = entity.TaskType

// TaskStatus represents the current status of a task in its lifecycle.
type TaskStatus = entity.TaskStatus

// Task status constants define the possible states a task can be in.
const (
	TaskStatusNew          = entity.TaskStatusNew          // Task is ready to be picked up
	TaskStatusPending      = entity.TaskStatusPending      // Task is scheduled for future processing
	TaskStatusProcessing   = entity.TaskStatusProcessing   // Task is currently being processed
	TaskStatusDone         = entity.TaskStatusDone         // Task completed successfully
	TaskStatusCanceled     = entity.TaskStatusCanceled     // Task was manually canceled
	TaskStatusError        = entity.TaskStatusError        // Task failed but has retry attempts remaining
	TaskStatusAttemptsLeft = entity.TaskStatusAttemptsLeft // Task failed and exhausted all retries
)

// Task represents a unit of work to be processed by the queue system.
type Task = entity.Task

// Task creation functions for adding new tasks to the queue.
var (
	// NewTask creates a new task with the specified type and payload.
	NewTask = entity.NewTask
	// NewTaskWithExternalID creates a new task with an external identifier for idempotency.
	NewTaskWithExternalID = entity.NewTaskWithExternalID
)
