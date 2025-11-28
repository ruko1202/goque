# Architecture

## System Architecture

Goque follows a layered architecture with clear separation of concerns.

```
┌─────────────────────────────────────────┐
│         Application Layer               │
│  (User code using Goque)                │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         Public API (goque.go)           │
│  - Goque Manager                        │
│  - QueueMngr (task submission)          │
└──────────┬────────────┬─────────────────┘
           │            │
┌──────────▼────────┐   │
│  Queue Processor  │   │
│  - GoqueProcessor │   │
│  - Task execution │   │
│  - Worker pools   │   │
└─────────┬─────────┘   │
          │             │
┌─────────▼─────────────▼─────────────────┐
│    Internal Processors                  │
│  - Healer (fix stuck tasks)             │
│  - Cleaner (remove old tasks)           │
└─────────┬───────────────────────────────┘
          │
┌─────────▼───────────────────────────────┐
│       Storage Layer                     │
│  - Task Storage (Multi-DB)              │
│  - PostgreSQL / MySQL / SQLite impls    │
│  - Type-safe queries (go-jet)           │
└─────────┬───────────────────────────────┘
          │
┌─────────▼───────────────────────────────┐
│  Database (PostgreSQL/MySQL/SQLite)     │
│  - tasks table                          │
└─────────────────────────────────────────┘
```

## Core Components

### 1. Goque Manager (`goque.go`)

**Purpose**: Main coordinator for task processing

**Responsibilities**:
- Register multiple task processors
- Start all processors concurrently
- Coordinate graceful shutdown
- Manage processor lifecycle

**Key Methods**:
- `NewGoque(taskStorage)` - Create manager
- `RegisterProcessor(type, processor, opts)` - Register processor
- `Run(ctx)` - Start all processors
- `Stop()` - Graceful shutdown

**Pattern**: Facade pattern, coordinates multiple processors

### 2. Queue Processor (`internal/processors/queueprocessor/`)

**Purpose**: Core task processing engine

**Components**:

#### GoqueProcessor
- Fetches tasks from storage
- Distributes tasks to worker pool
- Manages task lifecycle
- Handles retries and errors
- Executes hooks

#### Task Processor Interface
```go
type TaskProcessor interface {
    ProcessTask(ctx context.Context, task *entity.Task) error
}
```
- User implements this interface
- Contains business logic
- Receives task, processes it

#### Worker Pool
- Uses `ants` library for goroutine pooling
- Configurable worker count
- Efficient task execution

**Flow**:
1. Fetch tasks from storage (batch)
2. For each task:
   - Update status to `processing`
   - Submit to worker pool
   - Execute before hooks
   - Call TaskProcessor.ProcessTask()
   - Execute after hooks
   - Update status based on result
3. Schedule next fetch

### 3. Internal Processors (`internal/processors/internalprocessors/`)

#### Healer
**Purpose**: Fix stuck tasks

**Logic**:
- Periodically scans for tasks in `pending` status
- If task's `updated_at` is older than threshold (default: 5 min)
- Marks task as `error` to allow retry
- Prevents tasks from being permanently stuck

**Configuration**:
- `WithHealerUpdatedAtTimeAgo(duration)` - Time threshold
- `WithHealerMaxTasks(n)` - Max tasks to heal per cycle

#### Cleaner
**Purpose**: Remove old tasks

**Logic**:
- Periodically scans for old tasks
- Removes tasks in `done` or `attempts_left` status
- Configurable retention period
- Keeps database clean

**Configuration**:
- `WithCleanerUpdatedAtTimeAgo(duration)` - Age threshold
- `WithCleanerMaxTasks(n)` - Max tasks to clean per cycle

### 4. Queue Manager (`internal/queuemngr/`)

**Purpose**: Task submission interface

**Responsibilities**:
- Provide simple API to add tasks
- Validate task data
- Submit to storage

**Key Methods**:
- `PushTask(ctx, type, payload)` - Add new task
- `PushTaskWithExternalID(ctx, type, payload, externalID)` - Add with external ID

### 5. Storage Layer (`pkg/goquestorage/`, `internal/storages/`)

**Purpose**: Multi-database persistence layer with PostgreSQL, MySQL, and SQLite support

**Implementations**:
- **PostgreSQL** (`internal/storages/pg/task/`) - Primary implementation
  - Uses go-jet for type-safe queries
  - Row-level locking with FOR UPDATE SKIP LOCKED
- **MySQL** (`internal/storages/mysql/task/`) - Alternative implementation
  - Uses go-jet for type-safe queries
  - Row-level locking with FOR UPDATE
- **SQLite** (`internal/storages/sqlite/task/`) - Embedded database implementation
  - Uses go-jet for type-safe queries
  - Transaction-based locking
  - Ideal for development and small deployments
- **Common utilities** (`internal/storages/dbutils/`) - Shared database utilities
- **Shared entities** (`internal/storages/dbentity/`) - Common filters and entities

