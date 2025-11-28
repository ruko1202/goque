# Design Principles

## Core Principles

### 1. Reliability First

**Principle**: Tasks must not be lost

**Implementation**:
- PostgreSQL persistence with ACID guarantees
- Tasks survive application crashes
- Automatic retry mechanism
- Healer processor fixes stuck tasks
- Row-level locking prevents duplicate processing

**Example**:
```go
// Tasks are persisted immediately
err := storage.AddTask(ctx, task)
// Task is now safe in database

// FOR UPDATE SKIP LOCKED ensures no duplicate processing
query := Task.SELECT(Task.AllColumns).
    WHERE(Task.Status.EQ(String(StatusNew))).
    FOR(UPDATE().SKIP_LOCKED())
```

### 2. Simplicity

**Principle**: Simple solutions over complex ones

**Implementation**:
- Small, focused interfaces
- Clear separation of concerns
- Minimal dependencies
- Obvious code over clever code

**Example**:
```go
// Simple interface - one method, one responsibility
type TaskProcessor interface {
    ProcessTask(ctx context.Context, task *entity.Task) error
}

// Not:
// type TaskProcessor interface {
//     ProcessTask(...)
//     ValidateTask(...)
//     BeforeProcess(...)
//     AfterProcess(...)
//     HandleError(...)
//     ... 10 more methods
// }
```

### 3. Composability

**Principle**: Build complex systems from simple components

**Implementation**:
- Goque manager composes multiple processors
- Each processor handles one task type
- Storage, processor, and healer are independent
- Easy to add new processors

**Example**:
```go
goque := NewGoque(storage)

// Compose different processors
goque.RegisterProcessor("send_email", emailProcessor, opts...)
goque.RegisterProcessor("generate_report", reportProcessor, opts...)
goque.RegisterProcessor("process_image", imageProcessor, opts...)

// All run independently
goque.Run(ctx)
```

### 4. Extensibility

**Principle**: Easy to extend without modifying core

**Implementation**:
- Hook system for custom logic
- Options pattern for configuration
- Interface-based design
- No need to fork for customization

**Example**:
```go
// Add custom logic without changing core
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithHooksBeforeProcessing(func(ctx context.Context, task *entity.Task) error {
        // Custom logging
        // Metrics collection
        // Validation
        return nil
    }),
    WithHooksAfterProcessing(func(ctx context.Context, task *entity.Task, err error) error {
        // Custom error handling
        // Notifications
        return nil
    }),
)
```

### 5. Type Safety

**Principle**: Catch errors at compile time, not runtime

**Implementation**:
- go-jet for type-safe SQL
- Strong typing throughout
- No stringly-typed code
- Interfaces for contracts

**Example**:
```go
// Type-safe query
query := Task.
    SELECT(Task.AllColumns).
    WHERE(Task.Type.EQ(String(taskType)))

// Not:
// query := "SELECT * FROM tasks WHERE type = " + taskType  // SQL injection!
// query := fmt.Sprintf("SELECT * FROM tasks WHERE type = '%s'", taskType)  // Still unsafe!
```

### 6. Testability

**Principle**: Code must be easy to test

**Implementation**:
- Dependency injection
- Interface-based design
- Mock generation
- No global state

**Example**:
```go
// Easy to test - inject mock storage
func TestProcessTask(t *testing.T) {
    mockStorage := mocks.NewMockTaskStorage(ctrl)
    mockStorage.EXPECT().UpdateTask(gomock.Any(), gomock.Any()).Return(nil)

    processor := NewGoqueProcessor(mockStorage, "test", testProcessor)
    // Test processor
}
```

### 7. Explicit Over Implicit

**Principle**: Be explicit about behavior

**Implementation**:
- No hidden magic
- Clear error handling
- Explicit configuration
- Obvious control flow

**Example**:
```go
// Explicit configuration
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithWorkers(10),           // Explicit: 10 workers
    WithMaxAttempts(3),        // Explicit: 3 retries
    WithTaskTimeout(30*time.Second),  // Explicit: 30s timeout
)

// Not:
// processor := NewAutoProcessor("email", emailProcessor)
// // What's the worker count? Timeout? Max attempts?
// // All implicit and hidden!
```

### 8. Fail Fast

**Principle**: Detect errors early

**Implementation**:
- Validate inputs immediately
- Return errors, don't hide them
- No silent failures
- Compile-time checks where possible

**Example**:
```go
// Validate immediately
func (g *Goque) Run(ctx context.Context) error {
    if len(g.processors) == 0 {
        return errors.New("no processors to run")  // Fail fast
    }
    // ...
}

// Not:
// func (g *Goque) Run(ctx context.Context) {
//     if len(g.processors) == 0 {
//         return  // Silent failure!
//     }
// }
```

