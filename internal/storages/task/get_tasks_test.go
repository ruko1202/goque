package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestGetTask(t *testing.T) {
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		taskType := "test GetTask" + uuid.NewString()
		statuses := []entity.TaskStatus{entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusDone}

		createdTasks := make([]*entity.Task, 0)
		for i := range 10 {
			createdTasks = append(createdTasks, makeTaskWithStatus(t, taskType, statuses[i%len(statuses)]))
		}

		t.Run("GetTasks", func(t *testing.T) {
			expectedTasks := lo.Filter(createdTasks, func(item *entity.Task, _ int) bool {
				return item.Status == entity.TaskStatusNew
			})
			tasks, err := storage.GetTasks(ctx, &GetTasksFilter{
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
				equalTask(t, expected, actual)
			}
		})
	})

	t.Run("empty filter", func(t *testing.T) {
		makeTask(t, "test GetTask: empty filter")

		tasks, err := storage.GetTasks(ctx, &GetTasksFilter{}, 10)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(tasks), 1)
	})

	t.Run("not found", func(t *testing.T) {
		tasks, err := storage.GetTasks(ctx, &GetTasksFilter{
			TaskType: lo.ToPtr("not found"),
		}, 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
