package config

import (
	"context"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

// NewDB creates a new sqlx database connection based on configuration.
// If connection fails, it calls xlog.Fatal and terminates the application.
func NewDB(ctx context.Context, driver, dsn string) *sqlx.DB {
	db, err := sqlx.Open(driver, dsn)
	if err != nil {
		xlog.Fatal(ctx, "Failed to open database", zap.Error(err))
	}

	if err := db.Ping(); err != nil {
		db.Close()
		xlog.Fatal(ctx, "Failed to ping database", zap.Error(err))
	}

	return db
}
