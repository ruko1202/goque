package internalprocessors

import (
	"context"
	"fmt"
	"time"

	"github.com/ruko1202/xlog"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

const (
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

	updatedAtTimeAgo *xvalue.Positive[time.Duration]
}

// NewQueueHealer creates a new queue healer with the specified storage and options.
func NewQueueHealer(taskStorage HealerTaskStorage, taskType entity.TaskType) *QueueHealer {
	q := &QueueHealer{
		taskStorage: taskStorage,
		updatedAtTimeAgo: xvalue.NewPositive(
			entity.OperationHealth+".updated_at_time_ago",
			xvalue.NewStaticValue(defaultHealerUpdatedAtTimeAgo),
			defaultHealerUpdatedAtTimeAgo,
		),
	}
	q.baseProcessor = newBaseProcessor(
		entity.OperationHealth,
		taskType,
		defaultHealerTimeout,
		defaultHealerTickPeriod,
		q.CureTasks,
	)

	return q
}

// SetUpdatedAtTimeAgo sets the time threshold for considering a task as stuck.
// The value is re-read on every run. Must be called before Run.
func (q *QueueHealer) SetUpdatedAtTimeAgo(updatedAtTimeAgo xvalue.Value[time.Duration]) {
	q.updatedAtTimeAgo = xvalue.NewPositive(q.processorName+".updated_at_time_ago", updatedAtTimeAgo, defaultHealerUpdatedAtTimeAgo)
}

// CureTasks marks stuck tasks in pending status as errored based on the configured time threshold.
func (q *QueueHealer) CureTasks(ctx context.Context, taskType entity.TaskType) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "queue_healer.CureTasks")
	defer span.End()

	comment := "task was stuck"
	tasks, err := q.taskStorage.CureTasks(ctx, taskType, []entity.TaskStatus{
		entity.TaskStatusProcessing,
		entity.TaskStatusPending,
	}, q.updatedAtTimeAgo.Load(ctx), comment)
	if err != nil {
		return nil, fmt.Errorf("failed to cure the queue: %w", err)
	}

	return tasks, nil
}
