# Tech Stack

## Core Technologies

### Language
- **Go 1.23+**
  - Modern Go features
  - Generics support
  - Context-aware concurrency

### Database
Goque supports multiple database backends:

- **PostgreSQL 12+** (Primary)
  - ACID guarantees
  - Row-level locking (FOR UPDATE SKIP LOCKED)
  - JSONB support for task payloads
  - Advanced indexing
  - Best for production deployments

- **MySQL 8.0+** (Alternative)
  - ACID guarantees
  - Row-level locking (FOR UPDATE)
  - JSON column type for task payloads
  - Standard indexing
  - Good for MySQL-based infrastructure

- **SQLite 3+** (Embedded)
  - ACID guarantees
  - Transaction-based locking
  - JSON support via JSON1 extension
  - Embedded database (no separate server)
  - Ideal for development, testing, and small deployments

## Key Dependencies

### Database & SQL

#### go-jet v2.14.0
```go
github.com/go-jet/jet/v2
```
- **Purpose**: Type-safe SQL query builder
- **Usage**: All database queries (PostgreSQL and MySQL)
- **Benefits**:
  - Compile-time query validation
  - IDE auto-completion
  - Refactoring support
  - No string-based queries
  - Multi-database support (postgres and mysql packages)

#### lib/pq v1.10.9
```go
github.com/lib/pq
```
- **Purpose**: PostgreSQL driver
- **Usage**: PostgreSQL database connection
- **Benefits**: Pure Go implementation

#### go-sql-driver/mysql v1.8.1
```go
github.com/go-sql-driver/mysql
```
- **Purpose**: MySQL driver
- **Usage**: MySQL database connection
- **Benefits**: Pure Go implementation, full MySQL protocol support

#### mattn/go-sqlite3
```go
github.com/mattn/go-sqlite3
```
- **Purpose**: SQLite driver
- **Usage**: SQLite database connection
- **Benefits**:
  - Pure Go implementation
  - Embedded database (no separate server)
  - Zero configuration
  - Perfect for development and testing

#### sqlx v1.4.0
```go
github.com/jmoiron/sqlx
```
- **Purpose**: SQL extensions
- **Usage**: Named queries, struct scanning
- **Benefits**: Convenience over database/sql

### Concurrency

#### ants v2.11.3
```go
github.com/panjf2000/ants/v2
```
- **Purpose**: Goroutine pool management
- **Usage**: Worker pools in processors
- **Benefits**:
  - Memory efficient
  - Configurable pool size
  - Automatic goroutine reuse
  - Panic recovery

### Utilities

#### uuid
```go
github.com/google/uuid
```
- **Purpose**: UUID generation
- **Usage**: Task IDs, unique identifiers
- **Benefits**: RFC 4122 compliant

#### lo v1.52.0
```go
github.com/samber/lo
```
- **Purpose**: Functional programming utilities
- **Usage**: Collection operations, transformations
- **Benefits**: Clean, functional code style

### Configuration

#### viper v1.21.0
```go
github.com/spf13/viper
```
- **Purpose**: Configuration management
- **Usage**: Load environment variables, config files
- **Benefits**:
  - Multiple config sources
  - Environment variable support
  - Hot reloading

#### pflag v1.0.10
```go
github.com/spf13/pflag
```
- **Purpose**: Command-line flags
- **Usage**: CLI tools, configuration
- **Benefits**: POSIX/GNU-style flags

### Observability

#### xlog
```go
github.com/ruko1202/xlog
```
- **Purpose**: Unified structured logging interface
- **Usage**: All internal logging in Goque
- **Components**:
  - Zap adapter (recommended for production)
  - Slog adapter (Go 1.21+ standard library)
  - Custom adapter support
- **Features**:
  - Context-aware logging
  - Automatic logger propagation
  - Structured fields (task_id, task_type, etc.)
  - Multiple backend support
- **Benefits**:
  - Consistent logging interface
  - Easy to switch backends
  - Production-ready adapters

#### prometheus/client_golang
```go
github.com/prometheus/client_golang
```
- **Purpose**: Prometheus metrics instrumentation
- **Usage**: Built-in task queue metrics
- **Components**:
  - Counters for processed tasks
  - Histograms for processing duration
  - Error tracking with labels
  - Payload size monitoring
- **Benefits**:
  - Standard metrics format
  - Ready for Prometheus scraping
  - Grafana dashboard compatible

