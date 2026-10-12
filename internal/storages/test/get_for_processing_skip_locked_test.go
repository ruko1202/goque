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

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/pkg/goquestorage"
	"github.com/ruko1202/goque/test/testutils"
)

// TestGetTaskForProcessing_SkipLocked checks that rows locked by another
// transaction are skipped instead of blocking the fetch.
func TestGetTaskForProcessing_SkipLocked(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testGetTaskForProcessingSkipLocked)
}

//nolint:thelper
func testGetTaskForProcessingSkipLocked(t *testing.T, storage storages.AdvancedTaskStorage) {
	// SQLite has no row locks and the test pool is single-connection.
	if storage.GetDB().DriverName() == goquestorage.SqliteDriver {
		t.Skip("SQLite has no row-level locks")
	}

	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))
	taskType := "test GetTaskForProcessing skip locked " + uuid.NewString()

	tasks := make([]*entity.Task, 0, 6)
	for range 6 {
		task := makeTask(ctx, t, storage, taskType)
		task.NextAttemptAt = xtime.Now().Add(-time.Hour)
		updateTask(ctx, t, storage, task)
		tasks = append(tasks, task)
	}
	locked, free := tasks[:3], tasks[3:]

	tx, err := storage.GetDB().BeginTxx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// An UPDATE inside an open tx holds the row lock until the tx ends.
	for _, task := range locked {
		require.NoError(t, storage.UpdateTask(goque.WithTx(ctx, tx), task.ID, task))
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	fetched, err := storage.GetTasksForProcessing(fetchCtx, taskType, int64(len(tasks)))
	require.NoError(t, err, "fetch must not wait for rows locked by another tx")
	require.ElementsMatch(t,
		lo.Map(free, func(task *entity.Task, _ int) uuid.UUID { return task.ID }),
		lo.Map(fetched, func(task *entity.Task, _ int) uuid.UUID { return task.ID }),
	)
}
