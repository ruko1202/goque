package queueprocessor

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"
)

type typedPayload struct {
	Value string `json:"value"`
}

func TestTypedTaskProcessor(t *testing.T) {
	t.Parallel()
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		task := entity.NewTask("typed", `{"value":"ok"}`)
		processor := NewTypedTaskProcessor(
			TypedTaskProcessorFunc[typedPayload](func(_ context.Context, typedTask *entity.TypedTask[typedPayload]) error {
				require.Equal(t, task.ID, typedTask.ID)
				require.Equal(t, `{"value":"ok"}`, typedTask.Task.Payload)
				require.Equal(t, typedPayload{Value: "ok"}, typedTask.Payload)
				return nil
			}),
		)

		err := processor.ProcessTask(ctx, task)
		require.NoError(t, err)
	})

	t.Run("decode error", func(t *testing.T) {
		t.Parallel()

		t.Run("default behavior", func(t *testing.T) {
			t.Parallel()

			task := entity.NewTask("typed", `{"value":`)
			processorCalled := false
			processor := NewTypedTaskProcessor(
				TypedTaskProcessorFunc[typedPayload](func(_ context.Context, _ *entity.TypedTask[typedPayload]) error {
					processorCalled = true
					return nil
				}),
			)

			err := processor.ProcessTask(ctx, task)
			require.ErrorIs(t, err, entity.ErrPayloadUnmarshal)
			require.False(t, processorCalled)
		})

		t.Run("WithCancelTask", func(t *testing.T) {
			t.Parallel()

			task := entity.NewTask("typed", `{"value":`)
			processor := NewTypedTaskProcessor(
				NoopTypedTaskProcessor[typedPayload](),
				WithCancelTaskWhenPayloadDecodeError[typedPayload](),
			)
			goqueProc, _ := initGoqueProcessorWithMocks(t, task.Type, processor)

			err := goqueProc.processTask(ctx, task)
			require.ErrorIs(t, err, entity.ErrPayloadUnmarshal)
			require.ErrorIs(t, err, entity.ErrTaskCancel)
		})

		t.Run("set error `decoding payload` to task when canceling", func(t *testing.T) {
			t.Parallel()

			task := entity.NewTask("typed", `{"value":`)
			processor := NewTypedTaskProcessor(
				NoopTypedTaskProcessor[typedPayload](),
				WithCancelTaskWhenPayloadDecodeError[typedPayload](),
			)
			goqueProc, mocks := initGoqueProcessorWithMocks(t, task.Type, processor)

			gomock.InOrder(
				mocks.taskStorage.EXPECT().
					UpdateTask(gomock.Any(), task.ID, task).
					DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
						assert.Equal(t, task.ID, taskID)
						assert.Equal(t, entity.TaskStatusProcessing, task.Status)
						return nil
					}),
				mocks.taskStorage.EXPECT().
					UpdateTask(gomock.Any(), task.ID, task).
					DoAndReturn(func(_ context.Context, taskID uuid.UUID, updatedTask *entity.Task) error {
						assert.Equal(t, task.ID, taskID)
						assert.Equal(t, entity.TaskStatusCanceled, updatedTask.Status)
						assert.NotNil(t, updatedTask.Errors)
						assert.Contains(t, lo.FromPtr(updatedTask.Errors), entity.ErrPayloadUnmarshal.Error())
						return nil
					}),
			)

			goqueProc.doProcessTask(ctx, task)
		})
	})
}
