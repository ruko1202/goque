package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

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
	updatedAtTimeAgo time.Duration
}

// NewQueueHealer creates a new queue healer with the specified storage and options.
func NewQueueHealer(taskStorage HealerTaskStorage, taskType entity.TaskType) *QueueHealer {
	q := &QueueHealer{
		taskType:         taskType,
		taskStorage:      taskStorage,
		updatedAtTimeAgo: defaultHealerUpdatedAtTimeAgo,
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
	ctx = xlog.WithFields(ctx,
		zap.String("internal.processor.action", "CureTasks"),
		zap.Duration("timeout", q.processTimeout),
	)

	ctx, cancel := context.WithTimeout(ctx, q.processTimeout)
	defer cancel()

	tasks, err := q.taskStorage.CureTasks(ctx, q.taskType, []entity.TaskStatus{
		entity.TaskStatusProcessing,
		entity.TaskStatusPending,
	}, q.updatedAtTimeAgo, ErrTaskIsFrozen.Error())
	if err != nil {
		xlog.Error(ctx, "failed to cure the queue", zap.Error(err))
		return err
	}

	xlog.Info(ctx, fmt.Sprintf("cured the queue: %d tasks", len(tasks)))
	for _, task := range tasks {
		xlog.Info(ctx, "cured task from queue",
			zap.String("taskID", task.ID.String()),
			zap.String("externalID", task.ExternalID),
			zap.String("type", task.Type),
			zap.String("status", task.Status),
			zap.Any("errors", lo.FromPtr(task.Errors)),
			zap.Time("createdAt", task.CreatedAt),
			zap.Any("updatedAt", task.UpdatedAt),
		)
	}

	return nil
}
