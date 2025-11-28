package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/storages/task"

	"github.com/ruko1202/goque/internal/storages/dbutils"
)

var taskStorage *task.Storage

func TestMain(m *testing.M) {
	dbConn, err := dbutils.NewPGConn(context.Background())
	if err != nil {
		panic(err)
	}
	taskStorage = task.NewStorage(dbConn)

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
