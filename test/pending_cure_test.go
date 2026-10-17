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

func TestPendingCuresKeepAttempts(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testPendingCuresKeepAttempts)
}

// testPendingCuresKeepAttempts checks that a task cured from pending (it never
// started, e.g. it waited for a free worker longer than the healer threshold)
// keeps its attempts however often it is cured, and still runs.
func testPendingCuresKeepAttempts(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
	queueManager := goque.NewTaskQueueManager(storage)

	const (
		maxAttempts = 2
		cures       = maxAttempts + 2
	)
	taskType := "test pending cures keep attempts " + uuid.NewString()
	task := goque.NewTask(taskType, testutils.ToJSON(t, "payload"))
	pushToQueue(ctx, t, queueManager, task)
	startVersion := task.Version

	for range cures {
		// Re-arm the task returned by the previous cure, so a cure that still
		// consumed an attempt would be visible here, not reset.
		task.Status = goque.TaskStatusPending
		task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))

		cured, err := storage.CureTasks(ctx, taskType, goque.TaskStatusPending, time.Minute, "stuck")
		require.NoError(t, err)
		require.Len(t, cured, 1)
		task = cured[0]
		require.Zero(t, task.Attempts, "a pending cure must not consume an attempt")
	}
	require.Equal(t, startVersion+cures, task.Version)

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
		return dbTask.Status == goque.TaskStatusDone
	}, time.Second, 10*time.Millisecond)
	require.Equal(t, int32(1), processed.Load())
}
