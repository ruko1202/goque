// Package goquestorage provides task storage implementations for different database backends.
package goquestorage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

// Database driver constants.
const (
	// PgDriver represents the PostgreSQL database driver.
	PgDriver = "postgres"
	// MysqlDriver represents the MySQL database driver.
	MysqlDriver  = "mysql"
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
	slog.InfoContext(ctx, "Using config:",
		slog.Any("dsn", cfg.DSN),
		slog.Any("driver", cfg.Driver),
		slog.Any("max_open_conn", cfg.MaxOpenConn),
		slog.Any("max_idle_conn", cfg.MaxIdleConn),
		slog.Any("conn_max_lifetime", cfg.ConnMaxLifetime),
		slog.Any("conn_max_idle_time", cfg.ConnMaxIdleTime),
	)

	db, err := sqlx.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		slog.ErrorContext(ctx, "failed to open db connection", slog.Any("err", err))
		return nil, err
	}
	db.SetMaxOpenConns(cfg.MaxOpenConn)
	db.SetMaxIdleConns(cfg.MaxIdleConn)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	slog.InfoContext(ctx, "db connection opened")

	if err := waitOpenConn(ctx, db, 10*time.Second); err != nil {
		slog.ErrorContext(ctx,
			fmt.Sprintf("waiting for connection opening failed [timeout: %s]", 10*time.Second),
			slog.Any("err", err),
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
			slog.ErrorContext(ctx, "failed to ping db connection", slog.Any("err", err))
		}
	}
}
