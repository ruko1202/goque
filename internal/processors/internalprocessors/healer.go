package internalprocessors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xvalue"
)

const (
	defaultHealerTickPeriod       = 5 * time.Minute
	defaultHealerTimeout          = 30 * time.Second
	defaultHealerUpdatedAtTimeAgo = 1 * time.Hour

	// healerSafetyMargin is added to the processing floor.
	healerSafetyMargin = 10 * time.Second
)

// HealerTaskStorage defines the storage interface required for the healer processor to cure stuck tasks.
type HealerTaskStorage interface {
	CureTasks(ctx context.Context, taskType entity.TaskType, status entity.TaskStatus, updatedAtTimeAgo time.Duration, comment string) ([]*entity.Task, error)
}

// QueueHealer identifies and fixes tasks that have been stuck in pending or processing status for too long.
type QueueHealer struct {
	*baseProcessor
	taskStorage HealerTaskStorage

	updatedAtTimeAgo *xvalue.Positive[time.Duration]

	// processingTimeout returns the current processing timeout of the task type;
	// the threshold never goes below twice that plus a safety margin.
	processingTimeout func(ctx context.Context) time.Duration
}

// NewQueueHealer creates a new queue healer. processingTimeout returns the current
// processing timeout of the task type and is read on every run.
func NewQueueHealer(taskStorage HealerTaskStorage, taskType entity.TaskType, processingTimeout func(ctx context.Context) time.Duration) *QueueHealer {
	q := &QueueHealer{
		taskStorage:       taskStorage,
		processingTimeout: processingTimeout,
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

// CureTasks marks tasks stuck in pending or processing status as errored based on
// one threshold: the configured one, raised to the processing floor.
func (q *QueueHealer) CureTasks(ctx context.Context, taskType entity.TaskType) ([]*entity.Task, error) {
	ctx, span := xlog.WithOperationSpan(ctx, "queue_healer.CureTasks")
	defer span.End()

	threshold := q.threshold(ctx)
	processing, errProcessing := q.cure(ctx, taskType, threshold, entity.TaskStatusProcessing)
	pending, errPending := q.cure(ctx, taskType, threshold, entity.TaskStatusPending)

	return append(processing, pending...), errors.Join(errProcessing, errPending)
}

func (q *QueueHealer) cure(ctx context.Context, taskType entity.TaskType, threshold time.Duration, status entity.TaskStatus) ([]*entity.Task, error) {
	tasks, err := q.taskStorage.CureTasks(ctx, taskType, status, threshold, "task was stuck")
	if err != nil {
		return nil, fmt.Errorf("failed to cure the queue: %w", err)
	}

	return tasks, nil
}

// threshold returns how long a task must stay unchanged to be cured: the
// configured value, but never less than the processing floor (one processing
// timeout for the run plus one for the out-of-timeout tail — hooks, final
// write — plus a safety margin). A configured value below the floor could cure
// a processing task a live worker still holds, so it is raised and reported on
// every run until the configuration is fixed.
func (q *QueueHealer) threshold(ctx context.Context) time.Duration {
	configured := q.updatedAtTimeAgo.Load(ctx)
	floor := 2*q.processingTimeout(ctx) + healerSafetyMargin
	if configured >= floor {
		return configured
	}

	xlog.Warn(ctx, "healer threshold is below the processing floor, raised",
		xfield.Duration("configured", configured),
		xfield.Duration("applied", floor),
	)

	return floor
}