#### OpenTelemetry
```go
go.opentelemetry.io/otel
go.opentelemetry.io/otel/trace
```
- **Purpose**: Distributed tracing
- **Usage**: Optional tracing support
- **Default**: Noop tracer (zero overhead)
- **Configuration**: `goque.SetTracerProvider(tracerProvider)`
- **Traced Operations**:
  - Task processing loop (`queue_processor.fetchAndProcess`)
  - Task fetching (`queue_processor.fetchTasks`)
  - Hook execution (before/after)
  - Queue operations (`task_queue_manager.AddTaskToQueue`)
- **Benefits**:
  - Zero overhead by default
  - Full OpenTelemetry compatibility
  - Sampling support for production
  - Integration with Jaeger, Zipkin, etc.

### Testing

#### testify v1.11.1
```go
github.com/stretchr/testify
```
- **Purpose**: Testing toolkit
- **Usage**: All tests
- **Components**:
  - `assert` - Assertions
  - `require` - Assertions with stop
  - `suite` - Test suites
  - `mock` - Manual mocks

#### gomock v0.6.0
```go
go.uber.org/mock
```
- **Purpose**: Mock generation
- **Usage**: Generate mocks for interfaces
- **Benefits**:
  - Type-safe mocks
  - IDE support
  - Compile-time safety

## Development Tools

### Database Migrations

#### goose v3.26.0
```bash
github.com/pressly/goose/v3
```
- **Purpose**: Database migrations
- **Usage**: Schema versioning
- **Commands**:
  - `goose up` - Apply migrations
  - `goose down` - Rollback migrations
  - `goose create` - Create migration
  - `goose status` - Check status

### Code Quality

#### golangci-lint v2.3.0
```bash
github.com/golangci/golangci-lint/v2
```
- **Purpose**: Go linters aggregator
- **Usage**: Code quality checks
- **Linters Used**:
  - `gofmt` - Formatting
  - `goimports` - Import organization
  - `govet` - Suspicious code
  - `staticcheck` - Static analysis
  - `gosec` - Security issues
  - Many more (see `.golangci.yml`)

### Code Generation

#### go-jet codegen
- **Purpose**: Generate database models
- **Usage**: `make db-models`
- **Output**: Type-safe table and model definitions

#### mockgen
- **Purpose**: Generate mocks
- **Usage**: `make mocks`
- **Output**: Mock implementations of interfaces

## Build & Development

### Make
- **Purpose**: Build automation
- **Usage**: All development tasks
- **Key Targets**:
  - `make all` - Full setup
  - `make deps` - Install dependencies
  - `make bin-deps` - Install tools
  - `make test-cov` - Run tests with race detector
  - `make lint` - Run linter
  - `make fmt` - Format code
  - `make db-up` - Apply migrations
  - `make mocks` - Generate mocks

### Git
- **Version Control**: Git
- **Hosting**: GitHub
- **CI/CD**: GitHub Actions
- **Workflow**: Feature branches + PR

## CI/CD

### GitHub Actions
- **Config**: `.github/workflows/ci.yml`
- **Runs**:
  - Tests
  - Linters
  - Coverage
  - Build

### Coverage
- **Tool**: Go built-in coverage
- **Target**: 84%+ (current)
- **Reporting**: Badge in README

## Runtime Dependencies

### Required
- **Go Runtime**: 1.23+
- **Database** (one of):
  - PostgreSQL 12+ (production recommended)
  - MySQL 8.0+ (alternative)
  - SQLite 3+ (development/testing/embedded)

### Optional
- **Docker**: For local PostgreSQL and MySQL
- **Make**: For build automation

## Package Structure

### Public Packages (`pkg/`)
```
pkg/
└── entity/
    └── task.go          # Task entity
```
- **Purpose**: Public API
- **Usage**: Imported by users
- **Stability**: Stable API

### Internal Packages (`internal/`)
```
internal/
├── processors/          # Task processors
├── queuemngr/          # Queue management
├── storages/           # Storage implementations
└── pkg/
    └── generated/      # Generated code
```
- **Purpose**: Internal implementation
- **Usage**: Not importable by users
- **Stability**: Can change freely

## Version Management

### Go Modules
- **File**: `go.mod`
- **Go Version**: 1.23.0
- **Toolchain**: go1.24.9

### Semantic Versioning
- **Major**: Breaking changes
- **Minor**: New features
- **Patch**: Bug fixes

