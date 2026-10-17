package queueprocessor

import (
	"context"
	"errors"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/metrics"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
)

// updateTaskStateBeforeProcessing moves the task to processing. On error the
// task must not be processed and keeps its previous (fetched) status.
func (p *GoqueProcessor) updateTaskStateBeforeProcessing(ctx context.Context, task *entity.Task) error {
	prevStatus := task.Status
	task.Status = entity.TaskStatusProcessing

	writeCtx, cancel := p.writeContext(ctx)
	defer cancel()

	err := p.taskStorage.UpdateTask(writeCtx, task.ID, task)
	if err != nil {
		task.Status = prevStatus
		xlog.Error(ctx, "failed to update task state", xfield.Error(err))
		return err
	}

	return nil
}

func (p *GoqueProcessor) updateTaskState(ctx context.Context, task *entity.Task, taskErr error) error {
	xlog.WithFields(ctx, xfield.String("processor.action", "updateTaskState"))
	ctx, cancel := p.writeContext(ctx)
	defer cancel()

	var reschedule *entity.RescheduleError
	switch {
	case errors.Is(taskErr, entity.ErrTaskCancel):
		if errors.Is(taskErr, entity.ErrPayloadUnmarshal) {
			task.AddError(taskErr)
		}
		task.Status = entity.TaskStatusCanceled
	case errors.Is(taskErr, context.Canceled):
		return p.returnTaskWhenGracefulShutdown(ctx, task)
	case errors.Is(taskErr, entity.ErrTaskAttemptsExhausted):
		task.AddError(taskErr)
		task.Status = entity.TaskStatusAttemptsLeft
	case errors.As(taskErr, &reschedule):
		task.Status = entity.TaskStatusNew
		task.NextAttemptAt = xtime.Now().Add(max(reschedule.After, 0))
	case taskErr != nil:
		task.Attempts = lo.Ternary(task.Attempts == 0, 1, task.Attempts+1)
		task.AddError(taskErr)

		if task.Attempts >= p.processor.maxAttempts.Load(ctx) {
			task.Status = entity.TaskStatusAttemptsLeft
		} else {
			task.Status = entity.TaskStatusError
			task.NextAttemptAt = p.processor.nextAttemptAtFunc(task.Attempts)
		}
	default:
		task.Status = entity.TaskStatusDone
	}
	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", xfield.Error(err))
		return err
	}

	return nil
}

func (p *GoqueProcessor) returnTaskWhenGracefulShutdown(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "queue_processor.returnTaskWhenGracefulShutdown")
	defer span.End()
	ctx, cancel := p.writeContext(ctx)
	defer cancel()

	xlog.Info(ctx, "graceful shutdown: return task to queue")
	task.Status = entity.TaskStatusNew

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", xfield.Error(err))
		return err
	}

	return nil
}

// writeContext returns the context for a task state write. It is detached from
// the worker's cancellation, because a write canceled after commit would leave
// the stored version ahead of the in-memory one and every later write would
// conflict, but bounded by the processing timeout, so a hung database cannot
// block the worker forever.
func (p *GoqueProcessor) writeContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), p.processor.timeout.Load(ctx)) //nolint:gosec // callers defer cancel
}

// metricsBeforeProcessing is a placeholder hook for future extensions.
// OperationProcessing start time is captured directly in processTask method.
func (p *GoqueProcessor) metricsBeforeProcessing(_ context.Context, task *entity.Task) {
	metrics.IncProcessingTasks(task.Type, task.Status)
}

// metricsAfterProcessing collects metrics after task processing completes.
func (p *GoqueProcessor) metricsAfterProcessing(_ context.Context, task *entity.Task, err error) {
	metrics.IncProcessingTasks(task.Type, task.Status)

	if task.Attempts > 0 {
		metrics.SetTaskRetryAttempts(task.Type, task.Attempts)
	}

	// A version conflict means another writer (healer, cancel, reset, another
	// worker) owns the task now and the write was dropped.
	if errors.Is(err, entity.ErrTaskVersionConflict) {
		stage := metrics.VersionConflictStageFinish
		switch {
		case task.Status == entity.TaskStatusPending: // the start write failed
			stage = metrics.VersionConflictStageStart
		case errors.Is(err, context.Canceled): // returned to the queue on shutdown
			stage = metrics.VersionConflictStageReturn
		}
		metrics.IncTaskVersionConflicts(task.Type, stage)
	}
}
