package queueprocessor

import (
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-json"
	"github.com/ruko1202/goque/internal/metrics"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque/internal/entity"
)

type GoqueTypedProcessor[T any] struct {
	cancelTaskIfDecodePayloadError bool
	processor                      TypedTaskProcessor[T]
}

// NewTypedTaskProcessor wraps a typed task processor for use with RegisterProcessor.
func NewTypedTaskProcessor[T any](processor TypedTaskProcessor[T], opts ...GoqueTypedProcessorOpts[T]) TaskProcessor {
	p := &GoqueTypedProcessor[T]{
		cancelTaskIfDecodePayloadError: false,
		processor:                      processor,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func (p *GoqueTypedProcessor[T]) ProcessTask(ctx context.Context, task *entity.Task) error {
	ctx, span := xlog.WithOperationSpan(ctx, "typed_task_processor.ProcessTask")
	defer span.End()

	typedTask, err := decodeTaskPayload[T](task)
	if err != nil {
		xlog.Error(ctx, "failed to decode task payload",
			xfield.String("taskID", task.ID.String()),
			xfield.String("task_type", task.Type),
			xfield.Error(err),
		)

		if errors.Is(err, entity.ErrPayloadUnmarshal) {
			metrics.IncPayloadDecodeErrors(task.Type)
			// if cancelTaskIfDecodePayloadError is true, return the error to cancel the task
			// otherwise, log original error to retry task processing
			if p.cancelTaskIfDecodePayloadError {
				return fmt.Errorf("%w: %w", entity.ErrTaskCancel, err)
			}
		}

		return err
	}

	return p.processor.ProcessTask(ctx, typedTask)
}

func decodeTaskPayload[T any](task *entity.Task) (*entity.TypedTask[T], error) {
	var payload T
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return nil, fmt.Errorf("%w: %s task payload: %w", entity.ErrPayloadUnmarshal, task.Type, err)
	}

	return &entity.TypedTask[T]{
		Task:    task,
		Payload: payload,
	}, nil
}
