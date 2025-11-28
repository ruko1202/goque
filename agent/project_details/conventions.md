# Code Conventions

## General Principles

1. **Simplicity** - Prefer simple, obvious solutions
2. **Readability** - Code is read more than written
3. **Consistency** - Follow existing patterns
4. **Testability** - Design for easy testing
5. **Documentation** - Document non-obvious code

## Go Style

### Follow Standard Go Style
- Follow [Effective Go](https://golang.org/doc/effective_go)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` and `goimports`

### Formatting
```bash
# Format code
make fmt
```
- Uses `golangci-lint fmt -E gofmt -E goimports`
- Automatic import organization
- Consistent formatting

## Naming Conventions

### Packages
```go
// Good
package task
package processor
package queuemngr

// Bad
package task_storage
package TaskProcessor
```
- Short, lowercase
- No underscores
- Singular form preferred

### Files
```
// Good
task.go
processor.go
add_task.go
get_task.go

// Bad
Task.go
task-storage.go
addTask.go
```
- Lowercase
- Use underscores for multi-word names
- Descriptive names

### Types
```go
// Good
type Task struct { ... }
type TaskProcessor interface { ... }
type GoqueProcessor struct { ... }

// Bad
type task struct { ... }
type ITaskProcessor interface { ... }
type goque_processor struct { ... }
```
- PascalCase for exported
- camelCase for unexported
- No prefix/suffix for interfaces (except where standard)

### Variables
```go
// Good
var taskStorage TaskStorage
var maxAttempts int
var db *sql.DB

// Bad
var TaskStorage TaskStorage
var max_attempts int
var database *sql.DB
```
- camelCase
- Descriptive names
- Short names for small scope, longer for wider scope

### Constants
```go
// Good
const DefaultMaxAttempts = 3
const defaultFetchTick = time.Second

// Bad
const DEFAULT_MAX_ATTEMPTS = 3
const dEFAULTfETCHtICK = time.Second
```
- Follow same rules as variables
- Use iota for related constants

### Functions
```go
// Good
func NewGoque(storage TaskStorage) *Goque
func ProcessTask(ctx context.Context, task *entity.Task) error
func getTasksForProcessing(ctx context.Context) ([]*entity.Task, error)

// Bad
func new_goque(storage TaskStorage) *Goque
func process_task(ctx context.Context, task *entity.Task) error
func GetTasks_For_Processing(ctx context.Context) ([]*entity.Task, error)
```
- PascalCase for exported
- camelCase for unexported
- Verb-based names

## Code Organization

### Package Structure
```
internal/
├── processors/
│   ├── queueprocessor/
│   │   ├── processor.go          # Main processor
│   │   ├── processor_opts.go     # Options
│   │   ├── processor_test.go     # Tests
│   │   ├── task_processor.go     # Interface
│   │   └── hooks.go              # Hooks
│   └── internalprocessors/
│       ├── healer.go
│       └── cleaner.go
└── storages/
    └── sql/pg/task/
        ├── storage.go            # Main storage
        ├── add_task.go           # One file per operation
        ├── add_task_test.go
        ├── get_task.go
        └── get_task_test.go
```

### File Organization
```go
// 1. Package declaration
package task

// 2. Imports (grouped)
import (
    // Standard library
    "context"
    "time"

    // External dependencies
    "github.com/google/uuid"

    // Internal dependencies
    "github.com/ruko1202/goque/pkg/entity"
)

// 3. Constants
const (
    DefaultMaxAttempts = 3
)

// 4. Types
type Storage struct {
    db *sql.DB
}

// 5. Constructor
func NewStorage(db *sql.DB) *Storage {
    return &Storage{db: db}
}

// 6. Methods
func (s *Storage) AddTask(ctx context.Context, task *entity.Task) error {
    // Implementation
}

// 7. Helper functions
func validateTask(task *entity.Task) error {
    // Implementation
}
```

## Interface Design

### Small Interfaces
```go
// Good
type TaskProcessor interface {
    ProcessTask(ctx context.Context, task *entity.Task) error
}

type TaskStorage interface {
    AddTask(ctx context.Context, task *entity.Task) error
    GetTasksForProcessing(ctx context.Context, taskType string, limit int64) ([]*entity.Task, error)
    UpdateTask(ctx context.Context, task *entity.Task) error
}

// Bad (too large)
type TaskManager interface {
    AddTask(...)
    GetTask(...)
    UpdateTask(...)
    DeleteTask(...)
    ProcessTask(...)
    RetryTask(...)
    CancelTask(...)
    // ... 20 more methods
}
```
- Keep interfaces focused
- Single responsibility
- Easy to mock

### Interface Naming
```go
// Good
type TaskProcessor interface { ... }
type TaskStorage interface { ... }

// Acceptable when standard
type Reader interface { ... }
type Writer interface { ... }

// Bad
type ITaskProcessor interface { ... }
type TaskProcessorInterface interface { ... }
```

## Error Handling

### Return Errors
```go
// Good
func ProcessTask(ctx context.Context, task *entity.Task) error {
    if task == nil {
        return errors.New("task is nil")
    }
    // ...
    if err != nil {
        return fmt.Errorf("failed to process task: %w", err)
    }
    return nil
}

// Bad
func ProcessTask(ctx context.Context, task *entity.Task) {
    if task == nil {
        panic("task is nil")
    }
    // ...
    if err != nil {
        log.Fatal(err)
    }
}
```

### Error Wrapping
```go
// Good
if err != nil {
    return fmt.Errorf("failed to add task: %w", err)
}

// Bad
if err != nil {
    return errors.New("error occurred")
}
```
- Use `%w` to wrap errors
- Add context to errors
- Don't lose original error

### Custom Errors
```go
// Good
var (
    ErrTaskNotFound = errors.New("task not found")
    ErrInvalidStatus = errors.New("invalid task status")
)

// Usage
if task == nil {
    return ErrTaskNotFound
}

// Checking
if errors.Is(err, ErrTaskNotFound) {
    // Handle
}
```

## Context Usage

### Always Accept Context
```go
// Good
func ProcessTask(ctx context.Context, task *entity.Task) error

// Bad
func ProcessTask(task *entity.Task) error
```
- First parameter
- Enable cancellation
- Enable timeout

### Propagate Context
```go
// Good
func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
    // Pass context down
    err := p.taskProcessor.ProcessTask(ctx, task)
    // ...
    err = p.taskStorage.UpdateTask(ctx, task)
    return err
}
```

### Check Context
```go
// Good
select {
case <-ctx.Done():
    return ctx.Err()
case task := <-taskChan:
    // Process
}
```

## Concurrency

### Use Worker Pools
```go
// Good - Using ants
pool, err := ants.NewPool(workerCount)
if err != nil {
    return err
}
defer pool.Release()

err = pool.Submit(func() {
    processTask(ctx, task)
})
```
- Don't spawn unlimited goroutines
- Use ants for goroutine pooling

### Protect Shared State
```go
// Good
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}

// Bad
type Counter struct {
    count int  // Unprotected!
}
```

### Use Channels for Communication
```go
// Good
taskChan := make(chan *entity.Task)
go func() {
    for task := range taskChan {
        processTask(task)
    }
}()
```

## Testing

### Test File Naming
```
add_task.go      -> add_task_test.go
processor.go     -> processor_test.go
```

### Test Function Naming
```go
// Good
func TestAddTask(t *testing.T)
func TestAddTask_DuplicateExternalID(t *testing.T)
func TestAddTask_NilTask(t *testing.T)

// Bad
func TestAdd(t *testing.T)
func Test_add_task(t *testing.T)
func TestAddTaskWithDuplicateExternalID(t *testing.T)
```
- `Test<FunctionName>`
- Use underscore for variations

### Table-Driven Tests
```go
func TestProcessTask(t *testing.T) {
    tests := []struct {
        name    string
        task    *entity.Task
        wantErr bool
    }{
        {
            name: "success",
            task: &entity.Task{...},
            wantErr: false,
        },
        {
            name: "nil task",
            task: nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ProcessTask(ctx, tt.task)
            if (err != nil) != tt.wantErr {
                t.Errorf("ProcessTask() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Use testify
```go
// Good
require.NoError(t, err)
assert.Equal(t, expected, actual)
assert.NotNil(t, result)

// Bad
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
if expected != actual {
    t.Errorf("expected %v, got %v", expected, actual)
}
```

## Comments

### Package Comments
```go
// Package queueprocessor provides core task processing functionality
// for the Goque task queue system.
package queueprocessor
```

### Function Comments
```go
// NewGoque creates a new Goque instance with the specified task storage.
// The returned Goque instance is ready to register processors but not started.
func NewGoque(taskStorage TaskStorage) *Goque {
    // ...
}
```
- Complete sentence
- Start with function name
- Describe what, not how

### Type Comments
```go
// TaskProcessor defines the interface for processing individual tasks.
// Implementations contain the business logic for a specific task type.
type TaskProcessor interface {
    // ProcessTask processes a single task and returns an error if processing fails.
    // The context should be respected for cancellation and timeout.
    ProcessTask(ctx context.Context, task *entity.Task) error
}
```

### Avoid Obvious Comments
```go
// Bad
// Set the count to 0
count = 0

// Increment the counter
count++

// Good (when needed)
// Reset counter to avoid integer overflow after 1 million tasks
count = 0
```

## Options Pattern

### Functional Options
```go
// Good
type options struct {
    workers     int
    maxAttempts int32
    timeout     time.Duration
}

type GoqueProcessorOpts func(*options)

func WithWorkers(n int) GoqueProcessorOpts {
    return func(o *options) {
        o.workers = n
    }
}

// Usage
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithWorkers(10),
    WithMaxAttempts(3),
)
```

## Database Queries

### Use go-jet
```go
// Good
query := Task.
    SELECT(Task.AllColumns).
    WHERE(
        Task.Type.EQ(String(taskType)).
            AND(Task.Status.EQ(String(StatusNew))),
    ).
    LIMIT(limit).
    FOR(UPDATE().SKIP_LOCKED())

// Bad
query := "SELECT * FROM tasks WHERE type = $1 AND status = $2 LIMIT $3 FOR UPDATE SKIP LOCKED"
```

### One File Per Query
```
add_task.go
get_task.go
update_task.go
delete_task.go
```

## Logging

### Structured Logging
```go
// Good
slog.InfoContext(ctx, "processing task",
    slog.String("task_id", task.ID.String()),
    slog.String("task_type", task.Type),
)

// Bad
log.Printf("processing task %s of type %s", task.ID, task.Type)
```

### Log Levels
- **Debug**: Detailed diagnostic information
- **Info**: General informational messages
- **Warn**: Warning messages
- **Error**: Error messages

## Performance

### Batch Operations
```go
// Good
tasks, err := storage.GetTasksForProcessing(ctx, taskType, 10)

// Bad
for i := 0; i < 10; i++ {
    task, err := storage.GetTaskForProcessing(ctx, taskType)
}
```

### Use Pointers for Large Structs
```go
// Good
func ProcessTask(ctx context.Context, task *entity.Task) error

// Bad
func ProcessTask(ctx context.Context, task entity.Task) error
```

### Avoid Premature Optimization
- Profile before optimizing
- Measure performance
- Optimize only bottlenecks
