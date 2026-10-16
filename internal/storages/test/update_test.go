package test

import (
	"context"
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

func TestUpdateTask(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testUpdateTask)
}

//nolint:thelper
func testUpdateTask(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTask(ctx, t, storage, "test UpdateTask")

		task.Attempts++
		task.Status = entity.TaskStatusPending
		task.NextAttemptAt = task.NextAttemptAt.Add(time.Hour)
		err := storage.UpdateTask(ctx, task.ID, task)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
	})

	t.Run("bumps version", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTask(ctx, t, storage, "test UpdateTask version")
		version := task.Version

		// An update that changes nothing but the version must still match a row,
		// so MySQL "changed rows" never reports a false conflict.
		err := storage.UpdateTask(ctx, task.ID, task)
		require.NoError(t, err)
		require.Equal(t, version+1, task.Version)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, task.Version, dbTask.Version)
	})

	t.Run("stale version conflicts", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTask(ctx, t, storage, "test UpdateTask stale")
		stale := *task

		task.Status = entity.TaskStatusDone
		require.NoError(t, storage.UpdateTask(ctx, task.ID, task))

		stale.Status = entity.TaskStatusError
		err := storage.UpdateTask(ctx, stale.ID, &stale)
		require.ErrorIs(t, err, entity.ErrTaskVersionConflict)
		require.Equal(t, task.Version-1, stale.Version)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
	})

	t.Run("deleted row conflicts", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTaskWithStatus(ctx, t, storage, "test UpdateTask deleted "+uuid.NewString(), entity.TaskStatusDone)
		task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		updateTask(ctx, t, storage, task)

		deleted, err := storage.DeleteTasks(ctx, task.Type, []entity.TaskStatus{entity.TaskStatusDone}, time.Second)
		require.NoError(t, err)
		require.Len(t, deleted, 1)

		err = storage.UpdateTask(ctx, task.ID, task)
		require.ErrorIs(t, err, entity.ErrTaskVersionConflict)
	})

	t.Run("hard update ignores version", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTask(ctx, t, storage, "test HardUpdateTask")
		stale := *task
		require.NoError(t, storage.UpdateTask(ctx, task.ID, task))

		stale.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
		require.NoError(t, storage.HardUpdateTask(ctx, stale.ID, &stale))

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, task.Version, dbTask.Version)
	})
}
