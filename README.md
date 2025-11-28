# Goque
[![pipeline](https://github.com/ruko1202/goque/actions/workflows/ci.yml/badge.svg)](https://github.com/ruko1202/goque/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/Coverage-79.8%25-brightgreen)

A robust, database-backed task queue system for Go with built-in worker pools, retry logic, and graceful shutdown support. Supports PostgreSQL, MySQL, and SQLite.

## Features

- **Multi-database support** - Works with PostgreSQL, MySQL, and SQLite
- **Reliable persistence** - Task storage with ACID guarantees
- **Worker pool management** - Configurable concurrent task processing using goroutine pools
- **Automatic retry logic** - Configurable retry attempts with custom backoff strategies
- **Task lifecycle management** - Track task status through multiple states (new, processing, done, error, etc.)
- **Graceful shutdown** - Clean worker shutdown with in-flight task handling
- **Task timeout handling** - Per-task timeout configuration with context cancellation
- **Extensible hooks** - Before/after processing hooks for custom logic
- **Type-safe queries** - PostgreSQL uses go-jet for type-safe SQL query generation
- **External ID support** - Associate tasks with external identifiers for idempotency
- **Built-in task healer** - Automatically marks stuck tasks as errored for reprocessing
- **Multi-processor support** - Manage multiple task types with a single queue manager

## Installation

```bash
go get github.com/ruko1202/goque
```

## Quick Start

### 1. Prepare database

Goque supports three database backends: **PostgreSQL**, **MySQL**, and **SQLite**.

```bash
# Configure your database connection in .env.local
# For PostgreSQL:
echo 'DB_DRIVER postgres' > .env.local
echo 'DB_DSN postgres://postgres:postgres@localhost:5432/goque?sslmode=disable' >> .env.local

# For MySQL:
# echo 'DB_DRIVER mysql' > .env.local
# echo 'DB_DSN root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC' >> .env.local

# For SQLite:
# echo 'DB_DRIVER sqlite3' > .env.local
# echo 'DB_DSN ./goque.db' >> .env.local

# Install database tools
make bin-deps-db

# Run migrations (works with any database)
make db-up
```

For detailed database setup instructions, see [DATABASE.md](DATABASE.md).

### 2. Create a Task Processor

Implement the `TaskProcessor` interface to define how your tasks should be processed:

```go
type EmailProcessor struct{}

func (p *EmailProcessor) ProcessTask(ctx context.Context, task *entity.Task) error {
    // Your task processing logic here
    return sendEmail(task.Payload)
}
```

### 3. Initialize and Run the Queue Manager (Recommended)

```go
package main

import (
    "context"
    "database/sql"
    "time"

    "github.com/ruko1202/goque/internal"
    "github.com/ruko1202/goque/internal/processor"
    "github.com/ruko1202/goque/internal/storages/task"
)

func main() {
    // Initialize database connection
    // For PostgreSQL: sql.Open("postgres", dsn)
    // For MySQL: sql.Open("mysql", dsn)
    // For SQLite: sql.Open("sqlite3", dsn)
    db, err := sql.Open("postgres", "your-connection-string")
    if err != nil {
        panic(err)
    }

    // Create task storage (works with any supported database)
    taskStorage := task.NewStorage(db)

    // Create Goque manager (includes built-in healer processor)
    goque := internal.NewGoque(taskStorage)

    // Register your task processors
    goque.RegisterProcessor(
        "send_email",
        &EmailProcessor{},
        processor.WithWorkers(10),
        processor.WithMaxAttempts(3),
        processor.WithTaskTimeout(30 * time.Second),
    )

    // Run all processors
    ctx := context.Background()
    goque.Run(ctx)

    // Graceful shutdown
    defer goque.Stop()
}
```

### 4. Adding Tasks to the Queue

```go
import "github.com/ruko1202/goque/internal/entity"

// Create a new task
task := entity.NewTask("send_email", `{"to": "user@example.com", "subject": "Hello"}`)

// Or with external ID for idempotency
task := entity.NewTaskWithExternalID("send_email", payload, "order-123")

// Add to storage
err := taskStorage.AddTask(ctx, task)
```

## Configuration Options

The `GoqueProcessor` supports various configuration options:

- `WithWorkers(n int)` - Set the number of concurrent workers (default: 1)
- `WithMaxAttempts(n int32)` - Set maximum retry attempts (default: 3)
- `WithTaskTimeout(d time.Duration)` - Set per-task timeout (default: 30s)
- `WithFetchMaxTasks(n int64)` - Set maximum tasks to fetch per cycle (default: 10)
- `WithFetchTick(d time.Duration)` - Set fetch interval (default: 1s)
- `WithNextAttemptAtFunc(f NextAttemptAtFunc)` - Custom retry backoff strategy
- `WithHooksBeforeProcessing(hooks ...HookBeforeProcessing)` - Add pre-processing hooks
- `WithHooksAfterProcessing(hooks ...HookAfterProcessing)` - Add post-processing hooks

## Task Status Lifecycle

Tasks flow through the following states:

```
new → pending → processing → done
                 ↓       ↓
              canceled  error → attempts_left (no more retries)
                          ↓
                        (retry) → processing
  
```

- **new** - Task created and ready to be picked up
- **pending** - Task scheduled for future processing
- **processing** - Task currently being processed
- **done** - Task completed successfully
- **error** - Task failed but has retry attempts remaining
- **attempts_left** - Task failed and exhausted all retry attempts
- **canceled** - Task was manually canceled

## Built-in Features

### Task Healer

Goque includes a built-in healer processor that automatically monitors and fixes stuck tasks. Tasks that remain in the "pending" status for too long are automatically marked as errored, allowing them to be retried. The healer is automatically registered when you use `internal.NewGoque()`.

You can configure the healer behavior:

```go
import internalprocessors "github.com/ruko1202/goque/internal/internal_processors"

goque := internal.NewGoque(
    taskStorage,
    internalprocessors.WithHealerUpdatedAtTimeAgo(5 * time.Minute),
    internalprocessors.WithHealerMaxTasks(200),
)
```

## Development

### Prerequisites

- Go 1.23 or higher
- One of the supported databases:
  - PostgreSQL 12+
  - MySQL 8+
  - SQLite 3+
- Make

### Setup

```bash
# Install all dependencies and tools
make all

# Install only binary dependencies
make bin-deps

# Download Go modules
make deps
```

### Running Tests

```bash
# Run all tests
make tloc

# Run tests with coverage
make test-cov
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt
```

### Database Operations

All database commands work with any supported database (PostgreSQL, MySQL, SQLite).
Simply configure `DB_DRIVER` in `.env.local` and use universal commands:

```bash
# Show current database configuration
make db-info

# Create a new migration for current DB_DRIVER
make db-migrate-create name="your_migration_name"

# Check migration status
make db-status

# Apply migrations
make db-up

# Rollback last migration
make db-down

# Regenerate database models (PostgreSQL only)
make db-models

# Docker Compose commands for local development
make docker-up           # Start PostgreSQL and MySQL with Docker Compose
make docker-pg-up        # Start only PostgreSQL
make docker-mysql-up     # Start only MySQL
make docker-down         # Stop and remove all containers
make docker-down-volumes # Stop and remove containers with volumes
make docker-logs         # Show logs from all containers
make docker-ps           # Show status of containers
```

For complete database setup guide, migrations structure, and Docker Compose configuration, see [DATABASE.md](DATABASE.md).

### Generate Mocks

```bash
make mocks
```

## Project Structure

```
.
├── internal/
│   ├── goque.go                # Main queue manager
│   ├── entity/                 # Domain entities (Task, etc.)
│   ├── processor/              # Queue processor and task processor interfaces
│   ├── internal_processors/    # Built-in processors (healer, etc.)
│   ├── storages/               # Data access layer (multi-database support)
│   │   ├── pg/task/            # PostgreSQL storage (go-jet)
│   │   ├── mysql/task/         # MySQL storage (raw SQL)
│   │   ├── sqlite/task/        # SQLite storage (raw SQL)
│   │   └── dbutils/            # Database utilities
│   └── pkg/
│       └── generated/          # Generated code (models, mocks)
├── migrations/                 # Database migrations
│   ├── pg/                     # PostgreSQL migrations
│   ├── mysql/                  # MySQL migrations
│   └── sqlite/                 # SQLite migrations
├── DATABASE.md                 # Complete database setup guide
└── test/                       # Test utilities and fixtures
```

## License

[Add your license here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.