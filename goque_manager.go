package goque

import (
	"context"

	"github.com/google/uuid"

	"github.com/ruko1202/goque/internal/queuemanager"
)

// TaskQueueManager provides operations for managing tasks in the queue.
// It offers both synchronous and asynchronous methods for adding tasks,
// as well as querying and managing existing tasks.
type TaskQueueManager interface {
	AsyncAddTaskToQueue(ctx context.Context, task *Task)
	AddTaskToQueue(ctx context.Context, task *Task) error
	GetTask(ctx context.Context, taskID uuid.UUID) (*Task, error)
	GetTasks(ctx context.Context, filter *TaskFilter, limit int64) ([]*Task, error)
	ResetAttempts(ctx context.Context, taskID uuid.UUID) error
}

// NewTaskQueueManager creates a new TaskQueueManager instance with the specified task storage.
func NewTaskQueueManager(taskStorage TaskStorage) TaskQueueManager {
	return queuemanager.NewTaskQueueManager(taskStorage)
}