## Performance Considerations

### Database
- **Connection Pooling**: Recommended in production
- **Prepared Statements**: go-jet generates these
- **Indexes**: Optimized for task queries

### Concurrency
- **Worker Pools**: Limit goroutines
- **Batch Operations**: Fetch multiple tasks
- **Context Cancellation**: Proper cleanup

### Memory
- **Goroutine Pooling**: ants prevents goroutine explosion
- **Efficient Queries**: Fetch only needed data
- **JSONB**: Efficient payload storage

## Security

### Database
- **SQL Injection**: Prevented by go-jet
- **Connection Security**: SSL support via lib/pq

### Concurrency
- **Race Conditions**: Prevented by proper locking
- **Data Races**: Tested with `-race` flag

## Observability

### Logging (xlog)
- **Package**: `github.com/ruko1202/xlog`
- **Adapters**:
  - Zap (`xlog.NewZapAdapter()`) - Production recommended
  - Slog (`xlog.NewSlogAdapter()`) - Standard library
  - Custom adapters - Implement `xlog.Logger` interface
- **Format**: Structured logging with fields
- **Context**: Context-aware logging with automatic propagation
- **Usage**:
  ```go
  logger := xlog.NewZapAdapter(zap.Must(zap.NewProduction()))
  xlog.ReplaceGlobalLogger(logger)
  ctx = xlog.ContextWithLogger(ctx, logger)
  ```
- **What Gets Logged**:
  - Task lifecycle events (creation, processing, completion)
  - Error events and retry attempts
  - Database operations and transactions
  - Worker pool events
  - Healer and cleaner operations

### Metrics (Prometheus)
- **Built-in**: Prometheus metrics via `internal/metrics`
- **Metrics Exposed**:
  - `goque_processed_tasks_total` - Task counters by type and status
  - `goque_processed_tasks_with_error_total` - Error counters with details
  - `goque_task_processing_duration_seconds` - Processing time histograms
  - `goque_task_payload_size_bytes` - Payload size histograms
- **Labels**: `task_type`, `status`, `task_error_type`, `task_processing_operations`, `service`
- **Operations Tracked**: add_to_queue, processing, cleanup, health
- **Configuration**:
  ```go
  goque.SetMetricsServiceName("my-service")
  ```
- **Extensible**: Via hooks for custom metrics

### Tracing (OpenTelemetry)
- **Package**: `go.opentelemetry.io/otel`
- **Default**: Noop tracer (zero overhead)
- **Configuration**:
  ```go
  // Initialize TracerProvider (before creating Goque instances)
  tracerProvider := sdktrace.NewTracerProvider(
      sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.01)), // 1% sampling
      sdktrace.WithBatcher(exporter),
  )
  goque.SetTracerProvider(tracerProvider)
  ```
- **Traced Operations**:
  - `queue_processor.fetchAndProcess` - Main processing loop
  - `queue_processor.fetchTasks` - Getting tasks from DB
  - `queue_processor.callHooksBefore` - Pre-processing hooks
  - `queue_processor.processTask` - Task execution
  - `queue_processor.callHooksAfter` - Post-processing hooks
  - `task_queue_manager.AddTaskToQueue` - Task creation
  - `task_queue_manager.GetTasks` - Task retrieval
- **Performance Impact**:
  - Noop (default): 0% overhead
  - 1% sampling: <0.1% memory overhead
  - 100% sampling: ~6% memory overhead
- **Sampling Strategies**:
  - Production: 1% (`TraceIDRatioBased(0.01)`)
  - Staging: 10% (`TraceIDRatioBased(0.1)`)
  - Development: 100% (`AlwaysSample()`)
- **Integration**: Works with Jaeger, Zipkin, and other OTEL-compatible backends

## Testing Strategy

### Unit Tests
- **Coverage**: High coverage required
- **Mocking**: gomock for interfaces
- **Assertions**: testify

### Integration Tests
- **Database**: Real PostgreSQL in tests
- **Isolation**: Test database per run
- **Cleanup**: Automatic cleanup

### Test Execution
```bash
# Run all tests with coverage and race detector
make test-cov

# Run specific tests
go test ./internal/processors/queueprocessor/

# Run with race detector
go test -race ./...
```

## Documentation

### Code Documentation
- **Format**: GoDoc comments
- **Standard**: Exported symbols documented

### External Documentation
- **README.md**: User guide
- **agent/**: Agent documentation
