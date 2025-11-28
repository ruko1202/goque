package test

import (
	"context"
	"fmt"
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

func TestCureTasks(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testCureTasks)
}

//nolint:thelper
func testCureTasks(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := makeTaskWithStatus(ctx, t, storage, "test cure task"+uuid.NewString(), entity.TaskStatusPending)
		task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Minute))
		updateTask(ctx, t, storage, task)

		tasks, err := storage.CureTasks(ctx, task.Type, []entity.TaskStatus{
			entity.TaskStatusPending,
		}, time.Millisecond, "comment")
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		actualTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		task.Status = entity.TaskStatusError
		task.Errors = lo.ToPtr(fmt.Sprintf("attempt %d: comment\n", task.Attempts))
		testutils.EqualTask(t, task, actualTask)
	})
}
