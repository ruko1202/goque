package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestGetTaskForProcessing(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGetTaskForProcessing)
}

//nolint:thelper
func testGetTaskForProcessing(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		taskType := "test GetTaskForProcessing" + uuid.NewString()
		statuses := []entity.TaskStatus{entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusError}

		expectedTasks := make([]*entity.Task, 0)
		for i := range 10 {
			status := statuses[i%len(statuses)]
			task := makeTaskWithStatus(ctx, t, storage, taskType, status)
			task.NextAttemptAt = xtime.Now().Add(-time.Hour)
			updateTask(ctx, t, storage, task)

			if status == entity.TaskStatusNew || status == entity.TaskStatusError {
				expectedTasks = append(expectedTasks, task)
			}
		}

		tasks, err := storage.GetTasksForProcessing(ctx, taskType, 10)
		require.NoError(t, err)
		require.Equal(t, len(expectedTasks), len(tasks))

		actualTasks := lo.SliceToMap(tasks, func(item *entity.Task) (uuid.UUID, *entity.Task) {
			return item.ID, item
		})

		for _, expected := range expectedTasks {
			expected.Status = entity.TaskStatusPending
			expected.Version++
			actual := actualTasks[expected.ID]
			testutils.EqualTask(t, expected, actual)

			dbTask, err := storage.GetTask(ctx, expected.ID)
			require.NoError(t, err)
			require.Equal(t, actual.Version, dbTask.Version)
		}
	})

	t.Run("due now", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		// Enqueue past the middle of a second: a column that rounds to whole
		// seconds would push next_attempt_at into the next second.
		if frac := time.Duration(xtime.Now().Nanosecond()); frac < 600*time.Millisecond {
			time.Sleep(600*time.Millisecond - frac)
		}

		taskType := "test GetTaskForProcessing due now " + uuid.NewString()
		task := entity.NewTask(taskType, testutils.ToJSON(t, &testutils.TestPayload{Data: "test"}))
		require.NoError(t, storage.AddTask(ctx, task))

		tasks, err := storage.GetTasksForProcessing(ctx, taskType, 10)
		require.NoError(t, err)
		require.Len(t, tasks, 1, "a task due now must be fetchable immediately")
		require.Equal(t, task.ID, tasks[0].ID)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		tasks, err := storage.GetTasksForProcessing(ctx, "not found", 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
