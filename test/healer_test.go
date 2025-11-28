package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal"
	"github.com/ruko1202/goque/internal/entity"
	internalprocessors "github.com/ruko1202/goque/internal/internal_processors"
	"github.com/ruko1202/goque/internal/processor"
)

func TestHealer(t *testing.T) {
	ctx := context.Background()

	task := entity.NewTask(
		"test healer"+uuid.NewString(),
		toJSON(t, "test payload: "+uuid.NewString()),
	)
	pushToQueue(ctx, t, task)

	goque := internal.NewGoque(taskStorage)
	goque.TuneHealerProcessor(
		processor.WithTaskFetcherTick(100*time.Millisecond),
		internalprocessors.WithHealerUpdatedAtTimeAgo(100*time.Millisecond),
		internalprocessors.WithHealerMaxTasks(100),
	)
	// do nothing, only fetch
	goque.RegisterProcessor(
		task.Type,
		processor.NoopTaskProcessor(),
		processor.WithTaskFetcherTick(500*time.Millisecond),
		processor.WithReplaceHooksBeforeProcessing(processor.LoggingBeforeProcessing),
		processor.WithReplaceHooksAfterProcessing(processor.LoggingAfterProcessing),
	)
	goque.Run(ctx)
	defer goque.Stop()

	require.Eventually(t, func() bool {
		task, err := taskStorage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		t.Log("wait task status:", entity.TaskStatusError, "actual status:", task.Status)
		return task.Status == entity.TaskStatusError
	}, time.Second*10, time.Millisecond*500)
}
