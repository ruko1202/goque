GOBIN			 	?= $(PWD)/bin
MIGRATIONS_DIR   	?=migrations
ENV_CONFIG_FILE 	?= .env.local
ifndef STORAGE_MIGRATION_DSN
STORAGE_MIGRATION_DSN=$(shell cat $(ENV_CONFIG_FILE) | grep STORAGE_DSN | awk '{print $$2}' | sed 's/"//g')
endif


ifeq ($(wildcard ${GOBIN}),)
	mkdir -p ${GOBIN}
endif

.PHONY: all
all: deps bin-deps test fmt lint


# -------------------------------------
# Install deps and tools
# -------------------------------------
.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: bin-deps
bin-deps:
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0 && \
	GOBIN=$(GOBIN) go install go.uber.org/mock/mockgen@v0.6.0 && \
	GOBIN=$(GOBIN) go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

# -------------------------------------
# Tests
# -------------------------------------
.PHONY: test
test:
	go test -race -p 2 -count 2 ./...

.PHONY: ci-test-with-coverage
ci-test-with-coverage:
	go test -race -coverprofile=coverage.tmp -covermode atomic --coverpkg=./internal/... ./...
	@grep -v "mock" coverage.tmp > coverage.out
	go tool cover -func=coverage.out


# -------------------------------------
# Linter and formatter
# -------------------------------------
.PHONY: lint
lint:
	#$(info $(M) running linter...)
	@$(GOBIN)/golangci-lint run

.PHONY: fmt
fmt:
	$(info $(M) fmt project...)
	@# Format code with gofmt and organize imports with goimports
	@$(GOBIN)/golangci-lint fmt -E gofmt -E goimports ./...

# -------------------------------------
# Database
# -------------------------------------
.PHONY: db-migrate-create

# -------------------------------------
# Local dev commands
# -------------------------------------
.PHONY: mocks
mocks:
	rm -rf ./internal/pkg/generated/mocks
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_processor/goque_processor.go -source ./internal/processor/goque_processor.go
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_processor/task_processor.go -source ./internal/processor/task_processor.go
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_queue_mngr/queue_mngr.go -source ./internal/queue_mngr/queue.go

