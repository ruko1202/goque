package processor

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ruko1202/goque/internal/entity"
)

func TestGoqueProcessor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
			WithWorkersCount(1),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), task.ID, task).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskService.EXPECT().
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
		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
			WithTaskTimeout(100*time.Millisecond),
			WithStaticNextAttemptAtFunc(time.Minute),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusError, task.Status)
					assert.Equal(t, "attempt 1: task processing timeout: 100ms\n", lo.FromPtr(task.Errors))
					assert.Equal(t,
						now.Add(time.Minute).In(time.UTC).Round(time.Minute),
						task.NextAttemptAt.Round(time.Minute),
					)
					return nil
				}),
		)

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() == 1
		}, time.Second*20, time.Millisecond*500)
		goqueProc.Stop()
	})

	t.Run("max attempts", func(t *testing.T) {
		t.Parallel()
		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
				return fmt.Errorf("task processing error")
			}),
			WithTaskFetcherTick(100*time.Millisecond),
			WithMaxAttempts(1),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskService.EXPECT().
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
		}, time.Second*2, time.Millisecond*500)
		goqueProc.Stop()
	})

	t.Run("task canceled", func(t *testing.T) {
		t.Parallel()
		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
			WithMaxAttempts(1),
		)

		defaultFetcherMock(mocks, task.Type, []*entity.Task{task})

		gomock.InOrder(
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskService.EXPECT().
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
		}, time.Second*2, time.Millisecond*500)
		goqueProc.Stop()
	})

	t.Run("hooks", func(t *testing.T) {
		t.Parallel()
		task := &entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
			WithNextAttemptAtFunc(RoundStepNextAttemptAtFunc([]time.Duration{time.Minute})),
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
			mocks.taskService.EXPECT().
				UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, taskID uuid.UUID, task *entity.Task) error {
					assert.Equal(t, task.ID, taskID)
					assert.Equal(t, entity.TaskStatusProcessing, task.Status)
					return nil
				}),
			mocks.taskService.EXPECT().
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
		}, time.Second*2, time.Millisecond*500)
		goqueProc.Stop()
	})

	t.Run("graceful stop", func(t *testing.T) {
		t.Parallel()
		task := entity.Task{
			ID:            uuid.New(),
			Type:          "test task type",
			ExternalID:    "internal-" + uuid.NewString(),
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
				case <-time.After(time.Second):
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

		mocks.taskService.EXPECT().
			UpdateTask(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil).
			AnyTimes()

		err := goqueProc.Run(ctx)
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			return processedTasks.Load() >= 5
		}, time.Second*2, time.Millisecond*500)

		goqueProc.Stop()
		<-time.After(2 * time.Second)
		require.LessOrEqual(t, processedTasks.Load(), int32(10))
	})
}
