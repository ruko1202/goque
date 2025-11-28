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
	// Healer is a fake task type used internally by the healer.
	Healer = "task healer[internal processor]"
	// DefaultHealerTimeout is the default timeout for healer operations.
	DefaultHealerTimeout          = 30 * time.Second
	defaultHealerUpdatedAtTimeAgo = 1 * time.Hour
	defaultHealerMaxTasks         = 100
)

// QueueHealer identifies and fixes tasks that have been stuck in pending status for too long.
type QueueHealer struct {
	updatedAtTimeAgo time.Duration
	maxTasks         int64
	taskStorage      *taskstorage.Storage
}

// NewQueueHealer creates a new queue healer with the specified storage and options.
func NewQueueHealer(taskStorage *taskstorage.Storage, opts ...QueueHealerOpts) *QueueHealer {
	q := &QueueHealer{
		taskStorage:      taskStorage,
		updatedAtTimeAgo: defaultHealerUpdatedAtTimeAgo,
		maxTasks:         defaultHealerMaxTasks,
	}

	q.tune(opts)

	return q
}

// Tune reconfigures the healer with new options.
func (q *QueueHealer) Tune(opts []commonopts.InternalProcessorOpt) {
	q.tune(GetHealerOpts(opts))
}

// Tune reconfigures the healer with new options.
func (q *QueueHealer) tune(opts []QueueHealerOpts) {
	for _, opt := range opts {
		opt(q)
	}
}

// CureTasks marks stuck tasks in pending status as errored based on the configured time threshold.
func (q *QueueHealer) CureTasks(ctx context.Context) ([]*entity.Task, error) {
	tasks, err := q.taskStorage.CureTasks(ctx, entity.TaskStatusPending, q.updatedAtTimeAgo, q.maxTasks)
	if err != nil {
		slog.ErrorContext(ctx, "failed to cure the queue", slog.Any("err", err))
		return nil, err
	}

	for _, task := range tasks {
		slog.InfoContext(ctx, "cured task from queue",
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
