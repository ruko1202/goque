# Goque
[![pipeline](https://github.com/ruko1202/goque/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/ruko1202/goque/actions/workflows/ci.yml)
![Coverage](https://img.shields.io/badge/Coverage-87.0%25-brightgreen)

**A task queue in the database you already run.** Reliable background jobs for
Go on top of PostgreSQL, MySQL or SQLite. No Kafka, RabbitMQ or Redis.

## Contents

- [Why goque](#why-goque)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Managing tasks](#managing-tasks)
- [Example service](#example-service)
- [Databases](#databases)
- [Configuration Options](#configuration-options)
- [Task Status Lifecycle](#task-status-lifecycle)
- [Delivery Guarantees](#delivery-guarantees)
- [Built-in Features](#built-in-features)
- [Development](#development)
- [Project Structure](#project-structure)

## Why goque

Emails, webhooks, polling external systems, reports: background work usually
gets its own broker, which has to be deployed, monitored and upgraded. The broker
is also outside your database transaction, so an order can be saved while the
task for it is lost.

goque keeps the queue in a table of the database you already have:

- **Transactional outbox in one line:** enqueue a task in the same transaction as
  your domain write, so both are committed or neither is ([example](#5-transactional-outbox)).
- **One table, the same code on three databases:** `goque_task` with three indexes
  on PostgreSQL, MySQL or SQLite; SQL is type-safe through go-jet.
- **Safe with many instances:** the fetch uses `FOR UPDATE SKIP LOCKED`, so two
  instances never take the same task, and a task version stops a worker that lost
  its task from overwriting the new owner's result.
- **Many task types, one manager:** each type gets its own worker pool and settings.
- **Retries, timeouts and a healer:** retries with your own backoff, a per-task
  timeout through `ctx`, stuck tasks returned to work, panics recovered, graceful
  shutdown.
- **Less code around each task:** typed payloads, `RescheduleTask` for "not ready
  yet" without spending an attempt, duplicate protection by external ID, cron jobs,
  before/after hooks.
- **Settings change without a restart:** workers, batch size, intervals and
  timeouts take a `Value[T]` that can follow your config or feature flags.
- **Observable:** Prometheus metrics with a ready Grafana dashboard, structured
  logs (zap or slog), OpenTelemetry tracing.

## Installation

```bash
go get github.com/ruko1202/goque
```

Requirements: Go 1.25+ and one of PostgreSQL 12+, MySQL 8+ (with `binlog_format`
`ROW` or `MIXED`, see [MySQL caveats](#mysql-caveats)) or SQLite 3+ for
development and tests.

## Quick Start

### 1. Apply migrations

Apply the migrations from [`migrations/`](migrations/) with
[goose](https://github.com/pressly/goose):

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest

goose -dir migrations/pg postgres "your-dsn" up      # PostgreSQL
goose -dir migrations/mysql mysql "your-dsn" up      # MySQL
goose -dir migrations/sqlite sqlite3 "your-dsn" up   # SQLite
```

To run the databases locally with Docker, see [Development](#development).

### 2. Create a Task Processor

Implement the `TaskProcessor` interface to define how your tasks should be processed:

```go
type EmailProcessor struct{}

func (p *EmailProcessor) ProcessTask(ctx context.Context, task *goque.Task) error {
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
    goque.WithCancelTaskWhenPayloadDecodeError[EmailPayload](),
)
```

If payload decoding fails, the typed processor is not called. The decode error is returned through the normal task processing flow, recorded in `goque_payload_decode_errors_total`, written to `task.Errors`, and passed to `WithHooksAfterProcessing`. Add `WithCancelTaskWhenPayloadDecodeError` to the typed processor to cancel decode failures instead of retrying them:

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

Return `goque.RescheduleTask(after)` (optionally wrapped) when the task is not finished yet,
e.g. while polling an external status. The task goes back to `new` and runs again after `after`;
`attempts` and `errors` stay unchanged. Rescheduling is unbounded, so the processor should
enforce its own deadline (e.g. based on `task.CreatedAt`).

### 3. Run Goque

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/ruko1202/goque"
    "github.com/ruko1202/goque/pkg/goquestorage"
    "github.com/ruko1202/xlog"
    "go.uber.org/zap"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    ctx := context.Background()

    // Structured logging with xlog (zap, slog or a custom adapter)
    logger := xlog.NewZapAdapter(zap.Must(zap.NewProduction()))
    xlog.ReplaceGlobalLogger(logger)
    ctx = xlog.ContextWithLogger(ctx, logger)

    // Optional: OpenTelemetry tracing, set before creating Goque
    // goque.SetTracerProvider(tracerProvider)

    // Optional: service name for metrics labels
    goque.SetMetricsServiceName("my-service")

    db, err := goquestorage.NewDBConn(ctx, &goquestorage.Config{
        DSN:         "postgres://user:pass@localhost:5432/goque?sslmode=disable",
        Driver:      goquestorage.PgxDriver,
        MaxOpenConn: 20,
        MaxIdleConn: 20, // at least the workers count, see "Databases"
    })
    if err != nil {
        return err
    }

    // Task storage for whichever database the connection points to
    taskStorage, err := goque.NewStorage(db)
    if err != nil {
        return err
    }

    // The Goque manager also runs the built-in healer and cleaner
    goq := goque.NewGoque(taskStorage)
    goq.RegisterProcessor(
        "send_email",
        &EmailProcessor{},
        goque.WithWorkersCount(goque.NewStaticValue(10)),
        goque.WithTaskProcessingMaxAttempts(goque.NewStaticValue[int32](3)),
        goque.WithTaskProcessingTimeout(goque.NewStaticValue(30*time.Second)),
    )

    if err := goq.Run(ctx); err != nil {
        return err
    }
    // Graceful shutdown: in-flight tasks finish before Stop returns
    defer goq.Stop()

    // Serve until SIGINT or SIGTERM
    sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
    defer stop()
    <-sigCtx.Done()

    return nil
}
```

### 4. Add Tasks to the Queue

```go
taskQueueManager := goque.NewTaskQueueManager(taskStorage)

// A task with a raw JSON payload
task := goque.NewTask("send_email", `{"to": "user@example.com", "subject": "Hello"}`)
err := taskQueueManager.AddTaskToQueue(ctx, task)

// With an external ID: enqueuing the same ID again returns goque.ErrDuplicateTask
dedupTask := goque.NewTaskWithExternalID("send_email", payload, "external-order-123")
err = taskQueueManager.AddTaskToQueue(ctx, dedupTask)

// With a typed payload marshaled to JSON
typedTask, err := goque.NewTaskWithPayload("send_email", EmailPayload{
    To:      "user@example.com",
    Subject: "Hello",
})
if err == nil {
    err = taskQueueManager.AddTaskToQueue(ctx, typedTask)
}
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

## Managing tasks

`TaskQueueManager` (from `goque.NewTaskQueueManager`) covers the rest of a task's life:

| Method | What it does |
|---|---|
| `AddTaskToQueue(ctx, task)` | Enqueues synchronously; joins a tx attached with `goque.WithTx` |
| `AsyncAddTaskToQueue(ctx, task)` | Enqueues in a background goroutine; errors are logged, not returned; never joins a tx |
| `GetTask(ctx, id)` | Returns one task |
| `GetTasks(ctx, filter, limit)` | Returns tasks matching a `goque.TaskFilter` (IDs, type, statuses, age) |
| `CancelTask(ctx, id)` | Moves a non-terminal task to `canceled`; a run already in flight is not stopped, its result is dropped |
| `ResetAttempts(ctx, id)` | Sets the task back to `new` with zero attempts |
| `WaitAsyncEnqueues()` | Waits for `AsyncAddTaskToQueue` goroutines; `Goque.Stop` calls it for you |

`CancelTask` and `ResetAttempts` return `goque.ErrTaskVersionConflict` if the task
changed between their read and write; see [Delivery Guarantees](#delivery-guarantees).

## Example service

A complete example service lives in [`examples/service`](examples/service/):

```bash
cd examples/service
make docker-up
# localhost:8080  dashboard and API
# localhost:3000  Grafana
# outbox on orders: POST /api/orders
```

It runs PostgreSQL, Prometheus, Loki, Grafana, Jaeger and an OpenTelemetry
collector, five task types and a load generator. See
[examples/service/README.md](examples/service/README.md) for the API and dashboards.

## Databases

| Database | Enqueue | Processing, 1 → 50 workers | Best For | Production Ready |
|----------|---------|----------------------------|----------|------------------|
| **PostgreSQL** | 0.44 ms | 1,040 → 4,810 tasks/s | High-throughput production systems | ✅ **Recommended** |
| **MySQL** | 1.09 ms | 400 → 2,050 tasks/s | Teams already running MySQL | ✅ Yes, see [MySQL caveats](#mysql-caveats) |
| **SQLite** | 0.06 ms | 9,840 → 8,170 tasks/s | Local development, testing, CI/CD | ⚠️ **Dev/Test only** |

Measured with `make bench` against Docker databases on one laptop; see
[PERFORMANCE_ANALYSIS.md](PERFORMANCE_ANALYSIS.md) for the setup and what each
number means. Throughput is bound by database round trips: each processed task
costs two single-row writes plus a share of one fetch transaction. SQLite looks
fastest only because it has no network hop; it serializes all writes on one
connection and one process.

**Size the connection pool for your workers.** Keep `MaxIdleConns` at least at
the total number of workers plus a few for the fetcher, healer and cleaner.
Below that, `database/sql` closes and reopens connections on every burst; with
50 workers on PostgreSQL that cut throughput by almost 3×. Across processes,
keep `instances × MaxOpenConns` below the server's `max_connections` (151 on
stock MySQL).

### Production notes

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

#### MySQL caveats

goque runs its own MySQL transactions (task fetch, healer, cleaner,
`ResetAttempts`) at **READ COMMITTED**, whatever the server default is.
Under InnoDB's default REPEATABLE READ the fetch's locking range scan
also locks the index gaps around it, and those gaps belong to tasks that
workers are moving to another status at the same moment; the two
`UPDATE`s deadlock (`Error 1213`). READ COMMITTED locks only the rows
the scan returns.

This needs `binlog_format` `ROW` (the default) or `MIXED`. With
`STATEMENT`, MySQL rejects writes in a READ COMMITTED transaction with
`Error 1665`.

#### Schema

Goque installs a single table named **`goque_task`** plus three indexes
(`goque_task_type_external_id_idx`, `goque_task_type_status_next_attempt_at_idx`,
`goque_task_type_status_updated_at_idx`). The full DDL lives in
[migrations/](migrations/) — apply it with `make db-up`.

## Configuration Options

The `GoqueProcessor` supports various configuration options. Scalar options take a
`goque.Value[T]`, so they can be constant or change at runtime (see [Dynamic Configuration](#dynamic-configuration)):

- `WithWorkersCount(v Value[int])` - Number of concurrent workers (default: 10, max: 10000)
- `WithWorkersPanicHandler(handler func(context.Context) func(any))` - Set a custom worker panic handler
- `WithTaskProcessingMaxAttempts(v Value[int32])` - Maximum retry attempts (default: 5)
- `WithTaskProcessingTimeout(v Value[time.Duration])` - Per-task timeout (default: 30s)
- `WithTaskProcessingNextAttemptAtFunc(f)` - Custom retry backoff strategy
- `WithTaskFetcherMaxTasks(v Value[int64])` - Maximum tasks to fetch per cycle (default: 100)
- `WithTaskFetcherTick(v Value[time.Duration])` - Fetch interval (default: 30s)
- `WithTaskFetcherTimeout(v Value[time.Duration])` - Timeout for fetching tasks from storage (default: 30s)
- `WithHooksBeforeProcessing(hooks ...HookBeforeProcessing)` - Add pre-processing hooks
- `WithHooksAfterProcessing(hooks ...HookAfterProcessing)` - Add post-processing hooks
- `WithCleanerPeriod(v Value[time.Duration])` - Cleaner run interval (default: 5m)
- `WithCleanerUpdatedAtTimeAgo(v Value[time.Duration])` - Completed-task age threshold for cleanup (default: 3h)
- `WithCleanerTimeout(v Value[time.Duration])` - Cleaner operation timeout (default: 30s)
- `WithHealerPeriod(v Value[time.Duration])` - Healer run interval (default: 5m)
- `WithHealerUpdatedAtTimeAgo(v Value[time.Duration])` - Stuck-task age threshold for healing (default: 1h)
- `WithHealerTimeout(v Value[time.Duration])` - Healer operation timeout (default: 30s)

All scalar values must be positive. An invalid (`<= 0`) value is never applied: the last valid value
(or the default) is used and a warning is logged.

### Dynamic Configuration

Every scalar option is read on each processor cycle, so it can be changed without a restart.

- `goque.NewStaticValue(v)` - a constant value.
- `goque.NewDynamicValue(ctx, default, fetch, interval)` - polls `fetch` every `interval` in the
  background and caches the result. The first fetch is synchronous; on error `default` is used, and
  later errors keep the last successful value. `fetch` gets a context with a timeout of `interval`
  and must honor it. Polling stops when `ctx` is done. Invalid arguments (nil `ctx`/`fetch`,
  `interval <= 0`) are logged and the value stays at `default`.
- Any type implementing `Get() T` (non-blocking, must not panic, safe for concurrent use) works as a
  `goque.Value[T]`.

```go
// Read workers count from your config system / feature flags every 10 seconds.
goq.RegisterProcessor(
    "send_email",
    &EmailProcessor{},
    goque.WithWorkersCount(goque.NewDynamicValue(ctx, 10, func(ctx context.Context) (int, error) {
        return configClient.GetInt(ctx, "queue.send_email.workers")
    }, 10*time.Second)),
    goque.WithTaskFetcherMaxTasks(goque.NewStaticValue[int64](50)),
)
```

When changes are applied:

- fetch `maxTasks`/`timeout`, processing `timeout`, `maxAttempts`, healer/cleaner `timeout` and
  `updatedAtTimeAgo` - on their next use (next fetch, task or run);
- workers count - on the next fetch tick or before the next task submit; the worker pool is resized in
  place and in-flight tasks are not interrupted. When the pool is full, growth applies once a running
  task finishes; after a shrink, extra workers retire as they finish their current task;
- fetch tick and healer/cleaner period - after the current tick/period elapses.

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
new ──→ pending ──→ processing ──┬──→ done
 ▲         ▲                     ├──→ canceled
 │         │                     ├──→ attempts_left
 │         └────── error ◀───────┤   failed, attempts left (fetched again when due)
 └───────────────────────────────┘   RescheduleTask, or returned on graceful shutdown

Healer:         stuck pending or processing ──→ error
CancelTask:     any non-terminal status     ──→ canceled
ResetAttempts:  any status                  ──→ new (attempts reset to 0)
```

### Status Descriptions

- **new** - Created and waiting until `next_attempt_at` is due
- **pending** - Taken by a fetcher, waiting for a free worker
- **processing** - Being processed by a worker
- **done** - Completed successfully ✓ (terminal)
- **error** - Failed with attempts left; fetched again once `next_attempt_at` (set by the backoff) is due
- **attempts_left** - Failed with no attempts left ✗ (terminal)
- **canceled** - Canceled by `CancelTask` or by the processor (`goque.ErrTaskCancel`) ✗ (terminal)

### Valid State Transitions

| Current Status | Next Status | Trigger |
|----------------|-------------|---------|
| `new`, `error` | `pending` | A fetcher takes a due task (`next_attempt_at` has passed) |
| `pending` | `processing` | A worker starts the task |
| `processing` | `done` | Successful processing |
| `processing` | `error` | Failed processing with attempts left; `next_attempt_at` set by the backoff |
| `processing` | `attempts_left` | Failed processing with no attempts left, or its attempts were already used up (cures from `processing` count as attempts), in which case it is not run |
| `processing` | `canceled` | The processor returned an error wrapping `goque.ErrTaskCancel` |
| `processing` | `new` | The processor returned `RescheduleTask(after)` (attempts unchanged), or graceful shutdown returned the task to the queue |
| `pending`, `processing` | `error` | Healer marks a stuck task (cure); a cure from `processing` consumes one attempt, a cure from `pending` does not |
| any non-terminal | `canceled` | `CancelTask` |
| any | `new` | `ResetAttempts` (attempts set to zero) |

### Terminal States

Tasks in these states will not be processed again:
- **done** - Successfully completed
- **canceled** - Canceled by `CancelTask` or the processor
- **attempts_left** - Failed with no remaining retry attempts

## Delivery Guarantees

Goque delivers tasks **at least once**. A task can run more than once, so processors
must be idempotent.

### What the task version guarantees

Every task row has a `version` (`Task.Version`). Fetching, healing, canceling and
resetting a task bump it, and every write made by a worker is applied only if the
stored version still matches the one the worker holds. A write made with a stale
version is dropped with `goque.ErrTaskVersionConflict` (passed to after-hooks, logged
and counted in `goque_task_version_conflicts_total`). So a worker that lost its task can never
overwrite the result of whoever owns the task now. The version protects goque's own
row only: it cannot undo side effects the stale worker already caused.

### When a task can run twice

- **The healer cures a task that is still running**, e.g. the processor ignores
  `ctx` and outlives its timeout, pod clocks drift apart, or the process stalls. The
  healer never uses a processing threshold below `2 × processing timeout + 10s` (a
  lower configured value is raised and logged), but that is best effort; the version
  keeps the row correct when it is not enough.
- **`CancelTask` on a pending or processing task** wins the row but does not stop an
  in-flight run: its side effects may still happen and its result is dropped.
  `canceled` means "the result will not be recorded", not "did not execute".
- **`ResetAttempts` on a pending or processing task** takes it away from the current
  run; the task may be fetched and run again while the first run is still going.

### What a processor must do

- Use `task.ID` (or your own business key) as the idempotency key for external calls.
- Make domain state changes conditional (`UPDATE payment SET status = 'captured'
  WHERE id = $1 AND status = 'authorized'`), not blind writes.
- Honor `ctx`: stop work when it is canceled or its deadline passes.

### Attempts, hooks and the healer

- A healer cure from `processing` counts as an attempt (and appends
  `attempt N: task was stuck` to `errors`): a task that keeps overrunning ends in
  `attempts_left` instead of running forever. A cure from `pending` does not: the task
  never started, so its `attempts` and `errors` stay as they are. Lowering
  `maxAttempts` at runtime also moves fetched tasks at or above the new limit to
  `attempts_left` without another run.
- Accepted trade-off: a task that keeps getting stuck between fetch and start (healer
  threshold shorter than the worker backlog, pods dying right after fetch, the start
  write failing with a database error) is cured and fetched again indefinitely, without
  ever reaching `attempts_left`. None of these causes is the task's fault; fix the
  configuration or the infrastructure. Each cycle is one row write per healer tick and
  the row does not grow.
- Hooks always run in pairs for every fetched task. If moving the task to `processing`
  fails (for example with `goque.ErrTaskVersionConflict`), the task is not processed and
  after-hooks get that error with the task still `pending`.
- A panic while processing a task (in the processor or a hook) is recovered, logged and
  fails the task like any error (it uses an attempt); after-hooks get
  `goque.ErrTaskPanic`.
- After-hooks may receive `goque.ErrTaskAttemptsExhausted` (task moved to
  `attempts_left` without running). If writing the result failed, `err` also wraps
  that error (for example `goque.ErrTaskVersionConflict`) and the task holds the result
  that was not persisted.

### Transactions

If you call `UpdateTask` inside your own transaction (`WithTx`) and roll it back,
re-read the task: its in-memory version is ahead of the database.

## Built-in Features

### Task Healer

Goque includes a built-in healer processor that automatically monitors and fixes stuck tasks. Tasks that remain in the "pending" or "processing" status for too long are automatically marked as errored, allowing them to be retried; a cure from `processing` consumes one attempt, a cure from `pending` does not. The healer is automatically registered when you call `goque.NewGoque()`.

Tasks are cured after `WithHealerUpdatedAtTimeAgo`, but never earlier than
`2 × processing timeout + 10s`: a smaller configured value would cure processing tasks a
live worker still holds, so it is raised and a WARN is logged on every healer run until
the configuration is fixed. See [Delivery Guarantees](#delivery-guarantees).

You can configure the healer behavior:

```go
goq.RegisterProcessor(
    "send_email",
    &EmailProcessor{},
    goque.WithWorkersCount(goque.NewStaticValue(10)),
    goque.WithTaskProcessingMaxAttempts(goque.NewStaticValue[int32](3)),
    goque.WithTaskProcessingTimeout(goque.NewStaticValue(30 * time.Second)),

    goque.WithHealerPeriod(goque.NewStaticValue(10*time.Minute)),
    goque.WithHealerUpdatedAtTimeAgo(goque.NewStaticValue(time.Hour)),
    goque.WithHealerTimeout(goque.NewStaticValue(30*time.Second)),
)
```

### Observability

#### Prometheus Metrics

Goque includes built-in Prometheus metrics for comprehensive monitoring of your task queue. Metrics are automatically collected during task processing operations.

> 📊 **Ready-to-import Grafana dashboard** lives in [`grafana/`](grafana/) — a Prometheus metrics dashboard with import instructions. See [`grafana/README.md`](grafana/README.md).

##### Available Metrics

| Metric Name | Type | Labels | Description |
|-------------|------|--------|-------------|
| `goque_processed_tasks_total` | Counter | `task_type`, `status` | Tasks by type and status: counted on enqueue (`new`), when a worker starts a task and when it finishes (final status) |
| `goque_task_processing_duration_seconds` | Histogram | `task_type`, `operation` | Duration of task processing and of the internal operations |
| `goque_operations_total` | Counter | `task_type`, `operation` | Tasks handled by the fetcher (`fetch`), cleaner (`cleanup`) and healer (`health`) |
| `goque_task_retry_attempts` | Histogram | `task_type` | Attempts a task had used when it finished |
| `goque_processors_workers_count` | Gauge | `task_type` | Current number of workers per task type |
| `goque_task_payload_size_bytes` | Histogram | `task_type` | Task payload size distribution in bytes |
| `goque_payload_decode_errors_total` | Counter | `task_type` | Typed task payload JSON decode errors by task type |
| `goque_task_version_conflicts_total` | Counter | `task_type`, `stage` | Task writes dropped because another writer changed the task (`stage`: `start`, `finish`, `return`). Expected ≈ 0; `finish` > 0 means a result was dropped |

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

##### Operations

The `operation` label takes these values:
- `fetch` - Taking tasks from the database
- `processing` - Running a task (duration only)
- `cleanup` - Removing old finished tasks
- `health` - Curing stuck tasks

##### Example Queries

```promql
# Task processing rate by type and status
sum by (task_type, status) (rate(goque_processed_tasks_total[5m]))

# Failure rate: tasks finishing in error or attempts_left
sum by (task_type) (rate(goque_processed_tasks_total{status=~"error|attempts_left"}[5m]))

# Typed payload decode errors by task type
sum by (task_type) (rate(goque_payload_decode_errors_total[5m]))

# Average processing duration
rate(goque_task_processing_duration_seconds_sum{operation="processing"}[5m])
  / rate(goque_task_processing_duration_seconds_count{operation="processing"}[5m])

# 95th percentile processing time
histogram_quantile(0.95, sum by (le, task_type) (rate(goque_task_processing_duration_seconds_bucket{operation="processing"}[5m])))

# Tasks fetched per second
sum by (task_type) (rate(goque_operations_total{operation="fetch"}[5m]))
```

For a complete example with metrics integration, see [examples/service/](examples/service/).

#### Structured Logging (xlog)

Goque uses [xlog](https://github.com/ruko1202/xlog) for structured logging with support for multiple backends (Zap, Slog, custom adapters). See [Run Goque](#3-run-goque) for configuration.

**What gets logged:**
- Task lifecycle events (creation, processing, completion)
- Error events and retry attempts
- Database operations and transaction management
- Worker pool events and task distribution

#### Distributed Tracing (OpenTelemetry)

Goque supports OpenTelemetry for distributed tracing. By default it uses a no-op
tracer. To enable tracing, call `goque.SetTracerProvider(tracerProvider)` as in
[Run Goque](#3-run-goque).

**Traced operations:**
- Task processing loop and fetching
- Hook execution (before/after)
- Task queue operations (add, get)

⚠️ **Important:** Call `goque.SetTracerProvider()` **before** creating Goque instances.

## Development

### Prerequisites

- Go 1.25 or higher
- Docker (for the local PostgreSQL and MySQL), or SQLite 3+
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

### Local databases

```bash
# Start PostgreSQL and MySQL with Docker Compose
make docker-up

# Configure your database connection in .env.local (one "KEY: value" per line)
# For PostgreSQL (recommended):
echo 'DB_DRIVER: postgres' > .env.local
echo 'DB_DSN: postgres://postgres:postgres@localhost:5432/goque?sslmode=disable' >> .env.local

# For MySQL:
# echo 'DB_DRIVER: mysql' > .env.local
# echo 'DB_DSN: root:root@tcp(localhost:3306)/goque?parseTime=true&loc=UTC' >> .env.local

# For SQLite (dev/test only):
# echo 'DB_DRIVER: sqlite3' > .env.local
# echo 'DB_DSN: ./goque.db' >> .env.local

# Install database tools (goose)
make bin-deps-db

# Apply migrations
make db-up
```

For detailed database setup instructions, see [DATABASE.md](DATABASE.md).

### Running Tests

```bash
# Run all tests
make tloc

# Run tests with coverage
make test-cov
```

### Benchmarks

```bash
# All benchmarks against all databases
make bench

# One benchmark, one database, five runs
DB_DRIVER=pgx make bench BENCH=Processing BENCH_COUNT=5
```

Results and how to read them: [PERFORMANCE_ANALYSIS.md](PERFORMANCE_ANALYSIS.md).

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

# Regenerate database models for the current DB_DRIVER
make db-models

# Regenerate database models for all databases
make all-db-models

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
├── goque.go                    # Goque: processor registration, Run/Stop
├── goque_manager.go            # Public API - TaskQueueManager interface
├── goque_*.go                  # Public API - tasks, processors, options, values, periodic jobs, tx, metrics, tracer
├── pkg/
│   └── goquestorage/           # Database connection helper (NewDBConn, driver constants)
├── internal/
│   ├── entity/                 # Domain entities and errors
│   ├── queuemanager/           # Task queue manager implementation
│   ├── metrics/                # Prometheus metrics
│   ├── processors/             # Task processing components
│   │   ├── queueprocessor/     # Main queue processor (plain and typed)
│   │   ├── periodicprocessor/  # Periodic job scheduler
│   │   └── internalprocessors/ # Built-in processors (healer, cleaner)
│   ├── storages/               # Data access layer (multi-database support)
│   │   ├── pg/task/            # PostgreSQL storage (go-jet)
│   │   ├── mysql/task/         # MySQL storage (go-jet)
│   │   ├── sqlite/             # SQLite storage (go-jet)
│   │   ├── dbtx/               # ctx-aware tx executor (WithTx, WithinTx, Executor)
│   │   ├── dbentity/           # Cross-backend filter/query builders
│   │   └── dbutils/            # JSON validation + WHERE builders
│   ├── utils/                  # Internal helpers (dynamic values, tracing, time, pools)
│   └── pkg/
│       └── generated/          # Generated code (models, mocks)
├── migrations/                 # Database migrations
│   ├── pg/                     # PostgreSQL migrations
│   ├── mysql/                  # MySQL migrations
│   └── sqlite/                 # SQLite migrations
├── grafana/                    # Ready-to-import Grafana dashboard
├── examples/service/           # Production-ready example service
├── DATABASE.md                 # Complete database setup guide
└── test/                       # Test utilities and fixtures
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
