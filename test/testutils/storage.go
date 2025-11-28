package testutils

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ruko1202/goque/internal/storages"
	mysqltask "github.com/ruko1202/goque/internal/storages/mysql/task"
	pgtask "github.com/ruko1202/goque/internal/storages/pg/task"
	"github.com/ruko1202/goque/pkg/goquestorage"
)

// SetupStorages initializes test storages based on DB_DRIVER environment variable.
// If DB_DRIVER is not set, initializes all available databases (PostgreSQL and MySQL).
func SetupStorages(ctx context.Context) []storages.AdvancedTaskStorage {
	switch os.Getenv("DB_DRIVER") {
	case goquestorage.PgDriver:
		return []storages.AdvancedTaskStorage{pgtask.NewStorage(PgDBConn(ctx))}
	case goquestorage.MysqlDriver:
		return []storages.AdvancedTaskStorage{mysqltask.NewStorage(MysqlDBConn(ctx))}
	default:
		slog.InfoContext(ctx, fmt.Sprintf("DB_DRIVER doesn't define. Init storages for all DBs: %s",
			goquestorage.PgDriver+","+goquestorage.MysqlDriver,
		))
		return []storages.AdvancedTaskStorage{
			pgtask.NewStorage(PgDBConn(ctx)),
			mysqltask.NewStorage(MysqlDBConn(ctx)),
		}
	}
}

// TearDownStorages closes all database connections for test cleanup.
func TearDownStorages(taskStorages []storages.AdvancedTaskStorage) {
	for _, storage := range taskStorages {
		_ = storage.GetDB().Close()
	}
}
