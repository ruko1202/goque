# Goque
[![pipeline](https://github.com/ruko1202/goque/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/ruko1202/goque/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/Coverage-87.8%25-brightgreen)

A robust, database-backed task queue system for Go with built-in worker pools, retry logic, and graceful shutdown support. Supports PostgreSQL, MySQL, and SQLite.

## Features

### Current Features
- ✅ **Multi-database support** - Works with PostgreSQL, MySQL, and SQLite
- ✅ **Reliable persistence** - Task storage with ACID guarantees
- ✅ **Worker pool management** - Configurable concurrent task processing using goroutine pools
- ✅ **Automatic retry logic** - Configurable retry attempts with custom backoff strategies
- ✅ **Task lifecycle management** - Track task status through multiple states (new, processing, done, error, etc.)
- ✅ **Graceful shutdown** - Clean worker shutdown with in-flight task handling
- ✅ **Task timeout handling** - Per-task timeout configuration with context cancellation
- ✅ **Extensible hooks** - Before/after processing hooks for custom logic (metrics, logging, tracing)
- ✅ **Type-safe queries** - PostgreSQL/MySQL use go-jet for type-safe SQL query generation
- ✅ **External ID support** - Associate tasks with external identifiers for idempotency
- ✅ **Built-in task healer** - Automatically marks stuck tasks as errored for reprocessing
- ✅ **Multi-processor support** - Manage multiple task types with a single queue manager
- ✅ **Structured logging** - Built-in structured logging with `log/slog`
- ✅ **Production-ready example** - Complete example service with web dashboard and API
- ✅ **Prometheus metrics** - Built-in Prometheus metrics for monitoring task queue performance

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
    "time"

	"github.com/ruko1202/xlog"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/ruko1202/goque"
    "github.com/ruko1202/goque/pkg/goquestorage"
)

func main() {
	ctx := context.Background()
    // Initialize database connection
    db, err := goquestorage.NewDBConn(ctx, &goquestorage.Config{
        DSN: "postgres://user:pass@localhost:5432/goque?sslmode=disable",
        Driver: goquestorage.PgDriver,
    })
    if err != nil {
        xlog.Panic(ctx, err.Error())
    }

    // Create task storage (works with any supported database)
    taskStorage, err := goque.NewStorage(db)
    if err != nil {
        panic(err)
    }

    // Optional: Configure metrics with service name
    goque.SetMetricsServiceName("my-service")

    // Create Goque manager (includes built-in healer processor)
    goq := goque.NewGoque(taskStorage)

    // Register your task processors
    goq.RegisterProcessor(
        "send_email",
        &EmailProcessor{},
        goque.WithWorkersCount(10),
        goque.WithTaskProcessingMaxAttempts(3),
        goque.WithTaskProcessingTimeout(30 * time.Second),
    )

    // Run all processors
    goq.Run(ctx)

    // Graceful shutdown
    defer goq.Stop()
}
```

### 4. Adding Tasks to the Queue

```go
payload := `{"to": "user@example.com", "subject": "Hello"}`
// Create a new task
task := goque.NewTask("send_email", payload)

// Or with external ID for idempotency
task := goque.NewTaskWithExternalID("send_email", payload, "external-order-123")

// Add to queue using TaskQueueManager (recommended - includes metrics)
taskQueueManager := goque.NewTaskQueueManager(taskStorage)
err := taskQueueManager.AddTaskToQueue(ctx, task)
```

## Example Application

A complete, production-ready example service demonstrating real-world Goque usage is available in the `examples/service` directory.

For detailed instructions and API documentation, see [examples/service/README.md](examples/service/README.md).

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
        ↑ ↓         ↓    ↓
        │ └──error──┘  canceled
        │     ↓
        │  attempts_left
        │
        └──(healer fixes stuck pending tasks)
```

### Status Descriptions

- **new** - Task created and ready to be picked up
- **pending** - Task scheduled for future processing (via NextAttemptAt)
- **processing** - Task currently being processed by a worker
- **done** - Task completed successfully ✓ (terminal)
- **error** - Task failed but has retry attempts remaining
- **attempts_left** - Task failed and exhausted all retry attempts ✗ (terminal)
- **canceled** - Task was manually canceled ✗ (terminal)

### Valid State Transitions

| Current Status | Next Status | Trigger |
|----------------|-------------|---------|
| `new` | `pending` | Task scheduled for processing |
| `pending` | `processing` | Worker picks up task |
| `pending` | `error` | Healer marks stuck task (cure operation) |
| `processing` | `done` | Successful processing |
| `processing` | `error` | Failed processing with retries left |
| `processing` | `canceled` | Manual cancellation |
| `error` | `pending` | Retry logic schedules next attempt |
| `error` | `attempts_left` | No more retry attempts available |

### Terminal States

Tasks in these states will not be processed again:
- **done** - Successfully completed
- **canceled** - Manually canceled by user
- **attempts_left** - Failed with no remaining retry attempts

## Built-in Features

### Task Healer

Goque includes a built-in healer processor that automatically monitors and fixes stuck tasks. Tasks that remain in the "pending" status for too long are automatically marked as errored, allowing them to be retried. The healer is automatically registered when you use `internal.NewGoque()`.

You can configure the healer behavior:

```go
goq.RegisterProcessor(
    "send_email",
    &EmailProcessor{},
    goque.WithWorkersCount(10),
    goque.WithTaskProcessingMaxAttempts(3),
    goque.WithTaskProcessingTimeout(30 * time.Second),

    goque.WithHealerPeriod(10*time.Minute),
    goque.WithHealerUpdatedAtTimeAgo(time.Hour),
    goque.WithHealerTimeout(30*time.Second),
)
```

### Prometheus Metrics

Goque includes built-in Prometheus metrics for comprehensive monitoring of your task queue. Metrics are automatically collected during task processing operations.

#### Available Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `goque_processed_tasks_total` | Counter | `task_type`, `status` | Total number of processed tasks by type and final status |
| `goque_processed_tasks_with_error_total` | Counter | `task_type`, `task_processing_operations`, `task_error_type` | Tasks processed with errors, including error type details |
| `goque_task_processing_duration_seconds` | Histogram | `task_type` | Task processing duration distribution in seconds |
| `goque_task_payload_size_bytes` | Histogram | `task_type` | Task payload size distribution in bytes |

#### Configuration

```go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/ruko1202/goque"
    "net/http"
)

// Optional: Set service name for metrics labels
goque.SetMetricsServiceName("my-service")

// Expose metrics endpoint
http.Handle("/metrics", promhttp.Handler())
go http.ListenAndServe(":9090", nil)
```

#### Task Processing Operations

Metrics track errors across different operations:
- `add_to_queue` - Errors during task creation and queue insertion
- `processing` - Errors during task execution
- `cleanup` - Errors during task cleanup operations
- `health` - Errors during healer operations

#### Example Queries

```promql
# Task processing rate by type
rate(goque_processed_tasks_total[5m])

# Task error rate
rate(goque_processed_tasks_with_error_total[5m])

# Average processing duration
rate(goque_task_processing_duration_seconds_sum[5m])
  / rate(goque_task_processing_duration_seconds_count[5m])

# 95th percentile processing time
histogram_quantile(0.95, goque_task_processing_duration_seconds_bucket)

# Tasks by status
sum by (status) (goque_processed_tasks_total)
```

For a complete example with metrics integration, see [examples/service/](examples/service/).

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
├── goque_manager.go            # Public API - TaskQueueManager interface
├── internal/
│   ├── goque.go                # Main Goque implementation
│   ├── entity/                 # Domain entities and errors
│   │   ├── task.go             # Task entity
│   │   ├── errors.go           # Domain errors
│   │   └── operations.go       # Task operation constants
│   ├── queue_manager/          # Task queue manager implementation
│   ├── metrics/                # Prometheus metrics
│   │   ├── metrics.go          # Metrics collectors and functions
│   │   └── vars.go             # Metrics configuration
│   ├── processors/             # Task processing components
│   │   ├── queueprocessor/     # Main queue processor
│   │   └── internalprocessors/ # Built-in processors (healer, cleaner)
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
├── examples/service/           # Production-ready example service
├── DATABASE.md                 # Complete database setup guide
└── test/                       # Test utilities and fixtures
```

## License

[Add your license here]

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.