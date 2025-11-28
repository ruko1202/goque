package testutils

import (
	"context"
	"fmt"

	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ruko1202/goque/internal/storages"
	mysqltask "github.com/ruko1202/goque/internal/storages/mysql/task"
	pgtask "github.com/ruko1202/goque/internal/storages/pg/task"
	sqlitetask "github.com/ruko1202/goque/internal/storages/sqlite"
	"github.com/ruko1202/goque/pkg/goquestorage"
)

// SetupStorages initializes test storages based on DB_DRIVER environment variable.
// If DB_DRIVER is not set, initializes all available databases (PostgreSQL, MySQL and Sqlite).
func SetupStorages(ctx context.Context) []storages.AdvancedTaskStorage {
	dbDriver := viper.GetString("DB_DRIVER")

	if _, ok := availableDBs[dbDriver]; ok {
		return []storages.AdvancedTaskStorage{setupStorage(ctx, dbDriver)}
	}

	xlog.Info(ctx, "DB_DRIVER doesn't define. Init storages for all DBs", zap.Any("dbs", availableDBs))
	return lo.MapToSlice(availableDBs, func(dbDriver string, _ struct{}) storages.AdvancedTaskStorage {
		return setupStorage(ctx, dbDriver)
	})
}
func setupStorage(ctx context.Context, driver string) storages.AdvancedTaskStorage {
	switch driver {
	case goquestorage.PgDriver:
		return pgtask.NewStorage(PgDBConn(ctx))
	case goquestorage.MysqlDriver:
		return mysqltask.NewStorage(MysqlDBConn(ctx))
	case goquestorage.SqliteDriver:
		return sqlitetask.NewStorage(SqliteDBConn(ctx))
	default:
		panic(fmt.Sprintf("Unsupported driver: %s", driver))
	}
}

// TearDownStorages closes all database connections for test cleanup.
func TearDownStorages(taskStorages []storages.AdvancedTaskStorage) {
	for _, storage := range taskStorages {
		_ = storage.GetDB().Close()
	}
}
