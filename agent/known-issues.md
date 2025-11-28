# Known Issues

## Current Known Issues

### None Reported

No known critical issues at this time.

---

## Potential Issues & Workarounds

### 1. Database Connection Exhaustion

**Scenario**: High concurrency with many processors

**Symptoms**:
- "too many connections" error
- Slow task processing
- Connection timeouts

**Workaround**:
```go
// Configure connection pool
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(time.Hour)

// Or reduce worker count per processor
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithWorkers(5),  // Reduce from 10
)
```

**Recommendation**:
- Monitor connection usage
- Tune based on workload
- Consider connection pooler (pgbouncer)

---

### 2. Long-Running Tasks Blocking Workers

**Scenario**: Some tasks take very long to process

**Symptoms**:
- Worker pool exhausted
- Other tasks starve
- Slow task throughput

**Workaround**:
```go
// Separate processor for long tasks
goque.RegisterProcessor(
    "long_task",
    longTaskProcessor,
    WithWorkers(2),  // Few workers for long tasks
    WithTaskTimeout(5*time.Minute),  // Longer timeout
)

// Many workers for quick tasks
goque.RegisterProcessor(
    "quick_task",
    quickTaskProcessor,
    WithWorkers(50),  // Many workers
    WithTaskTimeout(30*time.Second),
)
```

**Recommendation**:
- Separate task types by duration
- Configure workers and timeout appropriately
- Consider task streaming for very long operations

---

### 3. Task Payload Size Limit

**Scenario**: Very large task payloads (>1MB)

**Symptoms**:
- Slow task insertion
- Slow task fetching
- High memory usage
- Network timeouts

**Workaround**:
```go
// Store large data externally
type TaskPayload struct {
    DataURL string  // S3/Storage URL
    Metadata map[string]string
}

// In processor
func (p *Processor) ProcessTask(ctx context.Context, task *entity.Task) error {
    var payload TaskPayload
    json.Unmarshal([]byte(task.Payload), &payload)

    // Fetch actual data from URL
    data := fetchFromStorage(payload.DataURL)

    // Process data
    // ...
}
```

**Recommendation**:
- Keep payload < 100KB
- Store large data in external storage
- Pass references in payload

---

### 4. Clock Skew in Distributed Systems

**Scenario**: Multiple instances with different system times

**Symptoms**:
- Tasks not processed when scheduled
- Healer triggers incorrectly
- Task ordering issues

**Workaround**:
```go
// Use database time instead of application time
// In query:
WHERE next_attempt_at <= NOW()  // Database NOW(), not app time

// When scheduling:
task.NextAttemptAt = time.Now().Add(delay)
// Consider using database transaction time instead
```

**Recommendation**:
- Synchronize system clocks (NTP)
- Use database time for time-sensitive operations
- Add clock skew tolerance

---

### 5. Task Ordering Not Guaranteed

**Scenario**: Need strict FIFO ordering

**Symptoms**:
- Tasks processed out of order
- Newer tasks processed before older

**Current Behavior**:
```go
// Tasks fetched by status, not strict order
WHERE status = 'new'
ORDER BY created_at ASC  // But not guaranteed across workers
```

**Workaround**:
```go
// Use single worker for strict ordering
processor := NewGoqueProcessor(
    storage,
    "ordered_task",
    processor,
    WithWorkers(1),  // Single worker = strict ordering
)

// Or use external ordering key
type TaskPayload struct {
    SequenceNumber int64
    // ...
}
```

**Recommendation**:
- Don't rely on strict ordering if possible
- Use single worker if ordering critical
- Consider external ordering mechanism

---

## Common User Errors

### 1. Forgetting to Call Run()

**Error**:
```go
goque := NewGoque(storage)
goque.RegisterProcessor("email", processor)
// Forgot to call Run()!
```

**Symptoms**:
- No tasks processed
- No errors shown
- Silent failure

**Solution**:
```go
goque := NewGoque(storage)
goque.RegisterProcessor("email", processor)
err := goque.Run(ctx)  // Don't forget!
if err != nil {
    log.Fatal(err)
}
```

---

### 2. Not Handling Stop() on Shutdown

**Error**:
```go
goque.Run(ctx)
// Application exits, workers forcefully killed
```

