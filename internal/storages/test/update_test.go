package test

import (
	"context"
	"testing"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
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
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

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
}
