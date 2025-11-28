// Package dbutils provides database utilities for connection management and transactions.
package dbutils

import (
	"context"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
)

// NewPGConn creates a new PostgreSQL database connection with configured settings.
func NewPGConn(ctx context.Context) (*sqlx.DB, error) {
	viper.SetDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("DB_DRIVER", "postgres")
	viper.SetDefault("DB_MAX_OPEN_CONN", 10)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	slog.Info("Using config:",
		slog.Any("DB_DSN", viper.GetString("DB_DSN")),
		slog.Any("DB_DRIVER", viper.GetString("DB_DRIVER")),
		slog.Any("DB_MAX_OPEN_CONN", viper.GetString("DB_MAX_OPEN_CONN")),
		slog.Any("DB_MAX_IDLE_CONN", viper.GetString("DB_MAX_IDLE_CONN")),
	)

	db, err := sqlx.Open(
		viper.GetString("DB_DRIVER"),
		viper.GetString("DB_DSN"),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to open db connection", slog.Any("err", err))
		return nil, err
	}
	db.SetMaxOpenConns(viper.GetInt("DB_MAX_OPEN_CONN"))
	db.SetMaxIdleConns(viper.GetInt("DB_MAX_IDLE_CONN"))

	slog.InfoContext(ctx, "db connection opened")

	if err := waitOpenConn(ctx, db, 10*time.Second); err != nil {
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