### 9. Context-Aware

**Principle**: Respect context for cancellation and timeout

**Implementation**:
- Context as first parameter
- Propagate context down the call stack
- Check context cancellation
- Context-aware logging

**Example**:
```go
func (p *GoqueProcessor) processTask(ctx context.Context, task *entity.Task) error {
    // Respect timeout from context
    err := p.taskProcessor.ProcessTask(ctx, task)

    // Check if context was cancelled
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // Context-aware logging
    slog.InfoContext(ctx, "task processed", slog.String("task_id", task.ID.String()))
    return nil
}
```

### 10. Graceful Degradation

**Principle**: Handle failures gracefully

**Implementation**:
- Automatic retry with backoff
- Healer fixes stuck tasks
- Graceful shutdown
- Error recovery

**Example**:
```go
// Automatic retry on failure
if err != nil {
    if task.Attempts < task.MaxAttempts {
        task.Status = StatusError
        task.NextAttemptAt = nextAttemptAtFunc(task)  // Exponential backoff
        task.Error = err.Error()
    } else {
        task.Status = StatusAttemptsLeft  // No more retries
    }
    storage.UpdateTask(ctx, task)
}
```

## Architectural Principles

### 11. Separation of Concerns

**Layers**:
1. **Public API** - User-facing interface
2. **Processor** - Task processing logic
3. **Storage** - Data persistence
4. **Entity** - Domain models

**No cross-layer violations**:
- Storage doesn't know about processors
- Processors don't know about HTTP/gRPC
- Entity is pure data

### 12. Dependency Inversion

**Principle**: Depend on abstractions, not concretions

**Implementation**:
```go
// Good - Depend on interface
type GoqueProcessor struct {
    taskStorage TaskStorage  // Interface
}

// Bad - Depend on concrete type
type GoqueProcessor struct {
    taskStorage *pgTaskStorage  // Concrete
}
```

### 13. Single Responsibility

**Each component has one job**:
- `GoqueProcessor` - Process tasks
- `Healer` - Fix stuck tasks
- `Cleaner` - Remove old tasks
- `Storage` - Persist data
- `QueueMngr` - Add tasks

### 14. Open/Closed Principle

**Open for extension, closed for modification**:

```go
// Extend via options
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithWorkers(10),  // Extend configuration
)

// Extend via hooks
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithHooksBeforeProcessing(customHook),  // Extend behavior
)

// Don't need to modify GoqueProcessor code
```

## Performance Principles

### 15. Efficient Resource Usage

**Implementation**:
- Goroutine pooling (ants)
- Batch operations
- Database connection pooling
- Efficient indexes

### 16. Predictable Performance

**Implementation**:
- Configurable limits (max tasks, workers)
- No unbounded operations
- Timeout handling
- Back pressure

### 17. Scalability

**Horizontal**:
- Multiple instances can run concurrently
- Database handles coordination

**Vertical**:
- Increase worker count
- Increase batch size

## Operational Principles

### 18. Observability

**Implementation**:
- Structured logging (slog)
- Context propagation
- Error tracking
- Extensible via hooks

### 19. Debuggability

**Implementation**:
- Clear error messages
- Logging at key points
- Task status tracking
- No hidden state

### 20. Maintainability

**Implementation**:
- Clear code structure
- Comprehensive tests
- Documentation
- Consistent conventions

## Security Principles

### 21. SQL Injection Prevention

**Implementation**:
- go-jet prevents SQL injection
- No string concatenation in queries
- Parameterized queries

### 22. Safe Concurrency

**Implementation**:
- No data races (test with `-race`)
- Proper locking
- No shared mutable state without protection

## Documentation Principles

### 23. Self-Documenting Code

**Prefer**:
```go
// Good - name explains purpose
func GetTasksForProcessing(ctx context.Context, taskType string, limit int64) ([]*entity.Task, error)

// Bad - needs comment to understand
func GetTasks(ctx context.Context, t string, l int64) ([]*entity.Task, error)
```

### 24. Comment Why, Not What

```go
// Good - explains why
// Use FOR UPDATE SKIP LOCKED to prevent multiple workers
// from processing the same task concurrently
FOR(UPDATE().SKIP_LOCKED())

// Bad - obvious what
// Get tasks from database
GetTasks(...)
```

## Evolution Principles

### 25. Backward Compatibility

**Implementation**:
- Stable public API
- Database migrations
- Deprecation warnings before removal

### 26. Progressive Enhancement

**Start simple, add complexity only when needed**:
1. Basic task queue
2. Add retry logic
3. Add healer
4. Add cleaner
5. Add hooks
6. ...

Not:
1. Build complex system with all features upfront
2. Most features unused
3. Hard to understand and maintain
