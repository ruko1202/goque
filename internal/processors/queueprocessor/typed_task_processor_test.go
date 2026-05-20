package queueprocessor

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
)

type typedPayload struct {
	Value string `json:"value"`
}

func TestTypedTaskProcessor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("decodes payload before processing", func(t *testing.T) {
		t.Parallel()

		task := entity.NewTask("typed", `{"value":"ok"}`)
		processor := NewTypedTaskProcessor(TypedTaskProcessorFunc[typedPayload](func(_ context.Context, typedTask *entity.TypedTask[typedPayload]) error {
			require.Equal(t, task.ID, typedTask.ID)
			require.Equal(t, `{"value":"ok"}`, typedTask.Task.Payload)
			require.Equal(t, typedPayload{Value: "ok"}, typedTask.Payload)
			return nil
		}))

		err := processor.ProcessTask(ctx, task)
		require.NoError(t, err)
	})

	t.Run("returns decode error without calling processor", func(t *testing.T) {
		t.Parallel()

		task := entity.NewTask("typed", `{"value":`)
		processorCalled := false
		processor := NewTypedTaskProcessor(TypedTaskProcessorFunc[typedPayload](func(_ context.Context, _ *entity.TypedTask[typedPayload]) error {
			processorCalled = true
			return nil
		}))

		err := processor.ProcessTask(ctx, task)
		require.ErrorIs(t, err, entity.ErrPayloadUnmarshal)
		require.False(t, processorCalled)
	})
}

func TestWithPayloadDecodeErrorCancel(t *testing.T) {
	t.Parallel()

	task := entity.NewTask("typed", `{"value":`)
	processor := NewTypedTaskProcessor(TypedTaskProcessorFunc[typedPayload](func(_ context.Context, _ *entity.TypedTask[typedPayload]) error {
		return nil
	}), WithCancelTaskWhenPayloadDecodeError[typedPayload]())
	goqueProc, _ := initGoqueProcessorWithMocks(t, task.Type, processor)

	err := goqueProc.processTask(context.Background(), task)
	require.ErrorIs(t, err, entity.ErrPayloadUnmarshal)
	require.ErrorIs(t, err, entity.ErrTaskCancel)
	require.True(t, errors.Is(err, entity.ErrPayloadUnmarshal))
}

func TestPayloadDecodeErrorCancelStoresTaskError(t *testing.T) {
	t.Parallel()

	task := entity.NewTask("typed", `{"value":`)
	goqueProc, mocks := initGoqueProcessorWithMocks(t, task.Type, NoopTaskProcessor())
	taskErr := errors.Join(entity.ErrTaskCancel, fmt.Errorf("%w: invalid payload", entity.ErrPayloadUnmarshal))

	mocks.taskStorage.EXPECT().
		UpdateTask(gomock.Any(), task.ID, task).
		DoAndReturn(func(_ context.Context, taskID uuid.UUID, updatedTask *entity.Task) error {
			require.Equal(t, task.ID, taskID)
			require.Equal(t, entity.TaskStatusCanceled, updatedTask.Status)
			require.NotNil(t, updatedTask.Errors)
			require.Contains(t, *updatedTask.Errors, entity.ErrPayloadUnmarshal.Error())
			return nil
		})

	goqueProc.updateTaskState(context.Background(), task, taskErr)
}
