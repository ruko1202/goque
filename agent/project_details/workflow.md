# Development Workflow

## Initial Setup

### 1. Clone Repository
```bash
git clone https://github.com/ruko1202/goque.git
cd goque
```

### 2. Install Dependencies
```bash
# Install all dependencies and tools
make all

# Or step by step:
make deps        # Go modules
make bin-deps    # Development tools
```

### 3. Setup Database
```bash
# Create .env.local file
cat > .env.local << EOF
DB_DSN "postgresql://postgres:postgres@localhost:5432/goque_dev?sslmode=disable"
DB_DRIVER "postgres"
EOF

# Start PostgreSQL (using Docker)
docker run --name goque-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goque_dev \
  -p 5432:5432 \
  -d postgres:15

# Run migrations
make db-up
```

### 4. Verify Setup
```bash
# Run tests with race detector
make test-cov

# Run linter
make lint

# Should all pass ✅
```

## Development Cycle

### Daily Workflow

```
1. Pull latest changes
   ↓
2. Create feature branch
   ↓
3. Make changes
   ↓
4. Run tests locally
   ↓
5. Run linter
   ↓
6. Commit changes
   ↓
7. Push to remote
   ↓
8. Create Pull Request
   ↓
9. Address review comments
   ↓
10. Merge
```

## Git Workflow

### ⚠️ CRITICAL: Never Commit to main/master

**Always work on feature branches. Never commit directly to main or master.**

Before starting any work:
```bash
# 1. Check current branch
CURRENT_BRANCH=$(git branch --show-current)

# 2. If on main/master, create a new feature branch
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
    git checkout -b feature/your-feature-name
else
    # Continue working on current feature branch
    echo "Working on branch: $CURRENT_BRANCH"
fi

# 3. Verify you're NOT on main/master
git branch --show-current  # Should NOT be 'main' or 'master'
```

**Branch Strategy**:
- **If currently on main/master** → Create new feature branch for your work
- **If currently on feature branch** → Continue working on that branch
- **Exception**: If user explicitly requests a new branch, create it regardless of current branch

### Branch Naming

**Quick Rules**:
- Maximum 3 words after prefix
- Use hyphens, lowercase only
- Format: `<type>/<word1>-<word2>-<word3>`

**Examples**:
```bash
# Good (1-3 words)
git checkout -b feature/add-priority           # 2 words ✅
git checkout -b fix/race-condition             # 2 words ✅
git checkout -b docs/update-readme             # 2 words ✅

# Bad (more than 3 words)
git checkout -b feature/add-priority-queue-system  # 4 words ❌
git checkout -b docs/add-architecture-guide        # 3 words but could be shorter
```

**See [Branch Naming Guide](branch-naming.md) for:**
- Complete rules and conventions
- Examples for all branch types
- Strategies for shortening long names
- Common mistakes to avoid

### Commit Flow

```bash
# 0. FIRST: Verify you're NOT on main/master
git branch --show-current  # Must be a feature branch!

# 1. Check status
git status

# 2. Run tests with race detector
make test-cov

# 3. Run linter
make lint

# 4. Format code
make fmt

# 5. Stage changes
git add .

# 6. CRITICAL: Check for sensitive information
git diff --cached | grep -i "password\|token\|secret\|key\|api_key"

# Scan for common patterns
git diff --cached | grep -E "(sk_live_|pk_live_|Bearer |Authorization:)"

# Look for emails, phone numbers, personal data
git diff --cached | grep -E "([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}|\+?[0-9]{1,3}[-.]?[0-9]{3}[-.]?[0-9]{3}[-.]?[0-9]{4})"

# If found, replace with placeholders like "<your-secret>" or "<user-data>"

# 7. Commit with meaningful message
git commit -m "feat: add priority queue support

- Add priority field to task entity
- Update database schema with migration
- Implement priority-based task fetching
- Add tests for priority ordering

Closes #123"

# 8. Push
git push origin feature/add-priority-queue
```

