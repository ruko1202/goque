package task

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestDeleteTasks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		taskShouldDeleted := makeTaskWithStatus(ctx, t, "test delete task"+uuid.NewString(), entity.TaskStatusDone)
		taskShouldNotDeleted := makeTaskWithStatus(ctx, t, taskShouldDeleted.Type, entity.TaskStatusPending)

		tasks, err := storage.DeleteTasks(ctx, taskShouldDeleted.Type, []entity.TaskStatus{
			entity.TaskStatusDone,
		}, time.Millisecond)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		_, err = storage.GetTask(ctx, taskShouldDeleted.ID)
		require.EqualError(t, err, sql.ErrNoRows.Error())

		_, err = storage.GetTask(ctx, taskShouldNotDeleted.ID)
		require.NoError(t, err)
	})
}
