GOBIN			 	?= $(PWD)/bin
MIGRATIONS_DIR   	?=migrations
ENV_CONFIG_FILE 	?= .env.local
ifndef DB_MIGRATION_DSN
DB_MIGRATION_DSN=$(shell cat $(ENV_CONFIG_FILE) | grep DB_DSN | awk '{print $$2}' | sed 's/"//g')
DB_DRIVER=$(shell cat $(ENV_CONFIG_FILE) | grep DB_DRIVER | awk '{print $$2}' | sed 's/"//g')
endif

$(shell mkdir -p $(GOBIN))

.PHONY: all
all: deps bin-deps tloc fmt lint

# -------------------------------------
# Install deps and tools
# -------------------------------------
.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: bin-deps-db
bin-deps-db:
	GOBIN=$(GOBIN) go install github.com/pressly/goose/v3/cmd/goose@v3.26.0

.PHONY: bin-deps
bin-deps: bin-deps-db
	GOBIN=$(GOBIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.3.0 && \
	GOBIN=$(GOBIN) go install go.uber.org/mock/mockgen@v0.6.0
	
# -------------------------------------
# Tests
# -------------------------------------
.PHONY: tloc
tloc:
	go test -p 2 -count 2 ./...
	#cd ./test/via_pkg/ && go test -p 2 -count 2 ./...


.PHONY: test-cov
test-cov:
	go test -race -p 2 -count 2 -coverprofile=coverage.tmp -covermode atomic --coverpkg=./internal/... ./...
	@grep -v "mock" coverage.tmp > coverage.out
	go tool cover -func=coverage.out | sed 's|github.com/ruko1202/goque||' | sed -E 's/\t+/\t/g' | tee coverage.report

# -------------------------------------
# Linter and formatter
# -------------------------------------
.PHONY: lint
lint:
	$(info $(M) running linter...)
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
db-migrate-create: name=
db-migrate-create: ## call `make install-db-tools` before
	$(info $(M) creating DB migration...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) postgres $(DB_MIGRATION_DSN) create "${name}" sql

.PHONY: db-status
db-status: ## call `make install-db-tools` before
	$(info $(M) check DB migrations status...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_MIGRATION_DSN) status

.PHONY: db-up
db-up: ## call `make install-db-tools` before
	$(info $(M) starting DB migration up...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_MIGRATION_DSN) up
	make db-models
	make db-status

.PHONY: db-down
db-down: ## call `make install-db-tools` before
	$(info $(M) starting DB migration down...)
	$(GOBIN)/goose -dir $(MIGRATIONS_DIR) $(DB_DRIVER) $(DB_MIGRATION_DSN) down
	make db-status

.PHONY: db-models
db-models:
	go run ./scripts/dbmodels/generate.go --dsn=$(DB_MIGRATION_DSN) --dest="internal/pkg/generated/"

# -------------------------------------
# Local dev commands
# -------------------------------------
.PHONY: mocks
mocks:
	rm -rf ./internal/pkg/generated/mocks
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_processors/queueprocessor/processor.go -source ./internal/processors/queueprocessor/processor.go
	$(GOBIN)/mockgen -typed -destination ./internal/pkg/generated/mocks/mock_processors/queueprocessor/task_processor.go -source ./internal/processors/queueprocessor/task_processor.go