### Pull Request Flow

```bash
# 1. Create PR via GitHub UI or gh CLI
gh pr create --title "Add priority queue support" \
  --body "## Summary
- Adds priority field to tasks
- Tasks with higher priority processed first
- Backward compatible

## Changes
- New migration: 003_add_priority.sql
- Updated Task entity
- New tests

## Testing
- All tests pass
- Tested with 1000+ tasks

Closes #123"

# 2. Wait for CI checks
# 3. Address review comments
# 4. Get approval
# 5. Merge (squash and merge preferred)
```

## Making Changes

### Adding New Feature

1. **Design First**
   ```
   - What problem does it solve?
   - How does it fit into architecture?
   - What's the API?
   - What tests are needed?
   ```

2. **Create Issue**
   - Describe feature
   - Discuss approach
   - Get feedback

3. **Create Branch**
   ```bash
   git checkout -b feature/task-priority
   ```

4. **Implement**
   - Write failing test first (TDD)
   - Implement feature
   - Make test pass
   - Refactor

5. **Test**
   ```bash
   make test-cov  # Runs tests with race detector
   make lint
   ```

6. **Document** (CRITICAL - Do not skip!)
   - **Always** update README if:
     - Public API changed
     - New feature added
     - Usage examples affected
   - **Always** add/update GoDoc comments for exported symbols
   - **Always** update agent/project_details/ docs if:
     - Architecture changed (architecture.md)
     - New pattern introduced (conventions.md)
     - New principle applied (principles.md)
     - Critical rule added/changed (critical-rules.md)
     - Workflow changed (workflow.md)
     - Any agreement or convention changed
   - **Always** update agent/quick_load/goque.toon if:
     - Critical rules, principles, architecture, or workflows change
     - Keep it synchronized with agent/project_details/
   - **When architecture, agreements, or known issues change**:
     - Review for contradictions between files
     - If contradictions found, keep the broader interpretation
     - Remove duplicate or contradicting rules
   - **Always** update agent/project_details/known-issues.md:
     - **Remove ALL resolved issues** - NO "Past Issues" section allowed
     - Add new issues found during development
     - Keep only CURRENT active issues
     - Resolved issues are tracked in git history
   - Add migration guide for breaking changes
   - Update examples if API changed

7. **Commit & PR**
   - **First commit: code changes**
   - **Second commit: documentation changes**
   - Follow commit conventions
   - Create PR with detailed description
   - Documentation commit should follow code commit immediately

### Fixing Bug

1. **Reproduce Bug**
   - Write test that fails
   - Confirms the bug exists

2. **Fix**
   - Implement fix
   - Test passes

3. **Prevent Regression**
   - Ensure test remains
   - Consider edge cases

4. **Commit**
   ```bash
   git commit -m "fix: prevent duplicate task processing

   - Add FOR UPDATE SKIP LOCKED to query
   - Add test for concurrent processing
   - Fixes #456"
   ```

### Refactoring

1. **Ensure Tests Exist**
   - Tests should pass before refactoring
   - Add tests if missing

2. **Refactor**
   - Make changes
   - Keep tests passing

3. **Verify**
   ```bash
   make test-cov  # Tests still pass with race detector
   make lint      # Code quality maintained
   ```

4. **Commit**
   ```bash
   git commit -m "refactor: simplify task processor options

   - Consolidate option types
   - Improve API clarity
   - No behavior changes"
   ```

## Database Changes

### Setup Database with Docker

**Always use Docker for local PostgreSQL** to ensure consistency:

```bash
# Start PostgreSQL container
docker run --name goque-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goque_dev \
  -p 5432:5432 \
  -d postgres:15

# Verify it's running
docker ps | grep goque-postgres

# Stop container (data persists)
docker stop goque-postgres

# Start existing container
docker start goque-postgres

# Remove container (deletes data!)
docker rm -f goque-postgres

# View logs
docker logs goque-postgres

# Connect to database
docker exec -it goque-postgres psql -U postgres -d goque_dev
```

