package queueprocessor

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestGoqueProcessor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := xtime.Now()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "type[ok]",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        nil,
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error {
				processedTasks.Add(1)
				return nil
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithTaskFetcherMaxTasks(defaultFetchMaxTasks),
			WithWorkersCount(10),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

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
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusDone, task.Status)
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 1
		}, time.Second*2, time.Millisecond*100)
		goqueProc.Stop()
	})

	t.Run("task process timeout", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "type[process timeout]",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        nil,
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
				defer func() {
					processedTasks.Add(1)
				}()

				select {
				case <-ctx.Done():
					t.Log("ctx done")
					return ctx.Err()
				case <-time.After(time.Second):
					t.Log("time.After")
					return nil
				}
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithTaskProcessingTimeout(100*time.Millisecond),
			WithTaskProcessingNextAttemptAtFunc(StaticNextAttemptAtFunc(time.Minute)),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusError, task.Status)
					assert.Equal(t, "attempt 1: task processing timeout: 100ms. context deadline exceeded\n", lo.FromPtr(task.Errors))
					testutils.AssertTimeInWithDelta(t, now.Add(time.Minute).In(time.UTC), task.NextAttemptAt, time.Minute)
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 1
		}, time.Second*20, time.Millisecond*100)
		goqueProc.Stop()
	})

	t.Run("max attempts", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "type[max attempts",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        nil,
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error {
				processedTasks.Add(1)
				return errors.New("task processing error")
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithTaskProcessingMaxAttempts(1),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusAttemptsLeft, task.Status)
					assert.Equal(t, "attempt 1: task processing error\n", lo.FromPtr(task.Errors))
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 1
		}, time.Second*2, time.Millisecond*100)
		goqueProc.Stop()
	})

	t.Run("task canceled", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "type[task canceled]",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        lo.ToPtr("attempt 1: task processing error\n"),
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error {
				processedTasks.Add(1)
				return ErrTaskCancel
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithTaskProcessingMaxAttempts(1),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusCanceled, task.Status)
					assert.Equal(t, "attempt 1: task processing error\n", lo.FromPtr(task.Errors))
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 1
		}, time.Second*2, time.Millisecond*100)
		goqueProc.Stop()
	})

	t.Run("hooks", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "type[hooks]",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        lo.ToPtr("attempt 1: task processing error\n"),
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(_ context.Context, _ *entity.Task) error {
				processedTasks.Add(1)
				return ErrTaskCancel
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithTaskProcessingNextAttemptAtFunc(RoundStepNextAttemptAtFunc([]time.Duration{time.Minute})),
			WithHooksBeforeProcessing(func(_ context.Context, task *entity.Task) {
				processedTasks.Add(1)
				t.Log("before processing task: ", task.ID)
			}),
			WithHooksAfterProcessing(func(_ context.Context, task *entity.Task, err error) {
				processedTasks.Add(1)
				t.Log("after processing task: ", task.ID, " err: ", err)
			}),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskStorage.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusCanceled, task.Status)
					assert.Equal(t, "attempt 1: task processing error\n", lo.FromPtr(task.Errors))
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 3
		}, time.Second*2, time.Millisecond*100)
		goqueProc.Stop()
	})

	t.Run("graceful stop", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := entity.Task{
			ID:            uuid.New(),
			Type:          "type[graceful stop]",
			ExternalID:    uuid.NewString(),
			Payload:       "test payload",
			Status:        entity.TaskStatusPending,
			Errors:        lo.ToPtr("attempt 1: task processing error\n"),
			CreatedAt:     now,
			UpdatedAt:     nil,
			NextAttemptAt: now,
		}

		processedTasks := atomic.Int32{}
		goqueProc, mocks := initGoqueProcessorWithMocks(t,
			task.Type,
			TaskProcessorFunc(func(ctx context.Context, _ *entity.Task) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(100 * time.Millisecond):
					processedTasks.Add(1)
				}

				return nil
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithWorkersCount(5),
		)

		defaultFetcherMock(mocks, task.Type, lo.RepeatBy(100, func(_ int) *entity.Task {
			copyTask := task
			copyTask.ID = uuid.New()
			return &copyTask
		}))

		mocks.taskStorage.EXPECT().
			UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() >= 5
		}, time.Second*2, time.Millisecond*100)

		goqueProc.Stop()
		require.LessOrEqual(t, processedTasks.Load(), int32(10))
	})
}
