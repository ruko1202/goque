package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/goquectx"
	"github.com/ruko1202/goque/test/testutils"
)

func TestGoque(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGoque)
}

func testGoque(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := context.Background()
	queueManager := goque.NewTaskQueueManager(storage)

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.ContextWithValue(ctx, "testname", t.Name())

		task := goque.NewTask(
			"test push and process type"+uuid.NewString(),
			testutils.ToJSON(t, "test payload: "+uuid.NewString()),
		)
		pushToQueue(ctx, t, queueManager, task)

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			task.Type,
			goque.NoopTaskProcessor(),
			goque.WithTaskFetcherTick(10*time.Millisecond),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			task, err := queueManager.GetTask(ctx, task.ID)
			require.NoError(t, err)
			t.Log("wait task status:", goque.TaskStatusDone, "actual status:", task.Status)
			return task.Status == goque.TaskStatusDone
		}, time.Second, time.Millisecond*50)
	})

		}, time.Second, time.Millisecond*50)
	})

	t.Run("stop when in pending a lot of tasks", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.ContextWithValue(ctx, "testname", t.Name())

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		taskType := "test push and process type" + uuid.NewString()

		tasks := make(map[string]*goque.Task)
		for range 10 {
			task := goque.NewTaskWithExternalID(
				taskType,
				testutils.ToJSON(t, "test payload: "+uuid.NewString()),
				uuid.NewString(),
			)
			tasks[task.ExternalID] = task
			pushToQueue(ctx, t, queueManager, task)
		}

		doneOneTask := atomic.Bool{}

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			taskType,
			goque.TaskProcessorFunc(func(_ context.Context, task *goque.Task) error {
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				t.Log("process task: ", task.ID, "payload:", task.Payload, "type:", task.Type)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(90 * time.Millisecond):
					tasks[task.ExternalID].Status = goque.TaskStatusDone
					doneOneTask.Store(true)
					return nil
				}
			}),
			goque.WithWorkersCount(1),
			goque.WithTaskFetcherTick(10*time.Millisecond),
			goque.WithTaskFetcherMaxTasks(100),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)

		require.Eventually(t, doneOneTask.Load, time.Second, 10*time.Millisecond)
		goq.Stop()

		processedTasks := lo.Filter(lo.Values(tasks), func(item *goque.Task, _ int) bool {
			return item.Status == goque.TaskStatusDone
		})
		assert.LessOrEqual(t, len(processedTasks), 3)

		for _, task := range tasks {
			actualTask, err := queueManager.GetTask(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, task.Status, actualTask.Status)
		}
	})
}
