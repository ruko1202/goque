package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/entity"
)

var (
	// ErrTaskIsFrozen is returned when a task cannot be cured because it is frozen.
	ErrTaskIsFrozen = errors.New("task is frozen")
)

const (
	queueHealer                   = "healer"
	defaultHealerTickPeriod       = 5 * time.Minute
	defaultHealerTimeout          = 30 * time.Second
	defaultHealerUpdatedAtTimeAgo = 1 * time.Hour
)

// HealerTaskStorage defines the storage interface required for the healer processor to cure stuck tasks.
type HealerTaskStorage interface {
	CureTasks(ctx context.Context, taskType entity.TaskType, unhealthStatuses []entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error)
}

// QueueHealer identifies and fixes tasks that have been stuck in pending status for too long.
type QueueHealer struct {
	*baseProcessor
	taskStorage HealerTaskStorage

	taskType         entity.TaskType
	timeout          time.Duration
	updatedAtTimeAgo time.Duration
}

// NewQueueHealer creates a new queue healer with the specified storage and options.
func NewQueueHealer(taskStorage HealerTaskStorage, taskType entity.TaskType) *QueueHealer {
	q := &QueueHealer{
		taskType:         taskType,
		taskStorage:      taskStorage,
		updatedAtTimeAgo: defaultHealerUpdatedAtTimeAgo,
		timeout:          defaultHealerTimeout,
	}
	q.baseProcessor = newBaseProcessor(
		queueHealer,
		defaultHealerTimeout,
		defaultHealerTickPeriod,
		q.CureTasks,
	)

	return q
}

// SetUpdatedAtTimeAgo sets the time threshold for considering a task as stuck.
func (q *QueueHealer) SetUpdatedAtTimeAgo(updatedAtTimeAgo time.Duration) {
	q.updatedAtTimeAgo = updatedAtTimeAgo
}

// CureTasks marks stuck tasks in pending status as errored based on the configured time threshold.
func (q *QueueHealer) CureTasks(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, q.timeout)
	defer cancel()

	tasks, err := q.taskStorage.CureTasks(ctx, q.taskType, []entity.TaskStatus{
		entity.TaskStatusProcessing,
		entity.TaskStatusPending,
	}, q.updatedAtTimeAgo, ErrTaskIsFrozen.Error())
	if err != nil {
		slog.ErrorContext(ctx, "failed to cure the queue", slog.Any("err", err))
		return err
	}

	slog.InfoContext(ctx, fmt.Sprintf("cured the queue: %d tasks", len(tasks)))
	for _, task := range tasks {
		slog.InfoContext(ctx, "cured task from queue",
			slog.Any("taskID", task.ID),
			slog.String("externalID", task.ExternalID),
			slog.String("type", task.Type),
			slog.String("status", task.Status),
			slog.Any("errors", lo.FromPtr(task.Errors)),
			slog.Time("createdAt", task.CreatedAt),
			slog.Any("updatedAt", task.UpdatedAt),
		)
	}

	return nil
}
