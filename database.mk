# =====================================
# Database Commands
# =====================================
# Universal database commands for PostgreSQL, MySQL, and SQLite
#
# This file provides database-agnostic commands that automatically work with
# the database configured in DB_DRIVER environment variable.
#
# Configuration:
#   Set DB_DRIVER and DB_DSN in .env.local (see DATABASE.md for details)
#
# Common commands:
#   make db-info           - Show current database configuration
#   make db-up             - Apply all pending migrations
#   make db-down           - Rollback last migration
#   make db-status         - Show migration status
#   make db-migrate-create - Create new migration file
#   make db-models         - Generate models (PostgreSQL only)
#
# Docker commands:
#   make docker-up         - Start all databases with Docker Compose
#   make docker-down       - Stop all database containers
#   make docker-pg-up      - Start only PostgreSQL
#   make docker-mysql-up   - Start only MySQL
#   make docker-logs       - Show logs from all containers
#   make docker-ps         - Show container status
# =====================================

# Include shared configuration (GOBIN, ENV_CONFIG_FILE, ROOT_DIR)
include scratch.mk

# -------------------------------------
# Database Configuration
# -------------------------------------
# Migration directories for each database type
# These paths are relative to ROOT_DIR (project root)
MIGRATIONS_DIR_PG   	?= migrations/pg
MIGRATIONS_DIR_MYSQL	?= migrations/mysql
MIGRATIONS_DIR_SQLITE	?= migrations/sqlite

# -------------------------------------
# Database Driver Detection
# -------------------------------------
# Determine DB_DRIVER from .env.local if not already set
# Default: postgres
# Supported values: postgres, mysql, sqlite3, sqlite
ifndef DB_DRIVER
DB_DRIVER=$(shell [ -f $(ENV_CONFIG_FILE) ] && cat $(ENV_CONFIG_FILE) | grep ^DB_DRIVER | awk '{print $$2}' | sed 's/"//g' || echo "postgres")
endif

# -------------------------------------
# Database DSN (Data Source Name)
# -------------------------------------
# Read DB_DSN from .env.local if not already set
# DSN format varies by database:
#   PostgreSQL: postgres://user:pass@host:port/dbname?sslmode=disable
#   MySQL: user:pass@tcp(host:port)/dbname?parseTime=true&loc=UTC
#   SQLite: ./path/to/database.db
ifndef DB_DSN
DB_DSN=$(shell [ -f $(ENV_CONFIG_FILE) ] && cat $(ENV_CONFIG_FILE) | grep ^DB_DSN | awk '{print $$2}' | sed 's/"//g' || echo "")
endif

# If DB_DSN is empty, show warning and use sensible defaults
# This allows quick start without configuration file
ifeq ($(DB_DSN),)
    ifeq ($(DB_DRIVER),mysql)
        DB_DSN := 'root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC'
    else ifeq ($(DB_DRIVER),sqlite3)
        DB_DSN := './goque.sqlite.db'
    else ifeq ($(DB_DRIVER),sqlite)
        DB_DSN := './goque.sqlite.db'
    else
        # Default to PostgreSQL with standard test database
        DB_DSN := 'postgres://postgres:postgres@localhost:5432/goque?sslmode=disable'
    endif

    $(warning DB_DSN not found in $(ENV_CONFIG_FILE), using defaults based on DB_DRIVER=$(DB_DRIVER): $(DB_DSN))
endif

# -------------------------------------
# Driver-Specific Configuration
# -------------------------------------
# Set migrations directory and goose driver based on DB_DRIVER
# This allows the same commands to work with any database
ifeq ($(DB_DRIVER),mysql)
    MIGRATIONS_DIR := $(MIGRATIONS_DIR_MYSQL)
    GOOSE_DRIVER := mysql
else ifeq ($(DB_DRIVER),sqlite3)
    MIGRATIONS_DIR := $(MIGRATIONS_DIR_SQLITE)
    GOOSE_DRIVER := sqlite3
else ifeq ($(DB_DRIVER),sqlite)
    MIGRATIONS_DIR := $(MIGRATIONS_DIR_SQLITE)
    GOOSE_DRIVER := sqlite3
else
    # Default to PostgreSQL
    MIGRATIONS_DIR := $(MIGRATIONS_DIR_PG)
    GOOSE_DRIVER := postgres
endif

# -------------------------------------
# Database - Universal Commands (use DB_DRIVER)
# -------------------------------------
# These commands automatically work with any configured database.
# The actual database used is determined by DB_DRIVER variable.

# db-migrate-create - Create a new migration file
# Usage: make db-migrate-create name="add_users_table"
# Creates timestamped migration files in the appropriate migrations/ directory
# Format: YYYYMMDDHHMMSS_<name>.sql
.PHONY: db-migrate-create
db-migrate-create: name=
db-migrate-create: ## Create new migration for current DB_DRIVER ($(DB_DRIVER))
	$(info $(M) creating $(DB_DRIVER) migration...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_DSN)" create "${name}" sql

# db-status - Show migration status
# Displays which migrations have been applied and which are pending
# Shows version number and migration name for each migration
.PHONY: db-status
db-status: ## Check migrations status for current DB_DRIVER ($(DB_DRIVER))
	$(info $(M) check $(DB_DRIVER) migrations status...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_DSN)" status

