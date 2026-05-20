package queueprocessor

import (
	"context"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

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
	return TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
		xlog.Info(ctx, "process task", xfield.String("processor", "noop"))
		return nil
	})
}

// TypedTaskProcessor defines the interface for processing typed task payloads.
type TypedTaskProcessor[T any] interface {
	ProcessTask(ctx context.Context, task *entity.TypedTask[T]) error
}

// TypedTaskProcessorFunc is a function type that implements the TypedTaskProcessor interface.
type TypedTaskProcessorFunc[T any] func(ctx context.Context, task *entity.TypedTask[T]) error

// ProcessTask executes the typed task processing function.
func (f TypedTaskProcessorFunc[T]) ProcessTask(ctx context.Context, task *entity.TypedTask[T]) error {
	return f(ctx, task)
}

// NoopTypedTaskProcessor returns a task processor that logs task information without performing any actual processing.
func NoopTypedTaskProcessor[T any]() TypedTaskProcessor[T] {
	return TypedTaskProcessorFunc[T](func(ctx context.Context, _ *entity.TypedTask[T]) error {
		xlog.Info(ctx, "process task", xfield.String("processor", "noop"))
		return nil
	})
}
