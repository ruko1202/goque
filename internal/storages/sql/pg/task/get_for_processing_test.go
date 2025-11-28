package task

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestGetTaskForProcessing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		taskType := "test GetTaskForProcessing" + uuid.NewString()
		statuses := []entity.TaskStatus{entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusError}

		expectedTasks := make([]*entity.Task, 0)
		for i := range 10 {
			status := statuses[i%len(statuses)]
			task := makeTaskWithStatus(ctx, t, taskType, status)
			if status == entity.TaskStatusNew || status == entity.TaskStatusError {
				expectedTasks = append(expectedTasks, task)
			}
		}

		time.Sleep(100 * time.Millisecond)
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
		t.Parallel()
		tasks, err := storage.GetTasksForProcessing(ctx, "not found", 10)
		require.NoError(t, err)
		require.Equal(t, 0, len(tasks))
	})
}
