package goque

import (
	"context"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
)

const (
	bigPayload = 100 * 1024 // 100Kb
)

// TaskPusher provides functionality for adding tasks to the queue storage.
type TaskPusher struct {
	taskStorage TaskStorage
}

// NewTaskPusher creates a new TaskPusher instance with the specified task storage.
func NewTaskPusher(taskStorage TaskStorage) *TaskPusher {
	return &TaskPusher{
		taskStorage: taskStorage,
	}
}

// AsyncAddTaskToQueue adds a task to the queue asynchronously without waiting for completion.
func (q *TaskPusher) AsyncAddTaskToQueue(ctx context.Context, task *entity.Task) {
	go q.AddTaskToQueue(ctx, task) //nolint:errcheck
}

// AddTaskToQueue adds a task to the queue and returns an error if the operation fails.
func (q *TaskPusher) AddTaskToQueue(ctx context.Context, task *entity.Task) error {
	if len(task.Payload) > bigPayload {
		xlog.Warn(ctx, "big payload size. may be potential performance problems: slow insert, fetch, etc")
	}
	err := q.taskStorage.AddTask(ctx, task)
	if err != nil {
		xlog.Error(ctx, "failed to add task to queue", zap.Error(err))
		return err
	}

	return nil
}
