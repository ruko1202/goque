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

func TestStaleWorkerAfterCure(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testStaleWorkerAfterCure)
}

// testStaleWorkerAfterCure reproduces the healer/worker race: worker A holds a
// task for too long, the healer cures it, worker B picks it up and finishes it.
// A's late write must not overwrite B's result.
//
//nolint:thelper
func testStaleWorkerAfterCure(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zaptest.NewLogger(t)))

	taskType := "test stale worker " + uuid.NewString()
	task := makeTask(ctx, t, storage, taskType)
	task.NextAttemptAt = xtime.Now().Add(-time.Hour)
	updateTask(ctx, t, storage, task)

	fetched, err := storage.GetTasksForProcessing(ctx, taskType, 1)
	require.NoError(t, err)
	require.Len(t, fetched, 1)
	workerA := fetched[0]

	workerA.Status = entity.TaskStatusProcessing
	require.NoError(t, storage.UpdateTask(ctx, workerA.ID, workerA))

	// Age the row through a separate object, so worker A's version is untouched.
	aged := *workerA
	aged.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
	updateTask(ctx, t, storage, &aged)

	cured, err := storage.CureTasks(ctx, taskType, entity.TaskStatusProcessing, time.Minute, "stuck")
	require.NoError(t, err)
	require.Len(t, cured, 1)

	fetched, err = storage.GetTasksForProcessing(ctx, taskType, 1)
	require.NoError(t, err)
	require.Len(t, fetched, 1)
	workerB := fetched[0]

	workerB.Status = entity.TaskStatusProcessing
	require.NoError(t, storage.UpdateTask(ctx, workerB.ID, workerB))
	workerB.Status = entity.TaskStatusDone
	require.NoError(t, storage.UpdateTask(ctx, workerB.ID, workerB))

	workerA.Status = entity.TaskStatusError
	err = storage.UpdateTask(ctx, workerA.ID, workerA)
	require.ErrorIs(t, err, entity.ErrTaskVersionConflict)

	dbTask, err := storage.GetTask(ctx, workerB.ID)
	require.NoError(t, err)
	testutils.EqualTask(t, workerB, dbTask)
	require.Equal(t, entity.TaskStatusDone, dbTask.Status)
}
