# =====================================
# Goque - Database Task Queue System
# =====================================
# Main Makefile for building, testing, and managing the Goque project.
#
# Common commands:
#   make all        - Run full build pipeline (deps, tools, tests, lint, format)
#   make deps       - Download and tidy Go dependencies
#   make bin-deps   - Install all required binary tools
#   make tloc       - Run all tests
#   make test-cov   - Run tests with coverage report
#   make lint       - Run linter checks
#   make fmt        - Format code with gofmt and goimports
#   make mocks      - Generate mock implementations
#
# Database commands (see scripts/database.mk):
#   make db-up      - Apply database migrations
#   make db-down    - Rollback last migration
#   make db-status  - Show migration status
#   make db-info    - Show database configuration
#   make docker-up  - Start databases with Docker Compose
# =====================================

# Include shared configuration (GOBIN, ENV_CONFIG_FILE)
include scratch.mk

# Include universal database commands
include database.mk

# -------------------------------------
# Default target
# -------------------------------------
# Runs the complete build pipeline: install dependencies, binary tools,
# run tests, format code, and run linter checks
.PHONY: all
all: deps bin-deps tloc fmt lint

# -------------------------------------
# Install deps and tools
# -------------------------------------

# deps - Download and organize Go module dependencies
# Downloads all required Go modules and removes unused ones
.PHONY: deps
deps:
	go mod download
	go mod tidy

# bin-deps-db - Install database-related binary tools
# Installs goose v3.26.0 for database migrations
.PHONY: bin-deps-db
bin-deps-db:
	GOBIN=$(GOBIN) go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

# bin-deps - Install all required binary tools
# Installs:
#   - goose: database migration tool
#   - golangci-lint: comprehensive Go linter
#   - mockgen: mock code generator for testing
.PHONY: bin-deps
bin-deps: bin-deps-db
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 && \
	GOBIN=$(GOBIN) go install go.uber.org/mock/mockgen@v0.6.0
	
# -------------------------------------
# Tests
# -------------------------------------

# tloc - Run all tests with race detection disabled
# Options:
#   -p 2: Run tests in parallel with max 2 processes
#   -count 2: Run each test 2 times to catch flaky tests
# Note: Package tests are run from project root
.PHONY: tloc
tloc:
	go test -p 2 -count 2 ./...
	#cd ./test/via_pkg/ && go test -p 2 -count 2 ./...

# test-cov - Run tests with coverage analysis
# Generates coverage report excluding mock files
# Options:
#   -race: Enable data race detection
#   -p 2: Run tests in parallel with max 2 processes
#   -count 2: Run each test 2 times
#   -coverprofile: Output file for coverage data
#   -covermode atomic: Use atomic mode for race detector compatibility
#   --coverpkg: Measure coverage for internal/ packages only
# Output files:
#   coverage.tmp: Raw coverage data
#   coverage.out: Filtered coverage data (no mocks)
#   coverage.report: Human-readable coverage report
.PHONY: test-cov
test-cov:
	go test -race -p 2 -count 2 -coverprofile=coverage.tmp -covermode atomic --coverpkg=./internal/... ./...
	@grep -vE "mock|internal/pkg/generated" coverage.tmp > coverage.out
	go tool cover -func=coverage.out | sed 's|github.com/ruko1202/goque||' | sed -E 's/\t+/\t/g' | tee coverage.report

# -------------------------------------
# Linter and formatter
# -------------------------------------

# lint - Run comprehensive linter checks
# Uses golangci-lint with configuration from .golangci.yml
# Checks for code quality, style, bugs, and best practices
.PHONY: lint
lint:
	$(info $(M) running linter...)
	@$(GOBIN)/golangci-lint run

# fmt - Format code and organize imports
# Uses gofmt for code formatting and goimports for import organization
# Automatically fixes formatting issues
.PHONY: fmt
fmt:
	$(info $(M) fmt project...)
	@# Format code with gofmt and organize imports with goimports
	@$(GOBIN)/golangci-lint fmt -E gofmt -E goimports ./...

# Database commands are defined in scripts/database.mk
# See available commands: make db-info, make db-up, make docker-up

# -------------------------------------
# Code generation
# -------------------------------------

# mocks - Generate mock implementations for testing
# Regenerates all mock files for processor interfaces
# Mocks are used for unit testing with go.uber.org/mock
# Generated files:
#   - mock_processors/queueprocessor/processor.go
#   - mock_processors/queueprocessor/task_processor.go
.PHONY: mocks
mocks:
	rm -rf ./internal/pkg/generated/mocks
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_storages/storages.go -source ./internal/storages/interface.go

