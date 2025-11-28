package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/storages"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/test/testutils"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

func TestGoque(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGoque)
}

func testGoque(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		task := entity.NewTask(
			"test push and process type"+uuid.NewString(),
			testutils.ToJSON(t, "test payload: "+uuid.NewString()),
		)
		pushToQueue(ctx, t, storage, task)

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			task.Type,
			queueprocessor.TaskProcessorFunc(func(_ context.Context, task *entity.Task) error {
				t.Log("process task: ", task.ID, "payload:", task.Payload, "type:", task.Type)
				return nil
			}),
			queueprocessor.WithTaskFetcherTick(10*time.Millisecond),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			task, err := storage.GetTask(ctx, task.ID)
			require.NoError(t, err)
			t.Log("wait task status:", entity.TaskStatusDone, "actual status:", task.Status)
			return task.Status == entity.TaskStatusDone
		}, time.Second, time.Millisecond*50)
	})

	t.Run("stop when in pending a lot of tasks", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		taskType := "test push and process type" + uuid.NewString()

		tasks := make(map[string]*entity.Task)
		for range 10 {
			task := entity.NewTaskWithExternalID(
				taskType,
				testutils.ToJSON(t, "test payload: "+uuid.NewString()),
				uuid.NewString(),
			)
			tasks[task.ExternalID] = task
			pushToQueue(ctx, t, storage, task)
		}

		doneOneTask := atomic.Bool{}

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			taskType,
			queueprocessor.TaskProcessorFunc(func(_ context.Context, task *entity.Task) error {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				t.Log("process task: ", task.ID, "payload:", task.Payload, "type:", task.Type)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(90 * time.Millisecond):
					tasks[task.ExternalID].Status = entity.TaskStatusDone
					doneOneTask.Store(true)
					return nil
				}
			}),
			queueprocessor.WithWorkersCount(1),
			queueprocessor.WithTaskFetcherTick(10*time.Millisecond),
			queueprocessor.WithTaskFetcherMaxTasks(100),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)

		require.Eventually(t, doneOneTask.Load, time.Second, 10*time.Millisecond)
		goq.Stop()

		processedTasks := lo.Filter(lo.Values(tasks), func(item *entity.Task, _ int) bool {
			return item.Status == entity.TaskStatusDone
		})
		assert.LessOrEqual(t, len(processedTasks), 3)

		for _, task := range tasks {
			actualTask, err := storage.GetTask(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, task.Status, actualTask.Status)
		}
	})
}
