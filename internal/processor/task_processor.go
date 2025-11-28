package processor

import (
	"context"
	"log/slog"

	"github.com/ruko1202/goque/internal/entity"
)

// TaskProcessor defines the interface for processing individual tasks.
type TaskProcessor interface {
	ProcessTask(ctx context.Context, task *entity.Task) error
}

// TaskProcessorFunc is a function type that implements the TaskProcessor interface.
type TaskProcessorFunc func(ctx context.Context, task *entity.Task) error

// ProcessTask executes the task processing function.
func (f TaskProcessorFunc) ProcessTask(ctx context.Context, task *entity.Task) error {
	return f(ctx, task)
}

// NoopTaskProcessor returns a task processor that logs task information without performing any actual processing.
func NoopTaskProcessor() TaskProcessor {
	return TaskProcessorFunc(func(ctx context.Context, task *entity.Task) error {
		slog.InfoContext(ctx, "process task",
			slog.String("processor", "noop with log"),
			slog.Any("taskID", task.ID),
		)
		return nil
	})
}
