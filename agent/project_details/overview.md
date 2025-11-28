# Project Overview

## What is Goque?

Goque is a robust, database-backed task queue system for Go applications with support for PostgreSQL, MySQL, and SQLite. It provides reliable task processing with built-in worker pools, retry logic, and graceful shutdown support.

## Core Purpose

Goque solves the problem of reliable asynchronous task processing in distributed systems by:
- Providing durable task storage using PostgreSQL, MySQL, or SQLite
- Managing concurrent task processing with worker pools
- Handling failures automatically with retry logic
- Ensuring tasks don't get stuck with automatic healing
- Cleaning up old completed/failed tasks automatically

## Key Features

### 1. Multi-Database Persistence
- Supports PostgreSQL 12+, MySQL 8+, and SQLite 3+
- Reliable task storage with ACID guarantees
- Tasks survive application restarts
- No data loss on crashes

### 2. Worker Pool Management
- Configurable concurrent task processing
- Uses goroutine pools via ants library
- Efficient resource utilization

### 3. Automatic Retry Logic
- Configurable retry attempts
- Custom backoff strategies
- Per-task attempt tracking

### 4. Task Lifecycle Management

**Task states:**
- `new` - Ready to be picked up
- `pending` - Scheduled for future processing (via NextAttemptAt)
- `processing` - Currently being processed by a worker
- `done` - Completed successfully ✓ (terminal)
- `error` - Failed but has retry attempts remaining
- `attempts_left` - Failed and exhausted all retries ✗ (terminal)
- `canceled` - Manually canceled ✗ (terminal)

**State flow:**
```
new → pending → processing → ✅ done
        ↑ ↓         ↓    ↓
        │ └──error──┘  ❌ canceled
        │     ↓
        │  ❌ attempts_left
        │
        └──(healer fixes stuck pending)
```

**Terminal states** (no further processing): `done`, `canceled`, `attempts_left`

### 5. Built-in Task Healer
- Monitors stuck tasks in `pending` status
- Automatically marks them as `error` for retry
- Prevents tasks from being lost

### 6. Built-in Task Cleaner
- Removes old completed tasks
- Removes old failed tasks
- Keeps database clean and performant

### 7. Graceful Shutdown
- Cleanly stops worker pools
- Waits for in-flight tasks to complete
- No task interruption during shutdown

### 8. Extensible Hooks
- Before/after processing hooks
- Custom logic injection
- Monitoring and logging support

### 9. Type-safe Queries
- Uses go-jet for type-safe SQL generation
- Compile-time query validation
- IDE auto-completion support

### 10. External ID Support
- Associate tasks with external identifiers
- Idempotency support
- Duplicate task prevention

### 11. Prometheus Metrics
- Built-in Prometheus metrics integration
- Task processing counters and error tracking
- Duration histograms for performance monitoring
- Payload size tracking

## Use Cases

1. **Email Processing** - Send emails asynchronously
2. **Data Processing** - Process large datasets in background
3. **Webhook Delivery** - Reliable webhook delivery with retries
4. **Scheduled Jobs** - Execute tasks at specific times
5. **Image/Video Processing** - Heavy processing tasks
6. **Notifications** - Push notifications with retry logic
7. **Report Generation** - Generate reports asynchronously

## Architecture Highlights

- **Modular Design** - Components are loosely coupled
- **Interface-based** - Easy to mock and test
- **Storage Abstraction** - Storage layer is abstracted
- **Processor Pattern** - Easy to add new task types
- **Context-aware** - Proper context propagation
- **Thread-safe** - Safe for concurrent use

## Project Structure

```
goque/
├── goque_manager.go              # Public API - TaskQueueManager interface
├── internal/
│   ├── goque.go                  # Main Goque implementation
│   ├── entity/                   # Domain entities and errors
│   ├── queue_manager/            # Task queue manager implementation
│   ├── metrics/                  # Prometheus metrics
│   ├── processors/
│   │   ├── queueprocessor/       # Core task processor
│   │   └── internalprocessors/   # Healer, Cleaner
│   └── storages/
│       ├── pg/                   # PostgreSQL storage
│       ├── mysql/                # MySQL storage
│       ├── sqlite/               # SQLite storage
│       └── dbutils/              # Database utilities
├── migrations/
│   ├── pg/                       # PostgreSQL migrations
│   ├── mysql/                    # MySQL migrations
│   └── sqlite/                   # SQLite migrations
├── examples/service/             # Production-ready example service
└── test/                         # Integration tests
```

## Target Users

- Go developers building distributed systems
- Teams needing reliable background job processing
- Applications requiring task queues with PostgreSQL, MySQL, or SQLite
- Systems needing automatic retry and healing
- Projects needing embedded database support (SQLite)
