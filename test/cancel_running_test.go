package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestCancelRunningTask(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testCancelRunningTask)
}

// testCancelRunningTask checks that cancel wins over a running worker: the
// processor is not interrupted, but its result is dropped and the task stays canceled.
func testCancelRunningTask(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
	queueManager := goque.NewTaskQueueManager(storage)

	taskType := "test cancel running " + uuid.NewString()
	task := goque.NewTask(taskType, testutils.ToJSON(t, "payload"))
	task.NextAttemptAt = xtime.Now().Add(-time.Hour)
	pushToQueue(ctx, t, queueManager, task)

	running := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	processed := atomic.Int32{}

	goq := goque.NewGoque(storage)
	goq.RegisterProcessor(
		taskType,
		goque.TaskProcessorFunc(func(context.Context, *goque.Task) error {
			processed.Add(1)
			close(running)
			<-release
			return nil
		}),
		goque.WithTaskFetcherTick(goque.NewStaticValue(10*time.Millisecond)),
		goque.WithHooksAfterProcessing(func(_ context.Context, _ *goque.Task, err error) {
			finished <- err
		}),
	)
	require.NoError(t, goq.Run(ctx))
	defer goq.Stop()

	select {
	case <-running:
	case <-time.After(5 * time.Second):
		t.Fatal("task was not started")
	}

	require.NoError(t, queueManager.CancelTask(ctx, task.ID))
	close(release)

	select {
	case err := <-finished:
		require.ErrorIs(t, err, goque.ErrTaskVersionConflict)
	case <-time.After(5 * time.Second):
		t.Fatal("task was not finished")
	}

	dbTask, err := queueManager.GetTask(ctx, task.ID)
	require.NoError(t, err)
	require.Equal(t, goque.TaskStatusCanceled, dbTask.Status)
	require.Equal(t, int32(1), processed.Load(), "cancel does not stop an in-flight run")
}
