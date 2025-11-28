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
	"github.com/ruko1202/goque/internal/queuemngr"
)

func TestGoque(t *testing.T) {
	ctx := context.Background()

	queueMngr := queuemngr.NewQueueMngr(taskStorage)

	task := entity.NewTask("test push and process type"+uuid.NewString(), toJSON(t, "test payload: "+uuid.NewString()))

	err := queueMngr.AddTaskToQueue(ctx, task)
	require.NoError(t, err)
	t.Log("added task:", task.ID, "payload:", task.Payload, "type:", task.Type)

	goque := internal.NewGoque(taskStorage)
	goque.RegisterProcessor(
		task.Type,
		processor.TaskProcessorFunc(func(_ context.Context, payload string) error {
			t.Log("process task: ", payload)
			return nil
		}),
		processor.WithFetcherTick(200*time.Millisecond),
		processor.WithFetcherMaxTasks(10),
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
