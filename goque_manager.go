package goque

import (
	"context"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
)

const (
	bigPayloadSize = 100 * 1024 // 100KB
)

// TaskQueueManager provides a high-level API for managing tasks in the queue.
// It combines task creation and storage operations in a single interface.
type TaskQueueManager struct {
	taskStorage TaskStorage
}

// NewTaskQueueManager creates a new TaskQueueManager instance with the specified task storage.
func NewTaskQueueManager(taskStorage TaskStorage) *TaskQueueManager {
	return &TaskQueueManager{
		taskStorage: taskStorage,
	}
}

// AsyncAddTaskToQueue adds a task to the queue asynchronously without waiting for completion.
func (m *TaskQueueManager) AsyncAddTaskToQueue(ctx context.Context, task *entity.Task) {
	go m.AddTaskToQueue(ctx, task) //nolint:errcheck
}

// AddTaskToQueue adds a task to the queue and returns an error if the operation fails.
func (m *TaskQueueManager) AddTaskToQueue(ctx context.Context, task *entity.Task) error {
	if len(task.Payload) > bigPayloadSize {
		xlog.Warn(ctx, "big payload size detected - may cause performance problems",
			zap.Int("payload_size", len(task.Payload)),
			zap.String("task_id", task.ID.String()),
			zap.String("task_type", task.Type))
	}

	return m.taskStorage.AddTask(ctx, task)
}

// GetTask retrieves a single task by its ID from the queue.
func (m *TaskQueueManager) GetTask(ctx context.Context, taskID uuid.UUID) (*Task, error) {
	return m.taskStorage.GetTask(ctx, taskID)
}

// GetTasks retrieves tasks from the queue based on the provided filter criteria.
// The limit parameter controls the maximum number of tasks to return.
func (m *TaskQueueManager) GetTasks(ctx context.Context, filter *TaskFilter, limit int64) ([]*Task, error) {
	return m.taskStorage.GetTasks(ctx, filter, limit)
}

// ResetAttempts resets the retry attempts counter for a task and sets its status back to new.
// This allows a failed task to be retried from the beginning.
func (m *TaskQueueManager) ResetAttempts(ctx context.Context, taskID uuid.UUID) error {
	return m.taskStorage.ResetAttempts(ctx, taskID)
}