# db-up - Apply all pending migrations
# Runs all unapplied migrations in sequential order
# For PostgreSQL: also regenerates type-safe models after migrations
# Safe to run multiple times (idempotent)
.PHONY: db-up
db-up: ## Apply migrations for current DB_DRIVER ($(DB_DRIVER))
	$(info $(M) starting $(DB_DRIVER) migration up...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_DSN)" up
	$(MAKE) db-status

# db-down - Rollback the last migration
# Reverts the most recently applied migration
# Useful for undoing mistakes or testing rollback logic
# WARNING: This modifies the database schema
.PHONY: db-down
db-down: ## Rollback migrations for current DB_DRIVER ($(DB_DRIVER))
	$(info $(M) starting $(DB_DRIVER) migration down...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(GOOSE_DRIVER) "$(DB_DSN)" down
	$(MAKE) db-status

# db-models - Generate type-safe database models (PostgreSQL only)
# Uses go-jet to generate Go structs and query builders from database schema
# Only works with PostgreSQL; MySQL and SQLite use raw SQL queries
# Generated files are placed in internal/pkg/generated/
.PHONY: db-models
db-models: ## Generate models
	$(info $(M) generating $(DB_DRIVER) models...)
	@go run ./scripts/dbmodels/generate.go --driver=$(DB_DRIVER) --dsn=$(DB_DSN) --dest="internal/pkg/generated/"

.PHONY: all-db-models
all-db-models: ## Generate models for all DB types
	@ENV_CONFIG_FILE=.env.pg.local make db-models
	@echo 
	@ENV_CONFIG_FILE=.env.mysql.local make db-models
	@echo
	@ENV_CONFIG_FILE=.env.sqlite.local make db-models

# db-info - Display current database configuration
# Shows all relevant database settings:
#   - DB_DRIVER: Database type (postgres/mysql/sqlite3)
#   - GOOSE_DRIVER: Goose-specific driver name
#   - MIGRATIONS_DIR: Directory containing migration files
#   - DB_DSN: Connection string (with credentials visible)
# Useful for debugging configuration issues
.PHONY: db-info
db-info: ## Show current database configuration
	@echo "Current database configuration:"
	@echo "  DB_DRIVER: $(DB_DRIVER)"
	@echo "  GOOSE_DRIVER: $(GOOSE_DRIVER)"
	@echo "  MIGRATIONS_DIR: $(MIGRATIONS_DIR)"
	@echo "  DB_DSN: $(DB_DSN)"

# -------------------------------------
# Docker Compose Commands
# -------------------------------------
# Convenient commands for managing database containers via Docker Compose
# Uses docker-compose.yml in project root
#
# Services defined:
#   - postgres: PostgreSQL 17 on port 5432
#   - mysql: MySQL 8 on port 3306
#
# All services include:
#   - Health checks for readiness detection
#   - Persistent volumes for data storage
#   - Standard test credentials (see docker-compose.yml)

# docker-up - Start all database services
# Starts both PostgreSQL and MySQL in background (detached mode)
# Creates containers, networks, and volumes if they don't exist
# Safe to run multiple times (idempotent)
.PHONY: docker-up
docker-up: ## Start all databases with Docker Compose
	$(info $(M) starting databases with Docker Compose...)
	docker compose up -d
	make docker-ps
	@ENV_CONFIG_FILE=.env.sqlite.local make db-up
	make all-db-models

# docker-down - Stop and remove all database containers
# Stops containers, removes them, and deletes associated volumes
# WARNING: This will delete all data stored in the databases
# Use this when you want a fresh start with empty databases
.PHONY: docker-down
docker-down: ## Stop and remove all database containers with volumes
	$(info $(M) stopping all database containers and removing volumes...)
	docker compose down -v
	make docker-ps

# docker-pg-up - Start only PostgreSQL service
# Useful when you only need PostgreSQL for testing/development
# Starts container in background with health checks enabled
.PHONY: docker-pg-up
docker-pg-up: ## Start only PostgreSQL with Docker Compose
	$(info $(M) starting PostgreSQL with Docker Compose...)
	docker compose up -d postgres apply-postgres-migrations
	make docker-ps

# docker-mysql-up - Start only MySQL service
# Useful when you only need MySQL for testing/development
# Starts container in background with health checks enabled
.PHONY: docker-mysql-up
docker-mysql-up: ## Start only MySQL with Docker Compose
	$(info $(M) starting MySQL with Docker Compose...)
	docker compose up -d mysql apply-mysql-migrations
	make docker-ps

# docker-logs - Stream logs from all database containers
# Shows continuous log output from both PostgreSQL and MySQL
# Press Ctrl+C to stop following logs
# Useful for debugging connection issues or monitoring queries
.PHONY: docker-logs
docker-logs: ## Show logs from all database containers
	docker compose logs -f

# docker-ps - Show status of database containers
# Displays:
#   - Container name and status (running/stopped)
#   - Ports mapping
#   - Health check status
# Useful for verifying that databases are ready to accept connections
.PHONY: docker-ps
docker-ps: ## Show status of database containers
	docker compose ps -a
