package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

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
			actual := actualTasks[expected.ID]
			testutils.EqualTask(t, expected, actual)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		tasks, err := storage.GetTasksForProcessing(ctx, "not found", 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
