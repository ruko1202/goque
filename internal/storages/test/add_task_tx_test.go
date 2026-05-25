package test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/goquectx"
	"github.com/ruko1202/goque/test/testutils"
)

// TestAddTaskTx verifies that AddTask honors a *sqlx.Tx carried in ctx via
// goque.WithTx: commit makes the task visible, rollback discards it, and
// the absence of a tx leaves existing behavior unchanged. Covers the
// transactional outbox pattern (RUK-123).
func TestAddTaskTx(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testAddTaskTx)
}

//nolint:thelper
func testAddTaskTx(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("commit makes task visible", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		task := entity.NewTask("test", testutils.ToJSON(t, testutils.TestPayload{Data: "commit"}))
		require.NoError(t, storage.AddTask(goque.WithTx(ctx, tx), task))

		require.NoError(t, tx.Commit())

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
	})

	t.Run("rollback discards task", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		task := entity.NewTask("test", testutils.ToJSON(t, testutils.TestPayload{Data: "rollback"}))
		require.NoError(t, storage.AddTask(goque.WithTx(ctx, tx), task))

		require.NoError(t, tx.Rollback())

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.True(t,
			errors.Is(err, sql.ErrNoRows),
			"expected sql.ErrNoRows after rollback, got %v", err,
		)
		require.Nil(t, dbTask, "no task should be returned after rollback")
	})

	t.Run("no tx in ctx falls back to db", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		task := entity.NewTask("test", testutils.ToJSON(t, testutils.TestPayload{Data: "no-tx"}))
		require.NoError(t, storage.AddTask(ctx, task))

		dbTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		testutils.EqualTask(t, task, dbTask)
	})
}
