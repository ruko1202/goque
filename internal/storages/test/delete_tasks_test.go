package test

import (
	"context"
	"database/sql"
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

func TestDeleteTasks(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testDeleteTasks)
}

//nolint:thelper
func testDeleteTasks(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		taskShouldDeleted := makeTaskWithStatus(ctx, t, storage, "test delete task"+uuid.NewString(), entity.TaskStatusDone)
		taskShouldDeleted.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		updateTask(ctx, t, storage, taskShouldDeleted)

		taskShouldNotDeleted := makeTaskWithStatus(ctx, t, storage, taskShouldDeleted.Type, entity.TaskStatusPending)
		taskShouldNotDeleted.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		updateTask(ctx, t, storage, taskShouldNotDeleted)

		tasks, err := storage.DeleteTasks(ctx, taskShouldDeleted.Type, []entity.TaskStatus{
			entity.TaskStatusDone,
		}, time.Second)
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		_, err = storage.GetTask(ctx, taskShouldDeleted.ID)
		require.EqualError(t, err, sql.ErrNoRows.Error())

		_, err = storage.GetTask(ctx, taskShouldNotDeleted.ID)
		require.NoError(t, err)
	})
}
