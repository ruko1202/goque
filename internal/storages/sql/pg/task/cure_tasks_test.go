package task

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

func TestCureTasks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		task := makeTaskWithStatus(ctx, t, "test cure task"+uuid.NewString(), entity.TaskStatusPending)

		time.Sleep(100 * time.Millisecond)
		tasks, err := storage.CureTasks(ctx, task.Type, []entity.TaskStatus{
			entity.TaskStatusPending,
		}, time.Millisecond, "comment")
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		actualTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		task.Status = entity.TaskStatusError
		task.Errors = lo.ToPtr(fmt.Sprintf("attempt %d: comment\n", task.Attempts))
		equalTask(t, task, actualTask)
	})
}