**Key Operations** (same interface for all databases):
- `AddTask(ctx, task)` - Insert task
- `GetTasksForProcessing(ctx, type, limit)` - Fetch tasks to process
- `UpdateTask(ctx, task)` - Update task status
- `CureTasks(ctx, type, updatedBefore, limit)` - Healer operation
- `DeleteTasks(ctx, statuses, updatedBefore, limit)` - Cleaner operation

**Features**:
- Type-safe queries using go-jet for all databases
- Transaction support with automatic rollback
- Row-level locking (FOR UPDATE SKIP LOCKED / FOR UPDATE / transaction-based)
- Efficient batch operations
- Database-agnostic public interface

## Data Flow

### Task Submission Flow

```
User Code
  │
  ├─> QueueMngr.PushTask()
  │
  ├─> entity.NewTask(type, payload)
  │
  └─> TaskStorage.AddTask()
      │
      └─> INSERT INTO tasks (status='new', ...)
```

### Task Processing Flow

```
GoqueProcessor.Run()
  │
  ├─> Fetch Loop (every FetchTick)
  │   │
  │   ├─> TaskStorage.GetTasksForProcessing()
  │   │   └─> SELECT ... WHERE status='new' FOR UPDATE SKIP LOCKED
  │   │
  │   ├─> For each task:
  │   │   │
  │   │   ├─> Update status to 'processing'
  │   │   │
  │   │   ├─> Submit to worker pool
  │   │   │   │
  │   │   │   ├─> Execute before hooks
  │   │   │   │
  │   │   │   ├─> TaskProcessor.ProcessTask()
  │   │   │   │   └─> User's business logic
  │   │   │   │
  │   │   │   ├─> Execute after hooks
  │   │   │   │
  │   │   │   └─> Update task status:
  │   │   │       ├─> Success: status='done'
  │   │   │       └─> Error:
  │   │   │           ├─> Has retries: status='error', next_attempt_at=...
  │   │   │           └─> No retries: status='attempts_left'
  │   │   │
  │   └─> Sleep(FetchTick)
  │
  └─> On Stop():
      └─> Wait for worker pool to drain
```

### Healing Flow

```
Healer.Run()
  │
  ├─> Periodic Loop (every interval)
  │   │
  │   ├─> TaskStorage.CureTasks()
  │   │   └─> UPDATE tasks
  │   │       SET status='error', attempts=attempts-1
  │   │       WHERE status='pending'
  │   │       AND updated_at < NOW() - threshold
  │   │
  │   └─> Tasks become available for retry
  │
  └─> On Stop(): Exit loop
```

### Cleaning Flow

```
Cleaner.Run()
  │
  ├─> Periodic Loop (every interval)
  │   │
  │   ├─> TaskStorage.DeleteTasks()
  │   │   └─> DELETE FROM tasks
  │   │       WHERE status IN ('done', 'attempts_left')
  │   │       AND updated_at < NOW() - threshold
  │   │
  │   └─> Old tasks removed
  │
  └─> On Stop(): Exit loop
```

## Key Design Patterns

### 1. Strategy Pattern
- TaskProcessor interface allows different processing strategies
- User implements interface with custom logic

### 2. Repository Pattern
- Storage layer abstracted behind interfaces
- Easy to mock for testing
- Easy to add new storage backends

### 3. Worker Pool Pattern
- Goroutine pool for efficient concurrency
- Resource limits and control

### 4. Hook Pattern
- Before/after processing hooks
- Extensibility without modifying core

### 5. Options Pattern
- Functional options for configuration
- Clean, extensible API

## Concurrency Model

### Task Fetching
- Single goroutine per processor fetches tasks
- Batch fetching for efficiency
- `FOR UPDATE SKIP LOCKED` prevents contention

### Task Processing
- Worker pool handles concurrent execution
- Each worker is a goroutine
- Configurable pool size

### Graceful Shutdown
- Stop() signals all processors
- Worker pools drain gracefully
- In-flight tasks complete

## Database Schema

```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    type VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    payload JSONB,
    external_id VARCHAR(255),
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    next_attempt_at TIMESTAMP,
    error TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,

    INDEX idx_tasks_processing (type, status, next_attempt_at)
    UNIQUE INDEX idx_tasks_external_id (external_id)
);
```

**Key Indexes**:
- `idx_tasks_processing` - Optimize task fetching
- `idx_tasks_external_id` - Ensure external ID uniqueness

## Scalability Considerations

### Horizontal Scaling
- Multiple instances can process tasks concurrently
- `FOR UPDATE SKIP LOCKED` prevents duplicate processing
- Each instance can run same or different processors

### Vertical Scaling
- Increase worker count per processor
- Limited by database connections
- Limited by CPU/memory

### Database Scaling
- PostgreSQL handles high concurrency well
- Read replicas not needed (write-heavy workload)
- Connection pooling recommended
