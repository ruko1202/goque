package test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/storages"

	"github.com/ruko1202/goque/test/testutils"

	"github.com/ruko1202/goque/internal/storages/dbentity"
)

func TestGetTask(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGetTask)
}

//nolint:thelper
func testGetTask(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		taskType := "test GetTask" + uuid.NewString()
		statuses := []entity.TaskStatus{entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusDone}

		createdTasks := make([]*entity.Task, 0)
		for i := range 10 {
			createdTasks = append(createdTasks, makeTaskWithStatus(ctx, t, storage, taskType, statuses[i%len(statuses)]))
		}

		t.Run("GetTasks", func(t *testing.T) {
			t.Parallel()
			expectedTasks := lo.Filter(createdTasks, func(item *entity.Task, _ int) bool {
				return item.Status == entity.TaskStatusNew
			})
			tasks, err := storage.GetTasks(ctx, &dbentity.GetTasksFilter{
				Status:   lo.ToPtr(entity.TaskStatusNew),
				TaskType: &taskType,
			}, 10)
			require.NoError(t, err)
			require.Equal(t, len(expectedTasks), len(tasks))

			tasksMap := lo.SliceToMap(tasks, func(item *entity.Task) (uuid.UUID, *entity.Task) {
				return item.ID, item
			})
			for _, expected := range expectedTasks {
				actual := tasksMap[expected.ID]
				require.NotNil(t, actual)
				testutils.EqualTask(t, expected, actual)
			}
		})
	})

	t.Run("empty filter", func(t *testing.T) {
		t.Parallel()
		makeTask(ctx, t, storage, "test GetTask: empty filter")

		tasks, err := storage.GetTasks(ctx, &dbentity.GetTasksFilter{}, 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		tasks, err := storage.GetTasks(ctx, &dbentity.GetTasksFilter{
			TaskType: lo.ToPtr("not found"),
		}, 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
