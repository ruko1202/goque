# Goque
[![pipeline](https://github.com/ruko1202/goque/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/ruko1202/goque/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/Coverage-87.0%25-brightgreen)

A robust, database-backed task queue system for Go with built-in worker pools, retry logic, and graceful shutdown support.
Supports PostgreSQL, MySQL, and SQLite.

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
- ✅ **Periodic jobs** - Schedule recurring task creation with cron expressions or custom schedulers
- ✅ **Structured logging** - Built-in structured logging with xlog (supports zap, slog, and custom adapters)
- ✅ **Production-ready example** - Complete example service with web dashboard and API
- ✅ **Prometheus metrics** - Built-in Prometheus metrics for monitoring task queue performance

## Installation

```bash
go get github.com/ruko1202/goque
```

## Database Support

Goque supports three database backends with different performance characteristics:

| Database | Latency | Memory/op | Best For | Production Ready |
|----------|---------|-----------|----------|------------------|
| **PostgreSQL** | **0.97 ms** | 35.4 KB | High-throughput production systems | ✅ **Recommended** |
| **MySQL** | 1.59 ms | **33.4 KB** | Memory-constrained environments | ✅ Yes |
| **SQLite** | 3.24 ms | 38.9 KB | Local development, testing, CI/CD | ⚠️ **Dev/Test only** |

**Key findings:**
- PostgreSQL is **3.4x faster** than SQLite and **39% faster** than MySQL
- MySQL uses **6% less memory** than PostgreSQL
- SQLite has file-level locking and is **not recommended for production**

#### SQLite caveats

SQLite serializes writes at the file level. Two practical consequences
for goque on SQLite:

- **Concurrent writers deadlock under the default `BEGIN DEFERRED`
  transaction mode.** Operations like `CureTasks` and
  `GetTasksForProcessing` open their own transactions via
  `dbtx.WithinTx`; when several workers call them in parallel, one
  SELECT takes a SHARED lock and the next UPDATE can't upgrade to
  RESERVED — SQLite returns `SQLITE_BUSY` immediately and the
  `_busy_timeout` parameter does **not** apply to lock upgrades.
  Mitigation: pass `?_txlock=immediate&_journal_mode=WAL&_busy_timeout=5000`
  in your DSN so all transactions start as `BEGIN IMMEDIATE` and
  multiple readers can coexist with the single writer.
- **One process at a time.** Even with the DSN tuning above, a single
  SQLite database file should be opened by a single goque process.
  Spinning up two binaries that both write to `goque.sqlite.db` will
  see `database is locked` errors no matter what.

If you need real horizontal scaling, run PostgreSQL or MySQL — they
have row-level locks and don't have this class of problem.

### Schema

Goque installs a single table named **`goque_task`** plus three indexes
(`goque_task_type_external_id_idx`, `goque_task_type_status_next_attempt_at_idx`,
`goque_task_type_status_updated_at_idx`). The full DDL lives in
[migrations/](migrations/) — apply it with `make db-up`.

> **Breaking change in this release**: the table was previously named `task`.
> Existing deployments must rename in place before adopting:
>
> ```sql
> ALTER TABLE task RENAME TO goque_task;
> ALTER INDEX task_type_external_id_idx          RENAME TO goque_task_type_external_id_idx;
> ALTER INDEX task_type_status_next_attempt_at_idx RENAME TO goque_task_type_status_next_attempt_at_idx;
> ALTER INDEX task_type_status_updated_at_idx      RENAME TO goque_task_type_status_updated_at_idx;
> ```

## Quick Start

### 1. Prepare database

Choose the database backend that fits your deployment requirements.

#### Production

For production deployments, apply migrations from the `migrations/` directory to your database:

```bash
# Install goose migration tool
go install github.com/pressly/goose/v3/cmd/goose@latest

# Apply PostgreSQL migrations
goose -dir migrations/pg postgres "your-production-dsn" up

# Or for MySQL
goose -dir migrations/mysql mysql "your-production-dsn" up
```

#### Local Development

For local development, use Docker Compose and Make commands:

```bash
# Start PostgreSQL and MySQL with Docker Compose
make docker-up

# Configure your database connection in .env.local
# For PostgreSQL (recommended):
echo 'DB_DRIVER=postgres' > .env.local
echo 'DB_DSN=postgres://postgres:postgres@localhost:5432/goque?sslmode=disable' >> .env.local

# For MySQL:
# echo 'DB_DRIVER=mysql' > .env.local
# echo 'DB_DSN=root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC' >> .env.local

# For SQLite (dev/test only):
# echo 'DB_DRIVER=sqlite3' > .env.local
# echo 'DB_DSN=./goque.db' >> .env.local

# Install database tools (goose)
make bin-deps-db

# Apply migrations
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

Use a typed processor when you want Goque to decode JSON payloads before your processing logic runs:

```go
type EmailPayload struct {
    To      string `json:"to"`
    Subject string `json:"subject"`
}

processor := goque.NewTypedTaskProcessor(
    goque.TypedTaskProcessorFunc[EmailPayload](func(ctx context.Context, task *goque.TypedTask[EmailPayload]) error {
        return sendEmail(task.Payload.To, task.Payload.Subject)
    }),
    goque.WithPayloadDecodeErrorCancel(),
)
```

If payload decoding fails, the typed processor is not called. The decode error is returned through the normal task processing flow, recorded in `goque_payload_decode_errors_total`, written to `task.Errors`, and passed to `WithHooksAfterProcessing`. Add `WithPayloadDecodeErrorCancel` to the typed processor to cancel decode failures instead of retrying them:

```go
goq.RegisterProcessor(
    "send_email",
    processor,
    goque.WithHooksAfterProcessing(func(ctx context.Context, task *goque.Task, err error) {
        if errors.Is(err, goque.ErrPayloadUnmarshal) {
            // alert or inspect invalid payload
        }
    }),
)
```
```

### 3. Initialize and Run the Queue Manager (Recommended)

```go
package main

import (
    "context"
    "time"

    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    "github.com/ruko1202/goque"
    "github.com/ruko1202/goque/pkg/goquestorage"
    "github.com/ruko1202/xlog"
    "go.uber.org/zap"
)

func main() {
    ctx := context.Background()

    // Configure structured logging with xlog
    logger := xlog.NewZapAdapter(zap.Must(zap.NewProduction()))
    xlog.ReplaceGlobalLogger(logger)
    ctx = xlog.ContextWithLogger(ctx, logger)

    // Optional: Configure OpenTelemetry tracing
    // goque.SetTracerProvider(tracerProvider) // See "Observability" section

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

// Or marshal a typed payload as JSON
task, err := goque.NewTaskWithPayload("send_email", EmailPayload{
    To:      "user@example.com",
    Subject: "Hello",
})

// Add to queue using TaskQueueManager (recommended - includes metrics)
taskQueueManager := goque.NewTaskQueueManager(taskStorage)
err := taskQueueManager.AddTaskToQueue(ctx, task)
```

### 5. Transactional Outbox

Enqueue a task atomically with your own domain writes by passing a
`*sqlx.Tx` through the context. If the tx rolls back, the enqueue is
discarded with it; if it commits, the task becomes visible to workers
in the same instant the domain rows do.

```go
tx, err := db.BeginTxx(ctx, nil)
if err != nil {
    return err
}
defer func() { _ = tx.Rollback() }() // no-op after a successful Commit

// 1) Your domain write inside the tx
if _, err := tx.ExecContext(ctx, "INSERT INTO orders ..."); err != nil {
    return err
}

// 2) Enqueue using the same tx — goque.WithTx wires it into ctx
task := goque.NewTask("send_order_confirmation", payload)
if err := taskQueueManager.AddTaskToQueue(goque.WithTx(ctx, tx), task); err != nil {
    return err
}

// 3) Commit — domain row and queued task become durable together
return tx.Commit()
```

Calls without `goque.WithTx` keep the existing behavior and write
directly to the storage's `*sqlx.DB`.

> **Scope of `WithTx`**
>
> `AddTaskToQueue` is the method that matters for the outbox pattern, and
> it participates in your tx on **every** backend. Other tx-aware methods:
> `GetTask`, `GetTasks`, `UpdateTask`, `CancelTask`, and (PostgreSQL only)
> `DeleteTasks`/`CureTasks` — they also honor an attached tx.
>
> `GetTasksForProcessing` and `ResetAttempts` always run in their own
> internally-managed tx (the worker fetch uses `FOR UPDATE SKIP LOCKED`
> and must not be entangled with a caller's tx). On MySQL/SQLite,
> `DeleteTasks` and `CureTasks` are in this group too because they do
> batched read+write internally.
>
> **`AsyncAddTaskToQueue` is not outbox-safe.** Its goroutine outlives
> your `Commit`/`Rollback`. If a tx is detected on the async path it is
> stripped automatically and logged at WARN — but the enqueue then runs
> against `*sqlx.DB` and is no longer atomic with your domain write.
> Always use the synchronous `AddTaskToQueue` for outbox.

## Example Application

A complete, production-ready example service demonstrating real-world Goque usage is available in the `examples/service` directory.

For detailed instructions and API documentation, see [examples/service/README.md](examples/service/README.md).

## Configuration Options

The `GoqueProcessor` supports various configuration options:

- `WithWorkersCount(n int)` - Set the number of concurrent workers (default: 1)
- `WithWorkersPanicHandler(handler func(context.Context) func(any))` - Set a custom worker panic handler
- `WithTaskProcessingMaxAttempts(n int32)` - Set maximum retry attempts (default: 3)
- `WithTaskProcessingTimeout(d time.Duration)` - Set per-task timeout (default: 30s)
- `WithTaskProcessingNextAttemptAtFunc(f)` - Custom retry backoff strategy
- `WithTaskFetcherMaxTasks(n int64)` - Set maximum tasks to fetch per cycle (default: 10)
- `WithTaskFetcherTick(d time.Duration)` - Set fetch interval (default: 1s)
- `WithTaskFetcherTimeout(d time.Duration)` - Set timeout for fetching tasks from storage
- `WithHooksBeforeProcessing(hooks ...HookBeforeProcessing)` - Add pre-processing hooks
- `WithHooksAfterProcessing(hooks ...HookAfterProcessing)` - Add post-processing hooks
- `WithCleanerPeriod(d time.Duration)` - Set the cleaner run interval
- `WithCleanerUpdatedAtTimeAgo(d time.Duration)` - Set the completed-task age threshold for cleanup
- `WithCleanerTimeout(d time.Duration)` - Set the cleaner operation timeout
- `WithHealerPeriod(d time.Duration)` - Set the healer run interval
- `WithHealerUpdatedAtTimeAgo(d time.Duration)` - Set the stuck-task age threshold for healing
- `WithHealerTimeout(d time.Duration)` - Set the healer operation timeout

### Periodic Jobs

Periodic jobs run in separate scheduler processors. On each schedule tick, Goque calls the job factory and inserts the returned task into the queue through `TaskQueueManager`. The inserted task is a normal one-shot task: successful tasks still become `done`, failed tasks still use the normal retry flow, and cancellation is handled by the queue manager.

Use `NewCronJob` for a standard 5-field cron expression:

```go
periodicJob, err := goque.NewCronJob(
    "daily_report_schedule",
    "0 3 * * *",
    time.UTC,
    func(ctx context.Context) (*goque.Task, error) {
        return goque.NewTask(
            "daily_report",
            `{"report":"sales"}`,
        ), nil
    },
    goque.WithPeriodicJobRunOnStart(),
)
if err != nil {
    return err
}

goq := goque.NewGoque(taskStorage)
goq.RegisterPeriodicJob(periodicJob)
```

`WithPeriodicJobRunOnStart()` is optional. When enabled, the periodic job enqueues one task immediately when Goque starts, then continues with its normal schedule.

Register a normal processor for the task type produced by the periodic factory:

```go
goq.RegisterProcessor(
    "daily_report",
    &DailyReportProcessor{},
)
```

For non-cron schedules, use `NewPeriodicJob` with `PeriodicSchedulerFunc` or any type that implements `PeriodicSchedule`:

```go
periodicJob, err := goque.NewPeriodicJob(
    "quarter_hour_report_schedule",
    goque.PeriodicSchedulerFunc(func(t time.Time) time.Time {
        return t.Add(15 * time.Minute)
    }),
    func(ctx context.Context) (*goque.Task, error) {
        return goque.NewTask(
            "quarter_hour_report",
            `{"report":"sales"}`,
        ), nil
    },
)
if err != nil {
    return err
}
```

Use `external_id` when the periodic job must be idempotent for a schedule slot:

```go
slot := time.Now().UTC().Truncate(15 * time.Minute)
externalID := fmt.Sprintf("quarter_hour_report:%s", slot.Format(time.RFC3339))
task := goque.NewTaskWithExternalID("quarter_hour_report", payload, externalID)
```

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

Goque includes a built-in healer processor that automatically monitors and fixes stuck tasks. Tasks that remain in the "pending" status for too long are automatically marked as errored, allowing them to be retried. The healer is automatically registered when you call `goque.NewGoque()`.

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

### Observability

#### Prometheus Metrics

Goque includes built-in Prometheus metrics for comprehensive monitoring of your task queue. Metrics are automatically collected during task processing operations.

> 📊 **Ready-to-import Grafana dashboard** lives in [`grafana/`](grafana/) — a Prometheus metrics dashboard with import instructions. See [`grafana/README.md`](grafana/README.md).

##### Available Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `goque_processed_tasks_total` | Counter | `task_type`, `status` | Total number of processed tasks by type and final status |
| `goque_processed_tasks_with_error_total` | Counter | `task_type`, `task_processing_operations`, `task_error_type` | Tasks processed with errors, including error type details |
| `goque_task_processing_duration_seconds` | Histogram | `task_type` | Task processing duration distribution in seconds |
| `goque_task_payload_size_bytes` | Histogram | `task_type` | Task payload size distribution in bytes |
| `goque_payload_decode_errors_total` | Counter | `task_type` | Typed task payload JSON decode errors by task type |

##### Configuration

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

##### Task Processing Operations

Metrics track errors across different operations:
- `add_to_queue` - Errors during task creation and queue insertion
- `processing` - Errors during task execution
- `cleanup` - Errors during task cleanup operations
- `health` - Errors during healer operations

##### Example Queries

```promql
# Task processing rate by type
rate(goque_processed_tasks_total[5m])

# Task error rate
rate(goque_processed_tasks_with_error_total[5m])

# Typed payload decode errors by task type
sum by (task_type) (rate(goque_payload_decode_errors_total[5m]))

# Average processing duration
rate(goque_task_processing_duration_seconds_sum[5m])
  / rate(goque_task_processing_duration_seconds_count[5m])

# 95th percentile processing time
histogram_quantile(0.95, goque_task_processing_duration_seconds_bucket)

# Tasks by status
sum by (status) (goque_processed_tasks_total)
```

For a complete example with metrics integration, see [examples/service/](examples/service/).

#### Structured Logging (xlog)

Goque uses [xlog](https://github.com/ruko1202/xlog) for structured logging with support for multiple backends (Zap, Slog, custom adapters). See the Quick Start example above for configuration.

**What gets logged:**
- Task lifecycle events (creation, processing, completion)
- Error events and retry attempts
- Database operations and transaction management
- Worker pool events and task distribution

#### Distributed Tracing (OpenTelemetry)

Goque supports OpenTelemetry for distributed tracing. By default, uses a noop tracer (zero overhead). See the Quick Start example above for enabling tracing.

**Traced operations:**
- Task processing loop and fetching
- Hook execution (before/after)
- Task queue operations (add, get)

**Performance impact:**
- Noop (default): 0% overhead
- With 1% sampling: <0.1% memory overhead
- Without sampling: ~6% memory overhead

⚠️ **Important:** Call `goque.SetTracerProvider()` **before** creating Goque instances.

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
│   │   ├── mysql/task/         # MySQL storage (go-jet)
│   │   ├── sqlite/             # SQLite storage (go-jet)
│   │   ├── dbtx/               # ctx-aware tx executor (WithTx, WithinTx, Executor)
│   │   ├── dbentity/           # Cross-backend filter/query builders
│   │   └── dbutils/            # JSON validation + WHERE builders
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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