**Container management tips**:
- Use same container name (`goque-postgres`) for consistency
- Data persists in Docker volume even after stop
- Use `docker rm -f` only when you want fresh database
- Configure `.env.local` to point to Docker database

### Working with Migrations (goose)

**All migrations use goose via make commands**:

```bash
# Check migration status
make db-status

# Apply all pending migrations
make db-up

# Rollback last migration
make db-down

# Create new migration
make db-migrate-create name="add_priority_field"

# After schema changes, regenerate models
make db-models
```

**IMPORTANT**: Never use goose directly, always use make commands.

### Adding Migration

```bash
# 1. Ensure PostgreSQL is running in Docker
docker start goque-postgres

# 2. Create migration
make db-migrate-create name="add_priority_field"

# 3. Edit migration file (migrations/xxx_add_priority_field.sql)
cat > migrations/xxx_add_priority_field.sql << EOF
-- +goose Up
ALTER TABLE tasks ADD COLUMN priority INT NOT NULL DEFAULT 0;
CREATE INDEX idx_tasks_priority ON tasks(priority DESC, created_at ASC);

-- +goose Down
DROP INDEX idx_tasks_priority;
ALTER TABLE tasks DROP COLUMN priority;
EOF

# 4. Test migration
make db-up     # Apply
make db-status # Verify
make db-down   # Test rollback
make db-up     # Re-apply

# 5. Regenerate models (CRITICAL!)
make db-models

# 6. Update code to use new field
# Edit internal/storages/sql/pg/task/...

# 7. Write tests
# Test with new field

# 8. Commit everything
git add migrations/xxx_add_priority_field.sql
git add internal/pkg/generated/postgres/
git commit -m "feat: add task priority field"
```

### Migration Best Practices

1. **Always test both up and down migrations**
   ```bash
   make db-up && make db-down && make db-up
   ```

2. **Use transactions when possible**
   ```sql
   -- +goose Up
   -- +goose StatementBegin
   ALTER TABLE tasks ADD COLUMN priority INT;
   UPDATE tasks SET priority = 0 WHERE priority IS NULL;
   ALTER TABLE tasks ALTER COLUMN priority SET NOT NULL;
   -- +goose StatementEnd
   ```

3. **Make migrations backward compatible when possible**
   - Add columns as nullable first, then make NOT NULL
   - Don't drop columns if old code still running
   - Use feature flags for schema-dependent features

4. **Document complex migrations**
   ```sql
   -- +goose Up
   -- This migration adds priority field for task ordering
   -- Default priority is 0 (normal priority)
   -- Higher values = higher priority
   ALTER TABLE tasks ADD COLUMN priority INT NOT NULL DEFAULT 0;
   ```

## Testing Workflow

### Running Tests

```bash
# All tests with coverage and race detector
make test-cov

# Or run specific tests
go test ./internal/processors/queueprocessor/

# Specific package
go test ./internal/processors/queueprocessor/

# Specific test
go test ./internal/processors/queueprocessor/ -run TestProcessTask

# With race detector
go test -race ./...

# Verbose
go test -v ./...

# With timeout
go test -timeout 30s ./...
```

### Writing Tests

```bash
# 1. Create test file next to code
# add_task.go -> add_task_test.go

# 2. Write test
# internal/storages/sql/pg/task/add_task_test.go

# 3. Run test
go test ./internal/storages/sql/pg/task/ -run TestAddTask

# 4. Fix until passing
# Make changes, rerun test

# 5. Run all tests to ensure no regressions
make test-cov
```

### Test-Driven Development (TDD)

```
1. Write failing test
   ↓
2. Run test (should fail)
   ↓
3. Write minimal code to pass
   ↓
4. Run test (should pass)
   ↓
5. Refactor
   ↓
6. Run test (should still pass)
   ↓
7. Repeat
```

