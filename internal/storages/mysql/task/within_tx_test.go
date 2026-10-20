package mysqltask_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbtx"
	mysqltask "github.com/ruko1202/goque/internal/storages/mysql/task"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/pkg/goquestorage"
	"github.com/ruko1202/goque/test/testutils"
)

// TestWithinTx_LocksOnlyMatchedRows pins the fix for the fetch/worker deadlock
// (Error 1213). Under REPEATABLE READ the fetch's locking range scan also takes a
// next-key lock on the first index record past the range, which belongs to a task
// a worker is about to move to processing; the worker's UPDATE then waits for the
// fetch tx, and the fetch's own status UPDATE waits for the worker. The storage's
// txs run at READ COMMITTED, so the scan locks only the rows it returns and the
// worker's write goes through while the fetch tx is still open.
func TestWithinTx_LocksOnlyMatchedRows(t *testing.T) {
	if driver := viper.GetString("DB_DRIVER"); driver != "" && driver != goquestorage.MysqlDriver {
		t.Skipf("MySQL-only test, DB_DRIVER=%s", driver)
	}

	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
	storage := mysqltask.NewStorage(testutils.MysqlDBConn(ctx))
	t.Cleanup(func() { _ = storage.GetDB().Close() })

	taskType := "test mysql within tx " + uuid.NewString()
	t.Cleanup(func() {
		_, err := storage.DeleteTasks(ctx, taskType, []entity.TaskStatus{
			entity.TaskStatusNew, entity.TaskStatusPending, entity.TaskStatusProcessing,
		}, 0)
		require.NoError(t, err)
	})

	due := addTask(ctx, t, storage, taskType, entity.TaskStatusNew)
	// In goque_task_type_status_next_attempt_at_idx the pending task is the first
	// record past the fetch's (type, new, <= now) range.
	inFlight := addTask(ctx, t, storage, taskType, entity.TaskStatusPending)

	err := mysqltask.WithinTx(storage, ctx, func(txCtx context.Context) error {
		tasks, err := mysqltask.SelectTasksForProcessing(storage, txCtx, taskType, 10)
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		require.Equal(t, due.ID.String(), tasks[0].ID)

		// The worker writes on its own connection, outside the fetch tx.
		writeCtx, cancel := context.WithTimeout(dbtx.WithoutTx(txCtx), 3*time.Second)
		defer cancel()

		inFlight.Status = entity.TaskStatusProcessing
		return storage.UpdateTask(writeCtx, inFlight.ID, inFlight)
	})
	require.NoError(t, err, "a worker's write must not wait for the fetch tx")
}

func addTask(
	ctx context.Context,
	t *testing.T,
	storage *mysqltask.Storage,
	taskType entity.TaskType,
	status entity.TaskStatus,
) *entity.Task {
	t.Helper()

	task := entity.NewTask(taskType, testutils.ToJSON(t, &testutils.TestPayload{Data: "test"}))
	require.NoError(t, storage.AddTask(ctx, task))

	task.Status = status
	task.NextAttemptAt = xtime.Now().Add(-time.Hour)
	require.NoError(t, storage.UpdateTask(ctx, task.ID, task))

	return task
}
