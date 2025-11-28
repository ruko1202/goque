// Package queuemanager provides high-level task queue management operations.
// It implements the TaskQueueManager interface and coordinates task storage,
// metrics tracking, and payload validation.
package queuemanager

import (
	"context"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/metrics"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/storages/dbentity"
)

const (
	bigPayloadSize = 100 * 1024 // 100KB
)

// TaskQueueManager provides a high-level API for managing tasks in the queue.
// It combines task creation and storage operations in a single interface.
type TaskQueueManager struct {
	taskStorage storages.Task
}

// NewTaskQueueManager creates a new TaskQueueManager instance with the specified task storage.
func NewTaskQueueManager(taskStorage storages.Task) *TaskQueueManager {
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
	metrics.SetTaskPayloadSize(task.Type, len(task.Payload))
	metrics.IncProcessingTasks(task.Type, entity.TaskStatusNew)

	if len(task.Payload) > bigPayloadSize {
		xlog.Warn(ctx, "big payload size detected - may cause performance problems",
			zap.Int("payload_size", len(task.Payload)),
			zap.String("task_id", task.ID.String()),
			zap.String("task_type", task.Type))
	}

	err := m.taskStorage.AddTask(ctx, task)
	if err != nil {
		return err
	}

	return nil
}

// GetTask retrieves a single task by its ID from the queue.
func (m *TaskQueueManager) GetTask(ctx context.Context, taskID uuid.UUID) (*entity.Task, error) {
	return m.taskStorage.GetTask(ctx, taskID)
}

// GetTasks retrieves tasks from the queue based on the provided filter criteria.
// The limit parameter controls the maximum number of tasks to return.
func (m *TaskQueueManager) GetTasks(ctx context.Context, filter *dbentity.GetTasksFilter, limit int64) ([]*entity.Task, error) {
	return m.taskStorage.GetTasks(ctx, filter, limit)
}

// ResetAttempts resets the retry attempts counter for a task and sets its status back to new.
// This allows a failed task to be retried from the beginning.
func (m *TaskQueueManager) ResetAttempts(ctx context.Context, taskID uuid.UUID) error {
	return m.taskStorage.ResetAttempts(ctx, taskID)
}
