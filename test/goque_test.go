package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/processor"
)

func TestGoque(t *testing.T) {
	ctx := context.Background()

	task := entity.NewTask(
		"test push and process type"+uuid.NewString(),
		toJSON(t, "test payload: "+uuid.NewString()),
	)
	pushToQueue(ctx, t, task)

	goque := internal.NewGoque(taskStorage)
	goque.RegisterProcessor(
		task.Type,
		processor.TaskProcessorFunc(func(_ context.Context, task *entity.Task) error {
			t.Log("process task: ", task.ID, "payload:", task.Payload, "type:", task.Type)
			return nil
		}),
		processor.WithTaskFetcherTick(200*time.Millisecond),
		processor.WithTaskFetcherMaxTasks(10),
	)
	goque.Run(ctx)
	defer goque.Stop()

	require.Eventually(t, func() bool {
		task, err := taskStorage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		t.Log("wait task status:", entity.TaskStatusDone, "actual status:", task.Status)
		return task.Status == entity.TaskStatusDone
	}, time.Second*10, time.Millisecond*500)
}
