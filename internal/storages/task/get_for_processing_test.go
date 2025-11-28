package task

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestGetTaskForProcessing(t *testing.T) {
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		taskType := "test GetTaskForProcessing" + uuid.NewString()
		statuses := []entity.TaskStatus{entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusError}

		createdTasks := make([]*entity.Task, 0)
		for i := range 10 {
			createdTasks = append(createdTasks, makeTaskWithStatus(t, taskType, statuses[i%len(statuses)]))
		}

		expectedTasks := lo.Filter(createdTasks, func(item *entity.Task, _ int) bool {
			return item.Status == entity.TaskStatusNew || item.Status == entity.TaskStatusError
		})

		tasks, err := storage.GetTasksForProcessing(ctx, taskType, 10)
		require.NoError(t, err)
		require.Equal(t, len(expectedTasks), len(tasks))

		for i, expected := range expectedTasks {
			expected.Status = entity.TaskStatusPending
			actual := tasks[i]
			equalTask(t, expected, actual)
		}
	})

	t.Run("not found", func(t *testing.T) {
		tasks, err := storage.GetTasksForProcessing(ctx, "not found", 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
