package task

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages/dbutils"
)

var storage *Storage

func TestMain(m *testing.M) {
	dbConn, err := dbutils.NewPGConn(context.Background())
	if err != nil {
		panic(err)
	}
	storage = NewStorage(dbConn)

	code := m.Run()

	dbConn.Close()
	os.Exit(code)
}

type testPayload struct {
	Data string
}

func toJSON(t *testing.T, obj any) string {
	t.Helper()

	b, err := json.Marshal(obj)
	require.NoError(t, err)

	return string(b)
}

func fromJSON(t *testing.T, j string) *testPayload {
	t.Helper()

	dest := &testPayload{}
	err := json.Unmarshal([]byte(j), dest)
	require.NoError(t, err)

	return dest
}

func equalTask(t *testing.T, expected, actual *entity.Task) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Type, actual.Type)
	require.Equal(t, expected.ExternalID, actual.ExternalID)
	require.Equal(t, fromJSON(t, expected.Payload), fromJSON(t, actual.Payload))
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.Attempts, actual.Attempts)
	require.Equal(t, expected.Errors, actual.Errors)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t,
		lo.FromPtr(expected.UpdatedAt).In(time.UTC).Round(time.Minute),
		lo.FromPtr(actual.UpdatedAt).In(time.UTC).Round(time.Minute),
	)
	require.Equal(t, expected.NextAttemptAt, actual.NextAttemptAt)
}

func makeTask(t *testing.T, taskType entity.TaskType) *entity.Task {
	t.Helper()

	return makeTaskWithStatus(t, taskType, entity.TaskStatusNew)
}

func makeTaskWithStatus(t *testing.T, taskType entity.TaskType, status entity.TaskStatus) *entity.Task {
	t.Helper()

	task := entity.NewTask(taskType, toJSON(t, &testPayload{Data: "test"}))
	task.Status = status

	err := storage.AddTask(context.Background(), task)
	require.NoError(t, err)

	return task
}
