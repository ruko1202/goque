package base

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/processors/queueprocessor"
)

func TestGoque(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()

		task := entity.NewTask(
			"test push and process type"+uuid.NewString(),
			toJSON(t, "test payload: "+uuid.NewString()),
		)
		pushToQueue(ctx, t, task)

		goq := goque.NewGoque(taskStorage)
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
			task, err := taskStorage.GetTask(ctx, task.ID)
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
				toJSON(t, "test payload: "+uuid.NewString()),
				uuid.NewString(),
			)
			tasks[task.ExternalID] = task
			pushToQueue(ctx, t, task)
		}

		goq := goque.NewGoque(taskStorage)
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
					return nil
				}
			}),
			queueprocessor.WithWorkersCount(1),
			queueprocessor.WithTaskFetcherTick(10*time.Millisecond),
			queueprocessor.WithTaskFetcherMaxTasks(100),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)

		<-time.After(100 * time.Millisecond)
		goq.Stop()

		processedTasks := lo.Filter(lo.Values(tasks), func(item *entity.Task, _ int) bool {
			return item.Status == entity.TaskStatusDone
		})
		assert.Len(t, processedTasks, 1)

		for _, task := range tasks {
			actualTask, err := taskStorage.GetTask(ctx, task.ID)
			require.NoError(t, err)
			require.Equal(t, task.Status, actualTask.Status)
		}
	})
}
