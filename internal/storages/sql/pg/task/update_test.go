package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestUpdateTask(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		task := makeTask(ctx, t, "test UpdateTask")

		task.Attempts++
		task.Status = entity.TaskStatusPending
		task.NextAttemptAt = task.NextAttemptAt.Add(time.Hour)
		err := storage.UpdateTask(ctx, task.ID, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		equalTask(t, task, dbTask)
	})
}
