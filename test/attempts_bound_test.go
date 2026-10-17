package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestCuredTaskAttemptsBound(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testCuredTaskAttemptsBound)
}

// testCuredTaskAttemptsBound checks that a task whose attempts were used up by
// healer cures from processing ends in attempts_left without another run.
func testCuredTaskAttemptsBound(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
	queueManager := goque.NewTaskQueueManager(storage)

	const maxAttempts = 2
	taskType := "test cured attempts bound " + uuid.NewString()
	task := goque.NewTask(taskType, testutils.ToJSON(t, "payload"))
	pushToQueue(ctx, t, queueManager, task)

	// Each processing cure counts as an attempt: cure the task maxAttempts times.
	for range maxAttempts {
		task.Status = goque.TaskStatusProcessing
		task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))

		cured, err := storage.CureTasks(ctx, taskType, goque.TaskStatusProcessing, time.Minute, "stuck")
		require.NoError(t, err)
		require.Len(t, cured, 1)
		task = cured[0]
	}
	require.EqualValues(t, maxAttempts, task.Attempts)

	// Make the task due now (MySQL rounds timestamps to seconds).
	task.NextAttemptAt = xtime.Now().Add(-time.Hour)
	require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))

	processed := atomic.Int32{}
	goq := goque.NewGoque(storage)
	goq.RegisterProcessor(
		taskType,
		goque.TaskProcessorFunc(func(context.Context, *goque.Task) error {
			processed.Add(1)
			return nil
		}),
		goque.WithTaskFetcherTick(goque.NewStaticValue(10*time.Millisecond)),
		goque.WithTaskProcessingMaxAttempts(goque.NewStaticValue[int32](maxAttempts)),
	)
	require.NoError(t, goq.Run(ctx))
	defer goq.Stop()

	require.Eventually(t, func() bool {
		dbTask, err := queueManager.GetTask(ctx, task.ID)
		require.NoError(t, err)
		return dbTask.Status == goque.TaskStatusAttemptsLeft
	}, time.Second, 10*time.Millisecond)
	require.Zero(t, processed.Load(), "exhausted task must not run again")
}
