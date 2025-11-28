package processor

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/entity"
)

type (
	// HookBeforeProcessing defines a hook function called before task processing begins.
	HookBeforeProcessing func(ctx context.Context, task *entity.Task)
	// HookAfterProcessing defines a hook function called after task processing completes.
	HookAfterProcessing func(ctx context.Context, task *entity.Task, err error)
)

func loggingBeforeProcessing(ctx context.Context, task *entity.Task) {
	slog.InfoContext(ctx, "processing task",
		slog.String("taskID", task.ID.String()),
	)
}

func loggingAfterProcessing(ctx context.Context, task *entity.Task, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "failed to process task",
			slog.String("taskID", task.ID.String()),
			slog.Any("err", err),
		)
		return
	}

	slog.InfoContext(ctx, "process task successfully",
		slog.String("taskID", task.ID.String()),
	)
}
