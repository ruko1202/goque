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

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
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
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		taskType := "test healer" + uuid.NewString()

		expectedCurredTaskIDs := make([]uuid.UUID, 0)
		for _, status := range []entity.TaskStatus{
			entity.TaskStatusPending,
			entity.TaskStatusProcessing,
		} {
			task := entity.NewTask(
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
			goque.WithHealerPeriod(10*time.Millisecond),
			goque.WithHealerUpdatedAtTimeAgo(time.Hour),
			goque.WithHealerTimeout(time.Second),
		)
		err := goq.Run(ctx)
		require.NoError(t, err)
		defer goq.Stop()

		require.Eventually(t, func() bool {
			tasks, err := queueManager.GetTasks(ctx, &dbentity.GetTasksFilter{
				IDs:      expectedCurredTaskIDs,
				TaskType: lo.ToPtr(taskType),
				Status:   lo.ToPtr(entity.TaskStatusError),
			}, 100)
			require.NoError(t, err)
			return len(expectedCurredTaskIDs) == len(tasks)
		}, time.Second, time.Millisecond*50)
	})
}