## Code Quality

### Linting

```bash
# Run all linters
make lint

# Auto-fix if possible
make fmt

# Common issues:
# - Unused variables: remove or use
# - Missing error checks: add error handling
# - Golint warnings: fix naming/comments
# - Gosec warnings: security issues, fix immediately
```

### Formatting

```bash
# Format all code
make fmt

# This runs:
# - gofmt: standard formatting
# - goimports: organize imports
```

### Code Review Checklist

Before requesting review:

- [ ] Tests pass with race detector (`make test-cov`)
- [ ] Linter passes (`make lint`)
- [ ] Code formatted (`make fmt`)
- [ ] All errors handled
- [ ] All exported symbols documented
- [ ] **No sensitive information** (secrets, tokens, user personal data)
- [ ] No breaking changes (or discussed)
- [ ] Database migrations tested (up and down)
- [ ] Generated code updated
- [ ] **agent/known-issues.md clean** (CRITICAL - check every commit):
  - [ ] NO "Past Issues" section
  - [ ] ALL resolved issues removed
  - [ ] Only current issues remain
- [ ] Commit messages follow conventions
- [ ] PR description is clear

## Release Process

### Version Numbering

Follow [Semantic Versioning](https://semver.org/):
- **Major** (v2.0.0): Breaking changes
- **Minor** (v1.1.0): New features, backward compatible
- **Patch** (v1.0.1): Bug fixes

### Creating Release

```bash
# 1. Ensure main is clean
git checkout main
git pull

# 2. Run full test suite
make test-cov
make lint

# 3. Update version
# Edit relevant files if needed

# 4. Create tag
git tag -a v1.1.0 -m "Release v1.1.0

Features:
- Add task priority support
- Add webhook processor example

Bug Fixes:
- Fix race condition in worker pool
- Prevent duplicate task processing

Breaking Changes:
- None
"

# 5. Push tag
git push origin v1.1.0

# 6. Create GitHub Release
gh release create v1.1.0 \
  --title "v1.1.0" \
  --notes "See tag message"

# 7. Announce release
# Update CHANGELOG.md
# Notify users
```

## Continuous Integration

### GitHub Actions

CI runs automatically on:
- Push to any branch
- Pull request creation
- Pull request updates

CI checks:
- Tests with race detector (`make test-cov`)
- Linter (`make lint`)
- Code coverage
- Build

### Monitoring CI

```bash
# Check status
gh pr checks

# View logs
gh run view

# Re-run failed checks
gh run rerun <run-id>
```

## Local Development Tips

### Quick Test Cycle

```bash
# Terminal 1: Watch mode (requires entr or similar)
ls **/*.go | entr -c go test ./...

# Make change -> test runs automatically
```

### Debugging

```go
// Use slog for debugging
slog.Debug("processing task",
    slog.String("task_id", task.ID.String()),
    slog.Any("task", task))

// Use debugger (delve)
dlv test ./internal/processors/queueprocessor/ -- -test.run TestProcessTask
```

### Database Inspection

```bash
# Connect to database
psql $DB_DSN

# Check migrations status
make db-status

# View tasks
psql $DB_DSN -c "SELECT * FROM tasks LIMIT 10;"
```

## Troubleshooting

### Tests Fail

1. Check if database is running
2. Check if migrations are applied (`make db-status`)
3. Check if generated code is up to date (`make db-models`)
4. Run with verbose: `go test -v ./...`

### Linter Errors

1. Run `make fmt` first
2. Fix errors one by one
3. Google specific linter rules if unclear
4. Ask for help if stuck

### Race Conditions

1. Run with race detector: `go test -race ./...`
2. Fix races immediately
3. Add proper locking or synchronization
4. Re-run until clean

### Database Issues

1. Check connection string in `.env.local`
2. Ensure PostgreSQL is running
3. Check migration status: `make db-status`
4. Try rolling back and reapplying: `make db-down && make db-up`
