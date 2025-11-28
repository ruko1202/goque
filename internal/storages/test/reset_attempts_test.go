package test

import (
	"context"
	"testing"

	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/test/testutils"
)

func TestResetAttempts(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testResetAttempts)
}

//nolint:thelper
func testResetAttempts(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, zaptest.NewLogger(t))

		task := makeTask(ctx, t, storage, "test ResetAttempts")
		task.Attempts = 10
		task.Status = entity.TaskStatusAttemptsLeft

		updateTask(ctx, t, storage, task)

		err := storage.ResetAttempts(ctx, task.ID)
		require.NoError(t, err)

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)

		require.Contains(t, lo.FromPtr(dbTask.Errors), "reset attempts")
		task.Attempts = 0
		task.Status = entity.TaskStatusNew
		task.Errors = dbTask.Errors
		testutils.EqualTask(t, task, dbTask)
	})
}
