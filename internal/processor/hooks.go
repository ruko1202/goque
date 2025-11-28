package processor

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/lo"

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
	slog.InfoContext(ctx, "processing task",
		slog.Any("taskID", task.ID),
		slog.String("externalID", task.ExternalID),
		slog.String("type", task.Type),
		slog.String("status", task.Status),
		slog.Any("errors", task.Errors),
		slog.Time("createdAt", task.CreatedAt),
		slog.Any("updatedAt", task.UpdatedAt),
	)
}

// LoggingAfterProcessing default log after processing the task.
func LoggingAfterProcessing(ctx context.Context, task *entity.Task, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "failed to process task",
			slog.Any("taskID", task.ID),
			slog.String("externalID", task.ExternalID),
			slog.String("type", task.Type),
			slog.String("status", task.Status),
			slog.Any("errors", task.Errors),
			slog.Time("createdAt", task.CreatedAt),
			slog.Any("updatedAt", task.UpdatedAt),
			slog.Any("err", err),
		)
		return
	}

	slog.InfoContext(ctx, "process task successfully",
		slog.String("taskID", task.ID.String()),
	)
}

func (p *GoqueProcessor) updateTaskStateBeforeProcessing(ctx context.Context, task *entity.Task) {
	task.Status = entity.TaskStatusProcessing

	err := p.taskStorage.UpdateTask(ctx, task.ID, task)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update task state", slog.Any("err", err))
	}
}

func (p *GoqueProcessor) updateTaskAfterProcessing(ctx context.Context, task *entity.Task, taskErr error) {
	switch {
	case errors.Is(taskErr, ErrTaskCancel):
		task.Status = entity.TaskStatusCanceled
	case errors.Is(taskErr, context.Canceled):
		slog.InfoContext(ctx, "graceful shutdown: return task to queue", slog.String("taskID", task.ID.String()))
		task.Status = entity.TaskStatusNew
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
		slog.ErrorContext(ctx, "failed to update task state", slog.Any("err", err))
	}
}
