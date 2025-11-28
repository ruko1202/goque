package testutils

import (
	"context"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"           // PostgreSQL driver
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/spf13/viper"

	"github.com/ruko1202/goque/pkg/goquestorage"
)

var availableDBs = map[string]struct{}{
	goquestorage.PgDriver:     {},
	goquestorage.MysqlDriver:  {},
	goquestorage.SqliteDriver: {},
}

// PgDBConn creates a PostgreSQL database connection for testing.
func PgDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("DB_DRIVER", goquestorage.PgDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 10)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	return DBConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// MysqlDBConn creates a MySQL database connection for testing.
func MysqlDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC")
	viper.SetDefault("DB_DRIVER", goquestorage.MysqlDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 10)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	return DBConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// SqliteDBConn creates a Sqlite database connection for testing.
func SqliteDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "goque.sqlite.db")
	viper.SetDefault("DB_DRIVER", goquestorage.SqliteDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 1) // SQLite in-memory works best with single connection
	viper.SetDefault("DB_MAX_IDLE_CONN", 1)

	path, err := GetPathFromRoot(viper.GetString("DB_DSN"))
	if err != nil {
		panic(err)
	}

	return DBConn(ctx, path, viper.GetString("DB_DRIVER"))
}

// DBConn creates a database connection using the specified DSN and driver.
func DBConn(ctx context.Context, dsn, driver string) *sqlx.DB {
	dbConn, err := goquestorage.NewDBConn(ctx, &goquestorage.Config{
		DSN:             dsn,
		Driver:          driver,
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
