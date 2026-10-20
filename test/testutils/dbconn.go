package testutils

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/spf13/viper"

	// PgDriver ("pgx") is registered by importing goquestorage.
	"github.com/ruko1202/goque/pkg/goquestorage"
)

// sqliteDSNParams is the DSN tuning the README recommends for SQLite: WAL lets
// readers run beside the single writer, immediate transactions avoid the
// SHARED->RESERVED upgrade deadlock, and busy_timeout waits out a held lock.
const sqliteDSNParams = "?_txlock=immediate&_journal_mode=WAL&_busy_timeout=5000"

var availableDBs = []string{
	goquestorage.SqliteDriver,
	goquestorage.PgxDriver,
	goquestorage.MysqlDriver,
}

// PgDBConn creates a PostgreSQL database connection for testing.
func PgDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("DB_DRIVER", goquestorage.PgxDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 50)
	viper.SetDefault("DB_MAX_IDLE_CONN", 50) // below the workers count, database/sql reopens connections
	viper.SetDefault("DB_CONN_MAX_IDLE_CONN", 5*time.Minute)
	viper.SetDefault("DB_CONN_MAX_LIFE_CONN", 1*time.Minute)

	return DBConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// MysqlDBConn creates a MySQL database connection for testing.
func MysqlDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC")
	viper.SetDefault("DB_DRIVER", goquestorage.MysqlDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 50)
	viper.SetDefault("DB_MAX_IDLE_CONN", 50) // below the workers count, database/sql reopens connections
	viper.SetDefault("DB_CONN_MAX_IDLE_CONN", 5*time.Minute)
	viper.SetDefault("DB_CONN_MAX_LIFE_CONN", 1*time.Minute)

	return DBConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// SqliteDBConn creates a Sqlite database connection for testing.
func SqliteDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "goque.sqlite.db")
	viper.SetDefault("DB_DRIVER", goquestorage.SqliteDriver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 1) // SQLite serializes writes, extra connections only add SQLITE_BUSY
	viper.SetDefault("DB_MAX_IDLE_CONN", 1)
	viper.SetDefault("DB_CONN_MAX_IDLE_CONN", 5*time.Minute)
	viper.SetDefault("DB_CONN_MAX_LIFE_CONN", 1*time.Minute)

	path, err := GetPathFromRoot(viper.GetString("DB_DSN"))
	if err != nil {
		panic(err)
	}

	if !strings.Contains(path, "?") {
		path += sqliteDSNParams
	}

	return DBConn(ctx, path, viper.GetString("DB_DRIVER"))
}

// DBConn creates a database connection using the specified DSN and driver.
func DBConn(ctx context.Context, dsn, driver string) *sqlx.DB {
	dbConn, err := goquestorage.NewDBConn(ctx, &goquestorage.Config{
		DSN:             dsn,
		Driver:          driver,
		MaxIdleConn:     viper.GetInt("DB_MAX_IDLE_CONN"),
		MaxOpenConn:     viper.GetInt("DB_MAX_OPEN_CONN"),
		ConnMaxIdleTime: viper.GetDuration("DB_CONN_MAX_IDLE_CONN"),
		ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFE_CONN"),
	})
	if err != nil {
		panic(err)
	}

	return dbConn
}
