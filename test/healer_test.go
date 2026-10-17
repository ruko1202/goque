package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/utils/goquectx"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/storages/dbentity"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestHealer(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testHealer)
}

func testHealer(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Helper()
	t.Parallel()
	ctx := context.Background()
	queueManager := goque.NewTaskQueueManager(storage)

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		taskType := "test healer" + uuid.NewString()

		expectedCurredTaskIDs := make([]uuid.UUID, 0, 2)
		for _, status := range []goque.TaskStatus{
			goque.TaskStatusPending,
			goque.TaskStatusProcessing,
		} {
			task := goque.NewTask(
				taskType,
				testutils.ToJSON(t, "test payload: "+uuid.NewString()),
			)
			task.Status = status
			task.AddError(errors.New("processing error"))
			task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-2 * time.Hour))
			expectedCurredTaskIDs = append(expectedCurredTaskIDs, task.ID)

			pushToQueue(ctx, t, queueManager, task)
		}

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			taskType,
			goque.NoopTaskProcessor(),
			goque.WithHealerPeriod(goque.NewStaticValue(10*time.Millisecond)),
			goque.WithHealerUpdatedAtTimeAgo(goque.NewStaticValue(time.Hour)),
			goque.WithHealerTimeout(goque.NewStaticValue(time.Second)),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			tasks, err := queueManager.GetTasks(ctx, &dbentity.GetTasksFilter{
				IDs:      expectedCurredTaskIDs,
				TaskType: lo.ToPtr(taskType),
				Status:   lo.ToPtr(goque.TaskStatusError),
			}, 100)
			require.NoError(t, err)
			return len(expectedCurredTaskIDs) == len(tasks)
		}, time.Second, time.Millisecond*50)
	})
	// Through the real goque wiring the processing floor is set, so the healer
	// cures processing and pending in separate CureTasks calls: only the
	// processing task consumes an attempt and gets the comment.
	t.Run("only processing cures consume an attempt", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		taskType := "test healer attempts" + uuid.NewString()
		seed := func(status goque.TaskStatus, attempts int32, errs *string) *goque.Task {
			task := goque.NewTask(taskType, testutils.ToJSON(t, "payload"))
			pushToQueue(ctx, t, queueManager, task)
			task.Status = status
			task.Attempts = attempts
			task.Errors = errs
			task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-2 * time.Hour))
			// Out of the fetcher's reach, so the cured state stays observable.
			task.NextAttemptAt = xtime.Now().Add(time.Hour)
			require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))
			return task
		}
		pending := seed(goque.TaskStatusPending, 2, lo.ToPtr("attempt 2: boom\n"))
		// NULL errors: the comment must still be appended (COALESCE).
		processing := seed(goque.TaskStatusProcessing, 1, nil)

		goq := goque.NewGoque(storage)
		goq.RegisterProcessor(
			taskType,
			goque.NoopTaskProcessor(),
			goque.WithHealerPeriod(goque.NewStaticValue(10*time.Millisecond)),
			goque.WithHealerUpdatedAtTimeAgo(goque.NewStaticValue(time.Hour)),
			goque.WithHealerTimeout(goque.NewStaticValue(time.Second)),
		)
		require.NoError(t, goq.Run(ctx))
		defer goq.Stop()

		ids := []uuid.UUID{pending.ID, processing.ID}
		require.Eventually(t, func() bool {
			tasks, err := queueManager.GetTasks(ctx, &dbentity.GetTasksFilter{
				IDs:      ids,
				TaskType: lo.ToPtr(taskType),
				Status:   lo.ToPtr(goque.TaskStatusError),
			}, 100)
			require.NoError(t, err)
			return len(tasks) == len(ids)
		}, time.Second, 10*time.Millisecond)

		actualPending, err := queueManager.GetTask(ctx, pending.ID)
		require.NoError(t, err)
		require.Equal(t, int32(2), actualPending.Attempts)
		require.Equal(t, "attempt 2: boom\n", lo.FromPtr(actualPending.Errors))
		require.Equal(t, pending.Version+1, actualPending.Version)

		actualProcessing, err := queueManager.GetTask(ctx, processing.ID)
		require.NoError(t, err)
		require.Equal(t, int32(2), actualProcessing.Attempts)
		require.Equal(t, "attempt 1: task was stuck\n", lo.FromPtr(actualProcessing.Errors))
		require.Equal(t, processing.Version+1, actualProcessing.Version)
	})
}
