// Package internalprocessors provides internal task processors for queue management and maintenance.
package internalprocessors

import (
	"context"
	"log/slog"
	"time"

	"github.com/ruko1202/goque/internal/commonopts"
	"github.com/ruko1202/goque/internal/entity"
	taskstorage "github.com/ruko1202/goque/internal/storages/task"
)

const (
	// CleanerProcessorName is the identifier for the queue cleaner processor.
	CleanerProcessorName = "task cleaner[internal processor]"
	// DefaultCleanerTimeout is the default timeout for cleaner operations.
	DefaultCleanerTimeout          = 30 * time.Second
	defaultCleanerUpdatedAtTimeAgo = 3 * time.Hour
)

// QueueCleaner removes old completed, canceled, or failed tasks from the queue.
type QueueCleaner struct {
	updatedAtTimeAgo time.Duration
	taskStorage      *taskstorage.Storage
}

// NewQueueCleaner creates a new queue cleaner with the specified storage and options.
func NewQueueCleaner(taskStorage *taskstorage.Storage, opts ...QueueCleanerOpts) *QueueCleaner {
	q := &QueueCleaner{
		taskStorage:      taskStorage,
		updatedAtTimeAgo: defaultCleanerUpdatedAtTimeAgo,
	}

	q.tune(opts)

	return q
}

// Tune reconfigures the cleaner with new options.
func (q *QueueCleaner) Tune(opts []commonopts.InternalProcessorOpt) {
	q.tune(GetCleanerOpts(opts))
}

// Tune reconfigures the cleaner with new options.
func (q *QueueCleaner) tune(opts []QueueCleanerOpts) {
	for _, opt := range opts {
		opt(q)
	}
}

// CleanTasksQueue removes old tasks with done, canceled, or attempts_left status from the queue.
func (q *QueueCleaner) CleanTasksQueue(ctx context.Context) ([]*entity.Task, error) {
	tasks, err := q.taskStorage.DeleteTasks(ctx, []entity.TaskStatus{
		entity.TaskStatusDone,
		entity.TaskStatusCanceled,
		entity.TaskStatusAttemptsLeft,
	}, q.updatedAtTimeAgo)
	if err != nil {
		slog.ErrorContext(ctx, "failed to clean the queue", slog.Any("err", err))
		return nil, err
	}
	for _, task := range tasks {
		slog.InfoContext(ctx, "removed task from queue",
			slog.Any("taskID", task.ID),
			slog.String("externalID", task.ExternalID),
			slog.String("type", task.Type),
			slog.String("status", task.Status),
			slog.Any("errors", task.Errors),
			slog.Time("createdAt", task.CreatedAt),
			slog.Any("updatedAt", task.UpdatedAt),
		)
	}

	return tasks, nil
}
