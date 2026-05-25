// Package goquestorage provides task storage implementations for different database backends.
package goquestorage

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers "pgx" driver
	"github.com/jmoiron/sqlx"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
)

// Database driver constants. Values are the names under which the
// underlying drivers register themselves with database/sql.
const (
	// PgDriver is the legacy lib/pq driver name. goque does not
	// register "postgres" itself — callers using this constant must
	// `_ "github.com/lib/pq"` in their own code.
	//
	// Deprecated: lib/pq is in maintenance mode. Use PgxDriver for
	// new code; the storage layer works identically with either.
	PgDriver = "postgres"
	// PgxDriver is the pgx/v5 stdlib driver name. Registered by
	// this package; recommended for new code.
	PgxDriver = "pgx"
	// PgxV5Driver is the alternative name pgx/v5/stdlib registers
	// itself under (alongside "pgx"). Accept both so callers using
	// either form via sqlx.Open hit the same code path.
	PgxV5Driver = "pgx/v5"
	// MysqlDriver represents the MySQL database driver.
	MysqlDriver = "mysql"
	// SqliteDriver represents the SQLite database driver (mattn/go-sqlite3).
	SqliteDriver = "sqlite3"
)

// Config holds the configuration parameters for database connection settings.
type Config struct {
	DSN             string
	Driver          string
	MaxOpenConn     int
	MaxIdleConn     int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewDBConn creates a new database connection with configured settings.
func NewDBConn(ctx context.Context, cfg *Config) (*sqlx.DB, error) {
	xlog.Info(ctx, "Using config:",
		xfield.String("dsn", cfg.DSN),
		xfield.String("driver", cfg.Driver),
		xfield.Int("max_open_conn", cfg.MaxOpenConn),
		xfield.Int("max_idle_conn", cfg.MaxIdleConn),
		xfield.Duration("conn_max_lifetime", cfg.ConnMaxLifetime),
		xfield.Duration("conn_max_idle_time", cfg.ConnMaxIdleTime),
	)

	db, err := sqlx.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		xlog.Error(ctx, "failed to open db connection", xfield.Error(err))
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConn)
	db.SetMaxIdleConns(cfg.MaxIdleConn)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	xlog.Info(ctx, "db connection opened", xfield.String("dsn", cfg.DSN))

	if err := waitOpenConn(ctx, db, 10*time.Second); err != nil {
		xlog.Error(ctx,
			fmt.Sprintf("waiting for connection opening failed [timeout: %s]", 10*time.Second),
			xfield.Error(err),
		)
		return nil, err
	}

	return db, nil
}

func waitOpenConn(ctx context.Context, db *sqlx.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := db.PingContext(ctx)
			if err == nil {
				return nil
			}
			xlog.Error(ctx, "failed to ping db connection", xfield.Error(err))
		}
	}
}
