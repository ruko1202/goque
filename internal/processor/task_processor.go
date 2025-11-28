package processor

import (
	"context"
)

// TaskProcessorFunc is a function type that implements the TaskProcessor interface.
type TaskProcessorFunc func(ctx context.Context, payload string) error

// ProcessTask executes the task processing function.
func (f TaskProcessorFunc) ProcessTask(ctx context.Context, payload string) error {
	return f(ctx, payload)
}
