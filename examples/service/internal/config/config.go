package config

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Queue         QueueConfig
	TaskGenerator TaskGeneratorConfig
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database connection configuration.
type DatabaseConfig struct {
	Driver string
	DSN    string
}

// QueueConfig holds queue processing configuration.
type QueueConfig struct {
	Workers           int
	MaxAttempts       int
	TaskTimeout       time.Duration
	HealerInterval    time.Duration
	CleanerInterval   time.Duration
	RetentionPeriod   time.Duration
	RetryBaseInterval time.Duration
}

// TaskGeneratorConfig holds task generator configuration.
type TaskGeneratorConfig struct {
	Enabled  bool
	Interval time.Duration
	MinTasks int
	MaxTasks int
}

// Load loads configuration from environment variables and config file.
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Environment variables
	v.SetEnvPrefix("GOQUE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		// Config file is optional
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)

	// Database defaults
	v.SetDefault("database.driver", "postgres")
	v.SetDefault("database.dsn", "postgres://postgres:postgres@localhost:5432/goque_example?sslmode=disable")

	// Queue defaults
	v.SetDefault("queue.workers", 5)
	v.SetDefault("queue.maxattempts", 3)
	v.SetDefault("queue.tasktimeout", 5*time.Minute)
	v.SetDefault("queue.healerinterval", 1*time.Minute)
	v.SetDefault("queue.cleanerinterval", 10*time.Minute)
	v.SetDefault("queue.retentionperiod", 24*time.Hour)
	v.SetDefault("queue.retrybaseinterval", 30*time.Second)

	// Task generator defaults
	v.SetDefault("taskgenerator.enabled", true)
	v.SetDefault("taskgenerator.interval", 10*time.Second)
	v.SetDefault("taskgenerator.mintasks", 1)
	v.SetDefault("taskgenerator.maxtasks", 5)
}