**Symptoms**:
- Tasks interrupted mid-processing
- Graceful shutdown not working
- Possible data corruption

**Solution**:
```go
// Setup signal handling
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan
    cancel()  // Trigger graceful shutdown
}()

goque.Run(ctx)
defer goque.Stop()  // Wait for workers to finish
```

---

### 3. Not Respecting Context in ProcessTask

**Error**:
```go
func (p *Processor) ProcessTask(ctx context.Context, task *entity.Task) error {
    // Long operation, doesn't check ctx
    time.Sleep(10 * time.Minute)
    return nil
}
```

**Symptoms**:
- Tasks don't timeout
- Graceful shutdown hangs
- Resource leaks

**Solution**:
```go
func (p *Processor) ProcessTask(ctx context.Context, task *entity.Task) error {
    // Use context-aware operations
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-time.After(10 * time.Minute):
        // Process
    }
    return nil
}
```

---

### 4. Creating Tasks with Same External ID

**Error**:
```go
// Creating multiple tasks with same external ID
for i := 0; i < 10; i++ {
    task := entity.NewTaskWithExternalID("email", payload, "order-123")
    storage.AddTask(ctx, task)  // Only first succeeds!
}
```

**Symptoms**:
- Database constraint violation
- Task creation fails
- Errors in logs

**Solution**:
```go
// Use unique external IDs
for i := 0; i < 10; i++ {
    externalID := fmt.Sprintf("order-123-item-%d", i)
    task := entity.NewTaskWithExternalID("email", payload, externalID)
    storage.AddTask(ctx, task)
}

// Or don't use external ID if not needed
task := entity.NewTask("email", payload)
storage.AddTask(ctx, task)
```

---

## Debugging Tips

### Enable Debug Logging

```go
// Set log level to debug
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
slog.SetDefault(logger)
```

### Check Task Status

```sql
-- Count tasks by status
SELECT status, COUNT(*)
FROM tasks
GROUP BY status;

-- Find stuck tasks
SELECT *
FROM tasks
WHERE status = 'processing'
  AND updated_at < NOW() - INTERVAL '5 minutes';

-- Find failed tasks
SELECT type, error, COUNT(*)
FROM tasks
WHERE status = 'attempts_left'
GROUP BY type, error;
```

### Monitor Goroutines

```go
// Add health endpoint
http.HandleFunc("/debug/goroutines", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Goroutines: %d\n", runtime.NumGoroutine())
})
```

### Use pprof

```go
import _ "net/http/pprof"

go func() {
    http.ListenAndServe(":6060", nil)
}()

// Then access:
// http://localhost:6060/debug/pprof/
```

---

## Reporting Issues

When reporting an issue, please include:

1. **Environment**:
   - Go version
   - PostgreSQL version
   - Goque version

2. **Configuration**:
   - Processor configuration
   - Worker count
   - Timeout settings

3. **Reproduction Steps**:
   - Minimal code to reproduce
   - Expected behavior
   - Actual behavior

4. **Logs**:
   - Relevant log output
   - Error messages
   - Stack traces

5. **Database State**:
   - Task counts by status
   - Example stuck tasks

## Issue Template

```markdown
**Environment**
- Go version: 1.23
- PostgreSQL version: 15.2
- Goque version: v1.0.0

**Configuration**
```go
processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithWorkers(10),
    WithMaxAttempts(3),
)
```

**Reproduction Steps**
1. Start application
2. Submit 100 tasks
3. Observe behavior

**Expected Behavior**
All tasks should be processed successfully.

**Actual Behavior**
Some tasks get stuck in 'processing' status.

**Logs**
```
ERROR: task processing failed: context deadline exceeded
```

**Database State**
```sql
SELECT status, COUNT(*) FROM tasks GROUP BY status;
-- processing: 15
-- done: 85
```

**Additional Context**
This happens only under high load (>1000 tasks/min).
```

---

## Getting Help

- Check [GitHub Issues](https://github.com/ruko1202/goque/issues)
- Search [GitHub Discussions](https://github.com/ruko1202/goque/discussions)
- Read documentation in [agent/](.)
- Review [examples](../examples) (if available)
