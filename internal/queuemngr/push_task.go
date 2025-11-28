// Package queuemngr provides queue management functionality for adding tasks to storage.
package queuemngr

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/entity"
)

// AsyncAddTaskToQueue adds a task to the queue asynchronously without waiting for completion.
func (q *QueueMngr) AsyncAddTaskToQueue(ctx context.Context, task *entity.Task) {
	go q.AddTaskToQueue(ctx, task) //nolint:errcheck
}

// AddTaskToQueue adds a task to the queue and returns an error if the operation fails.
func (q *QueueMngr) AddTaskToQueue(ctx context.Context, task *entity.Task) error {
	err := q.taskStorage.AddTask(ctx, task)
	if err != nil {
		slog.ErrorContext(ctx, "failed to add task to queue", slog.Any("err", err))
		return err
	}

	return nil
}
