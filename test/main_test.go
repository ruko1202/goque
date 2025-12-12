package test

import (
	"context"
	"os"
	"testing"

	"github.com/ruko1202/goque"
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

func pushToQueue(ctx context.Context, t *testing.T, queueManager goque.TaskQueueManager, task *goque.Task) {
	t.Helper()

	err := queueManager.AddTaskToQueue(ctx, task)
	require.NoError(t, err)

	t.Log("added task:", task.ID, "payload:", task.Payload, "type:", task.Type)
}
