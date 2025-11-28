package test

import (
	"context"
	"os"
	"testing"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/test/testutils"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	taskStorages []storages.AdvancedTaskStorage
	taskPushers  map[string]*goque.TaskPusher
)

func TestMain(m *testing.M) {
	taskStorages = testutils.SetupStorages(context.Background())

	taskPushers = make(map[string]*goque.TaskPusher, len(taskStorages))
	for _, storage := range taskStorages {
		taskPushers[storage.GetDB().DriverName()] = goque.NewTaskPusher(storage)
	}

	code := m.Run()

	testutils.TearDownStorages(taskStorages)

	os.Exit(code)
}

//nolint:gocritic
func pushToQueue(ctx context.Context, t *testing.T, storage storages.AdvancedTaskStorage, task *entity.Task) {
	t.Helper()

	pusher, ok := taskPushers[storage.GetDB().DriverName()]
	require.True(t, ok, "pusher not found")

	err := pusher.AddTaskToQueue(ctx, task)
	require.NoError(t, err)

	t.Log("added task:", task.ID, "payload:", task.Payload, "type:", task.Type)
}
