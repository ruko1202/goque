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

### Logging
- **Package**: `log/slog` (standard library)
- **Format**: Structured logging
- **Context**: Context-aware logging

### Metrics
- Not built-in, but extensible via hooks

### Tracing
- Not built-in, but extensible via context

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
