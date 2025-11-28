package testutils

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"

	"github.com/ruko1202/goque/pkg/goquestorage"
)

// PgDBConn creates a PostgreSQL database connection for testing.
func PgDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	viper.SetDefault("DB_DRIVER", goquestorage.PgDriver)

	return DBlConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// MysqlDBConn creates a MySQL database connection for testing.
func MysqlDBConn(ctx context.Context) *sqlx.DB {
	viper.SetDefault("DB_DSN", "root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC")
	viper.SetDefault("DB_DRIVER", goquestorage.MysqlDriver)

	return DBlConn(ctx, viper.GetString("DB_DSN"), viper.GetString("DB_DRIVER"))
}

// DBlConn creates a database connection using the specified DSN and driver.
func DBlConn(ctx context.Context, dsn, driver string) *sqlx.DB {
	viper.SetDefault("DB_DSN", dsn)
	viper.SetDefault("DB_DRIVER", driver)
	viper.SetDefault("DB_MAX_OPEN_CONN", 10)
	viper.SetDefault("DB_MAX_IDLE_CONN", 10)

	dbConn, err := goquestorage.NewDBConn(ctx, &goquestorage.Config{
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
