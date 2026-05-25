package test

import (
	"context"
	"testing"
	"time"

	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/queuemanager"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/goquectx"
	"github.com/ruko1202/goque/pkg/goquestorage"
	"github.com/ruko1202/goque/test/testutils"
)

// TestWithTx_Scope pins the documented contract from goque.WithTx
// GoDoc: which storage methods honor a caller-attached *sqlx.Tx and
// which run their own internal tx and therefore ignore it.
//
// Tx-aware (rollback unwinds the operation):
//   - UpdateTask, CancelTask
//
// Not tx-aware (rollback can't touch the result, the op ran on its
// own tx via dbtx.WithinTx):
//   - ResetAttempts
//
// Pattern for tx-aware methods:
//  1. Pre-seed a task on *sqlx.DB.
//  2. Open a fresh tx; invoke the method under WithTx(ctx, tx).
//  3. Rollback. Assert the effect is GONE.
//
// Pattern for not-tx-aware methods is the inverse — assert the effect
// SURVIVED the rollback (proving the method opened its own tx and
// committed independently).
//
// If a future PR changes the WithTx scope contract these tests should
// flip — that's the signal docs (GoDoc on goque.WithTx, README outbox
// section) need to update in lockstep.
//
// Subtests run sequentially: SQLite test storage caps max_open_conn=1,
// so a parallel subtest that holds a tx would starve sibling subtests.
func TestWithTx_Scope(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testWithTxScope)
}

//nolint:thelper
func testWithTxScope(t *testing.T, storage storages.AdvancedTaskStorage) {
	// SQLite test setup caps max_open_conn at 1 (see
	// test/testutils/dbconn.go). The pattern of this test — open a tx,
	// then issue a second *sqlx.DB call to verify scope — needs at
	// least 2 concurrent connections. PG and MySQL pools are sized 25+
	// and cover the contract. This skip is a test-infrastructure
	// limit, not a product-behavior gap.
	if storage.GetDB().DriverName() == goquestorage.SqliteDriver {
		t.Skip("WithTx scope test requires concurrent tx+DB connections; SQLite test pool is single-conn (test-config limit, not a product gap)")
	}

	ctx := context.Background()

	t.Run("UpdateTask honors WithTx — rollback unwinds change", func(t *testing.T) {
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		task := seedTask(ctx, t, storage, "withtx-scope-update")
		originalStatus := task.Status

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		// UpdateTask runs through Executor(ctx) — it picks up the tx
		// and the change is bound to the caller's tx.
		task.Status = entity.TaskStatusDone
		require.NoError(t, storage.UpdateTask(goque.WithTx(ctx, tx), task.ID, task))

		require.NoError(t, tx.Rollback())

		// Rollback undid the UpdateTask.
		got, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, originalStatus, got.Status,
			"UpdateTask must honor WithTx — change should NOT survive rollback")
	})

	t.Run("ResetAttempts ignores WithTx — own internal tx", func(t *testing.T) {
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		task := seedTask(ctx, t, storage, "withtx-scope-reset")
		task.Attempts = 5
		task.Status = entity.TaskStatusError
		require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		require.NoError(t, storage.ResetAttempts(goque.WithTx(ctx, tx), task.ID))

		require.NoError(t, tx.Rollback())

		// ResetAttempts goes through dbtx.WithinTx which always opens
		// a fresh tx on the underlying *sqlx.DB — caller's rollback
		// cannot unwind it.
		got, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, int32(0), got.Attempts,
			"ResetAttempts must bypass WithTx — reset should survive rollback")
	})

	t.Run("CancelTask honors WithTx — rollback unwinds cancel", func(t *testing.T) {
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		// CancelTask lives on the high-level TaskQueueManager. It
		// internally does GetTask + UpdateTask, both of which honor
		// WithTx via Executor(ctx).
		mgr := queuemanager.NewTaskQueueManager(storage)
		task := seedTask(ctx, t, storage, "withtx-scope-cancel")
		originalStatus := task.Status

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		require.NoError(t, mgr.CancelTask(goque.WithTx(ctx, tx), task.ID))

		require.NoError(t, tx.Rollback())

		got, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, originalStatus, got.Status,
			"CancelTask must honor WithTx — cancel should NOT survive rollback")
	})

	t.Run("GetTasksForProcessing ignores WithTx — own internal tx", func(t *testing.T) {
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))
		ctx = goquectx.WithValue(ctx, "testname", t.Name())

		// GetTasksForProcessing always opens its own tx via WithinTx
		// (FOR UPDATE SKIP LOCKED must not be entangled with caller's
		// outbox tx). The new→pending flip therefore survives a caller
		// rollback.
		taskType := t.Name()
		task := seedTask(ctx, t, storage, taskType)
		// Push next_attempt_at into the past — server-side NOW() can
		// trail the Go-side clock by microseconds on MySQL, otherwise
		// the freshly-seeded row may not be picked up.
		task.NextAttemptAt = time.Now().Add(-time.Minute)
		require.NoError(t, storage.HardUpdateTask(ctx, task.ID, task))

		tx, err := storage.GetDB().BeginTxx(ctx, nil)
		require.NoError(t, err)

		fetched, err := storage.GetTasksForProcessing(goque.WithTx(ctx, tx), taskType, 10)
		require.NoError(t, err)
		require.NotEmpty(t, fetched, "must fetch the seeded task")

		require.NoError(t, tx.Rollback())

		got, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		require.Equal(t, entity.TaskStatusPending, got.Status,
			"GetTasksForProcessing must bypass WithTx — status flip should survive caller rollback")
	})
}

// seedTask inserts a fresh task via the no-tx path. Does NOT use
// makeTask from main_test.go because that runs UpdateTask under the
// hood, which under SQLite's single-connection cap can interleave
// awkwardly with the tx these tests open right after.
func seedTask(ctx context.Context, t *testing.T, storage storages.Task, taskType entity.TaskType) *entity.Task {
	t.Helper()
	task := entity.NewTask(taskType, testutils.ToJSON(t, &testutils.TestPayload{Data: "seed"}))
	require.NoError(t, storage.AddTask(ctx, task))
	return task
}
