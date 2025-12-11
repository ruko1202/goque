package test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/test/testutils"
)

func TestCleaner(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testCleaner)
}

func testCleaner(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := context.Background()
	queueManager := goque.NewTaskQueueManager(storage)

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))
		ctx = goquectx.ContextWithValue(ctx, "testname", t.Name())

		task := goque.NewTask(
			"test healer"+uuid.NewString(),
			testutils.ToJSON(t, "test payload: "+uuid.NewString()),
		)
		pushToQueue(ctx, t, queueManager, task)

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			task.Type,
			goque.NoopTaskProcessor(),
			goque.WithTaskFetcherTick(10*time.Millisecond),
			goque.WithCleanerPeriod(10*time.Millisecond),
			goque.WithCleanerUpdatedAtTimeAgo(10*time.Millisecond),
			goque.WithCleanerTimeout(time.Second),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			t.Log("wait removing the task", task.ID)

			_, err := queueManager.GetTask(ctx, task.ID)
			return errors.Is(err, sql.ErrNoRows)
		}, time.Second, time.Millisecond*50)
	})
}
