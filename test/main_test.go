package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/queuemngr"
	"github.com/ruko1202/goque/internal/storages/dbutils"
	"github.com/ruko1202/goque/internal/storages/task"
)

var (
	taskStorage *task.Storage
	queueMngr   *queuemngr.QueueMngr
)

func TestMain(m *testing.M) {
	dbConn, err := dbutils.NewPGConn(context.Background())
	if err != nil {
		panic(err)
	}
	taskStorage = task.NewStorage(dbConn)

	queueMngr = queuemngr.NewQueueMngr(taskStorage)
	code := m.Run()

	dbConn.Close()
	os.Exit(code)
}

func toJSON(t *testing.T, obj any) string {
	t.Helper()

	b, err := json.Marshal(obj)
	require.NoError(t, err)

	return string(b)
}

//nolint:gocritic
func pushToQueue(ctx context.Context, t *testing.T, task *entity.Task) {
	t.Helper()

	err := queueMngr.AddTaskToQueue(ctx, task)
	require.NoError(t, err)

	t.Log("added task:", task.ID, "payload:", task.Payload, "type:", task.Type)
}
