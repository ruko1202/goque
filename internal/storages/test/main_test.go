// Package test provides integration tests for storage implementations.
//
// Database Selection:
// Tests can run against specific database by setting DB_DRIVER environment variable:
//   - DB_DRIVER=postgres  - Run tests only on PostgreSQL
//   - DB_DRIVER=mysql     - Run tests only on MySQL
//   - (not set)           - Run tests on all available databases (default)
package test

import (
	"context"
	"os"
	"testing"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/test/testutils"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	taskStorages []storages.AdvancedTaskStorage
)

func TestMain(m *testing.M) {
	taskStorages = testutils.SetupStorages(context.Background())

	code := m.Run()

	testutils.TearDownStorages(taskStorages)

	os.Exit(code)
}

func makeTask(ctx context.Context, t *testing.T, storage storages.Task, taskType entity.TaskType) *entity.Task {
	t.Helper()

	return makeTaskWithStatus(ctx, t, storage, taskType, entity.TaskStatusNew)
}

func makeTaskWithStatus(ctx context.Context, t *testing.T, storage storages.Task, taskType entity.TaskType, status entity.TaskStatus) *entity.Task {
	t.Helper()

	task := entity.NewTask(taskType, testutils.ToJSON(t, &testutils.TestPayload{Data: "test"}))

	err := storage.AddTask(ctx, task)
	require.NoError(t, err)

	task.Status = status
	err = storage.UpdateTask(ctx, task.ID, task)
	require.NoError(t, err)

	return task
}

func updateTask(ctx context.Context, t *testing.T, storage storages.AdvancedTaskStorage, task *entity.Task) {
	t.Helper()

	err := storage.HardUpdateTask(ctx, task.ID, task)
	require.NoError(t, err)
}
