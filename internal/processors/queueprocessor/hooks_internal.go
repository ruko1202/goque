package queueprocessor

import (
	"context"
	"errors"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
	"github.com/samber/lo"

	"github.com/ruko1202/goque/internal/metrics"

	"github.com/ruko1202/goque/internal/entity"
)

func (p *GoqueProcessor) updateTaskStateBeforeProcessing(ctx context.Context, task *entity.Task) {
	task.Status = entity.TaskStatusProcessing

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", xfield.Error(err))
	}
}

func (p *GoqueProcessor) updateTaskState(ctx context.Context, task *entity.Task, taskErr error) {
	xlog.WithFields(ctx, xfield.String("processor.action", "updateTaskState"))
	ctx = context.WithoutCancel(ctx)

	switch {
	case errors.Is(taskErr, entity.ErrTaskCancel):
		task.Status = entity.TaskStatusCanceled
	case errors.Is(taskErr, context.Canceled):
		p.returnTaskWhenGracefulShutdown(ctx, task)
		return
	case taskErr != nil:
		task.Attempts = lo.Ternary(task.Attempts == 0, 1, task.Attempts+1)
		task.AddError(taskErr)

		if task.Attempts >= p.processor.maxAttempts {
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
	}
}

func (p *GoqueProcessor) returnTaskWhenGracefulShutdown(ctx context.Context, task *entity.Task) {
	ctx = context.WithoutCancel(ctx)

	xlog.Info(ctx, "graceful shutdown: return task to queue")
	task.Status = entity.TaskStatusNew

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", xfield.Error(err))
	}
}

// metricsBeforeProcessing is a placeholder hook for future extensions.
// OperationProcessing start time is captured directly in processTask method.
func (p *GoqueProcessor) metricsBeforeProcessing(_ context.Context, task *entity.Task) {
	metrics.IncProcessingTasks(task.Type, task.Status)
}

// metricsAfterProcessing collects metrics after task processing completes.
func (p *GoqueProcessor) metricsAfterProcessing(_ context.Context, task *entity.Task, _ error) {
	metrics.IncProcessingTasks(task.Type, task.Status)

	if task.Attempts > 0 {
		metrics.SetTaskRetryAttempts(task.Type, task.Attempts)
	}
}
