package task

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/goccy/go-json"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/pkg/utils/sqldb"
)

var storage *Storage

func TestMain(m *testing.M) {
	dbConn := initSQLDBConn(context.Background())
	storage = NewStorage(dbConn)

	code := m.Run()

	dbConn.Close()
	os.Exit(code)
}

func initSQLDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("DB_MAX_OPEN_CONN", 10)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	dbConn, err := sqldb.NewDBConn(ctx, &sqldb.Config{
		DSN:             viper.GetString("DB_DSN"),
		Driver:          viper.GetString("DB_DRIVER"),
		MaxIdleConn:     viper.GetInt("DB_MAX_OPEN_CONN"),
		MaxOpenConn:     viper.GetInt("DB_MAX_OPEN_CONN"),
		ConnMaxIdleTime: viper.GetDuration("DB_CONN_MAX_IDLE_CONN"),
		ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFE_CONN"),
	})
	if err != nil {
		panic(err)
	}

	return dbConn
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
	require.Equal(t, lo.FromPtr(expected.Errors), lo.FromPtr(actual.Errors))
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
	require.Equal(t,
		lo.FromPtr(expected.UpdatedAt).In(time.UTC).Round(time.Minute),
		lo.FromPtr(actual.UpdatedAt).In(time.UTC).Round(time.Minute),
	)
	require.Equal(t, expected.NextAttemptAt, actual.NextAttemptAt)
}

func makeTask(ctx context.Context, t *testing.T, taskType entity.TaskType) *entity.Task {
	t.Helper()

	return makeTaskWithStatus(ctx, t, taskType, entity.TaskStatusNew)
}

func makeTaskWithStatus(ctx context.Context, t *testing.T, taskType entity.TaskType, status entity.TaskStatus) *entity.Task {
	t.Helper()

	task := entity.NewTask(taskType, toJSON(t, &testPayload{Data: "test"}))

	err := storage.AddTask(ctx, task)
	require.NoError(t, err)

	task.Status = status
	err = storage.UpdateTask(ctx, task.ID, task)
	require.NoError(t, err)

	return task
}
