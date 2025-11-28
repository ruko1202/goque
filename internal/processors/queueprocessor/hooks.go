package queueprocessor

import (
	"context"
	"errors"

	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/entity"
)

type (
	// HookBeforeProcessing defines a hook function called before task processing begins.
	HookBeforeProcessing func(ctx context.Context, task *entity.Task)
	// HookAfterProcessing defines a hook function called after task processing completes.
	HookAfterProcessing func(ctx context.Context, task *entity.Task, err error)
)

// LoggingBeforeProcessing default log before processing the task.
func LoggingBeforeProcessing(ctx context.Context, task *entity.Task) {
	xlog.Info(ctx, "processing task",
		zap.String("externalID", task.ExternalID),
		zap.String("type", task.Type),
		zap.String("status", task.Status),
		zap.String("errors", lo.FromPtr(task.Errors)),
		zap.Time("createdAt", task.CreatedAt),
		zap.Any("updatedAt", task.UpdatedAt),
	)
}

// LoggingAfterProcessing default log after processing the task.
func LoggingAfterProcessing(ctx context.Context, task *entity.Task, err error) {
	if err != nil {
		xlog.Error(ctx, "failed to process task",
			zap.String("externalID", task.ExternalID),
			zap.String("type", task.Type),
			zap.String("status", task.Status),
			zap.String("errors", lo.FromPtr(task.Errors)),
			zap.Time("createdAt", task.CreatedAt),
			zap.Any("updatedAt", task.UpdatedAt),
			zap.Error(err),
		)
		return
	}

	xlog.Info(ctx, "process task successfully")
}

func (p *GoqueProcessor) updateTaskStateBeforeProcessing(ctx context.Context, task *entity.Task) {
	task.Status = entity.TaskStatusProcessing

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", zap.Error(err))
	}
}

func (p *GoqueProcessor) updateTaskState(ctx context.Context, task *entity.Task, taskErr error) {
	xlog.WithFields(ctx, zap.String("processor.action", "updateTaskState"))
	ctx = context.WithoutCancel(ctx)

	switch {
	case errors.Is(taskErr, ErrTaskCancel):
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
		xlog.Error(ctx, "failed to update task state", zap.Error(err))
	}
}

func (p *GoqueProcessor) returnTaskWhenGracefulShutdown(ctx context.Context, task *entity.Task) {
	ctx = context.WithoutCancel(ctx)

	xlog.Info(ctx, "graceful shutdown: return task to queue")
	task.Status = entity.TaskStatusNew

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		xlog.Error(ctx, "failed to update task state", zap.Error(err))
	}
}
