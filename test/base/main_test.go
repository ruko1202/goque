package base

import (
	"context"
	"os"
	"testing"

	"github.com/goccy/go-json"

	"github.com/ruko1202/goque/internal/entity"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/queuemngr"
	"github.com/ruko1202/goque/internal/storages/sql/pg/task"
	"github.com/ruko1202/goque/pkg/utils/sqldb"
)

var (
	taskStorage *task.Storage
	queueMngr   *queuemngr.QueueMngr
)

func TestMain(m *testing.M) {
	dbConn := initSQLDBConn(context.Background())
	taskStorage = task.NewStorage(dbConn)

	queueMngr = queuemngr.NewQueueMngr(taskStorage)
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
