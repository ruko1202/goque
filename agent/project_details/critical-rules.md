# Critical Rules

These are **non-negotiable** rules that must **always** be followed when working on Goque.

## 🚨 MUST Follow Rules

### 1. Never Break Public API

**Rule**: Do not break backward compatibility in public API (`goque.go`, `pkg/`)

**Rationale**: Users depend on stable API

**Examples**:

❌ **WRONG**:
```go
// Breaking change - renamed method
func (g *Goque) StartProcessors(ctx context.Context) error  // Was: Run()

// Breaking change - changed signature
func NewGoque(storage TaskStorage, logger *slog.Logger) *Goque  // Added parameter
```

✅ **RIGHT**:
```go
// Add new method, keep old one
func (g *Goque) Run(ctx context.Context) error  // Keep existing

// Add option instead of parameter
func NewGoque(storage TaskStorage, opts ...GoqueOption) *Goque
```

**If you must break API**:
1. Discuss with team first
2. Bump major version
3. Document migration path

---

### 2. Always Use go-jet for SQL Queries

**Rule**: Never write raw SQL strings, always use go-jet

**Rationale**: Prevent SQL injection, type safety

❌ **WRONG**:
```go
query := fmt.Sprintf("SELECT * FROM tasks WHERE type = '%s'", taskType)
db.Query(query)
```

✅ **RIGHT**:
```go
query := Task.
    SELECT(Task.AllColumns).
    WHERE(Task.Type.EQ(String(taskType)))
```

**No exceptions**.

---

### 3. Context Must Be First Parameter

**Rule**: All functions that can block or perform I/O must accept `context.Context` as first parameter

**Rationale**: Enable cancellation, timeout, tracing

❌ **WRONG**:
```go
func ProcessTask(task *entity.Task) error
func AddTask(task *entity.Task, ctx context.Context) error  // Wrong position
```

✅ **RIGHT**:
```go
func ProcessTask(ctx context.Context, task *entity.Task) error
```

**No exceptions**.

---

### 4. Always Handle Errors

**Rule**: Never ignore errors, always handle them properly

❌ **WRONG**:
```go
storage.AddTask(ctx, task)  // Ignoring error
_ = storage.AddTask(ctx, task)  // Explicitly ignoring
```

✅ **RIGHT**:
```go
err := storage.AddTask(ctx, task)
if err != nil {
    return fmt.Errorf("failed to add task: %w", err)
}
```

**No exceptions**.

---

### 5. Wrap Errors with Context

**Rule**: Always wrap errors with context using `%w`

**Rationale**: Error chain for debugging, `errors.Is/As` support

❌ **WRONG**:
```go
if err != nil {
    return errors.New("failed to add task")  // Lost original error
}
if err != nil {
    return fmt.Errorf("failed to add task: %v", err)  // Not wrapped
}
```

✅ **RIGHT**:
```go
if err != nil {
    return fmt.Errorf("failed to add task: %w", err)  // Wrapped
}
```

---

### 6. Run Tests Before Committing

**Rule**: Always run `make test-cov` before committing

**Rationale**: Prevent broken builds, catch regressions, detect race conditions

```bash
make test-cov  # Must pass (includes -race flag)
```

**No exceptions**.

---

### 7. Run Linter Before Committing

**Rule**: Always run `make lint` before committing

**Rationale**: Code quality, consistency

```bash
make lint  # Must pass with no errors
```

**No exceptions**.

---

### 8. No Goroutine Leaks

**Rule**: Every started goroutine must be stoppable

**Rationale**: Prevent resource leaks, enable graceful shutdown

❌ **WRONG**:
```go
go func() {
    for {
        processTask()  // No way to stop!
        time.Sleep(time.Second)
    }
}()
```

✅ **RIGHT**:
```go
go func() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return  // Stoppable
        case <-ticker.C:
            processTask()
        }
    }
}()
```

---

### 9. Use Worker Pools for Concurrent Processing

**Rule**: Never spawn unlimited goroutines, always use worker pools

**Rationale**: Prevent goroutine explosion, control resources

❌ **WRONG**:
```go
for _, task := range tasks {
    go processTask(task)  // Unbounded goroutines!
}
```

✅ **RIGHT**:
```go
pool, _ := ants.NewPool(10)
defer pool.Release()

for _, task := range tasks {
    pool.Submit(func() {
        processTask(task)
    })
}
```

---

### 10. Database Migrations Must Be Reversible

**Rule**: Every `up` migration must have a corresponding `down` migration. Always use goose via make commands and Docker for database.

**Rationale**: Enable rollback, safety net, consistent tooling

**Tools**:
- **goose** - Migration tool (via `make` commands)
- **Docker** - For running PostgreSQL locally

**Migration workflow**:
```bash
# 1. Start PostgreSQL with Docker
docker run --name goque-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=goque_dev \
  -p 5432:5432 \
  -d postgres:15

# 2. Create migration
make db-migrate-create name="add_priority_field"

# 3. Apply migrations
make db-up

# 4. Check status
make db-status

# 5. Regenerate models after schema changes
make db-models

# 6. Rollback if needed
make db-down

```

❌ **WRONG**:
```sql
-- 001_add_column.sql

-- +goose Up
ALTER TABLE tasks ADD COLUMN priority INT;

-- +goose Down
-- Nothing here!  ❌
```

✅ **RIGHT**:
```sql
-- +goose Up
ALTER TABLE tasks ADD COLUMN priority INT;

-- +goose Down
ALTER TABLE tasks DROP COLUMN priority;
```

**No exceptions** - Always provide rollback path.

---

### 11. Regenerate Database Models After Schema Changes

**Rule**: After changing schema, always run `make db-models`

**Rationale**: Keep generated code in sync

```bash
# After migration
make db-up
make db-models  # Must run this!
```

---

### 12. Use FOR UPDATE SKIP LOCKED

**Rule**: Task fetching must use `FOR UPDATE SKIP LOCKED`

**Rationale**: Prevent duplicate processing, handle concurrency

❌ **WRONG**:
```go
query := Task.
    SELECT(Task.AllColumns).
    WHERE(Task.Status.EQ(String(StatusNew)))
// Missing FOR UPDATE SKIP LOCKED!
```

✅ **RIGHT**:
```go
query := Task.
    SELECT(Task.AllColumns).
    WHERE(Task.Status.EQ(String(StatusNew))).
    FOR(UPDATE().SKIP_LOCKED())
```

---

### 13. Never Modify `internal/pkg/generated/`

**Rule**: Never manually edit generated code

**Rationale**: Changes will be overwritten

**Generated directories**:
- `internal/pkg/generated/postgres/` - Database models
- `internal/pkg/generated/mocks/` - Mock implementations

**To update**:
```bash
make db-models  # Regenerate database models
make mocks      # Regenerate mocks
```

---

### 14. Tests Must Be Isolated

**Rule**: Tests must not depend on each other or external state

**Rationale**: Reliable, parallelizable tests

❌ **WRONG**:
```go
var sharedTask *entity.Task  // Shared state!

func TestAddTask(t *testing.T) {
    sharedTask = &entity.Task{...}
    storage.AddTask(ctx, sharedTask)
}

func TestGetTask(t *testing.T) {
    task := storage.GetTask(ctx, sharedTask.ID)  // Depends on TestAddTask!
}
```

✅ **RIGHT**:
```go
func TestAddTask(t *testing.T) {
    task := &entity.Task{...}
    storage.AddTask(ctx, task)
    // Test completes, no shared state
}

func TestGetTask(t *testing.T) {
    task := &entity.Task{...}
    storage.AddTask(ctx, task)
    result := storage.GetTask(ctx, task.ID)
    // Independent test
}
```

---

### 15. No Data Races

**Rule**: Code must pass race detector

**Rationale**: Prevent race conditions, ensure correctness

```bash
go test -race ./...  # Must pass
```

**If race detected**: Fix immediately, don't ignore.

---

### 16. Exported Symbols Must Be Documented

**Rule**: All exported types, functions, methods must have GoDoc comments

❌ **WRONG**:
```go
type Goque struct {  // No comment!
    taskStorage TaskStorage
}

func NewGoque(storage TaskStorage) *Goque {  // No comment!
    return &Goque{taskStorage: storage}
}
```

✅ **RIGHT**:
```go
// Goque is the main task queue manager that coordinates multiple task processors.
type Goque struct {
    taskStorage TaskStorage
}

// NewGoque creates a new Goque instance with the specified task storage.
func NewGoque(storage TaskStorage) *Goque {
    return &Goque{taskStorage: storage}
}
```

---

### 17. Use Pointers for Methods

**Rule**: Methods that modify receiver must use pointer receiver

❌ **WRONG**:
```go
func (g Goque) RegisterProcessor(...)  // Value receiver, won't modify g!
```

✅ **RIGHT**:
```go
func (g *Goque) RegisterProcessor(...)  // Pointer receiver
```

**Consistency**: If any method uses pointer receiver, all methods should.

---

### 18. Defer for Cleanup

**Rule**: Always use `defer` for cleanup operations

✅ **RIGHT**:
```go
pool, err := ants.NewPool(10)
if err != nil {
    return err
}
defer pool.Release()  // Guaranteed cleanup
```

❌ **WRONG**:
```go
pool, err := ants.NewPool(10)
if err != nil {
    return err
}
// ... lots of code ...
pool.Release()  // Easy to forget!
```

---

### 19. No Panic in Library Code

**Rule**: Library code must not panic, return errors instead

**Rationale**: Users should control error handling

❌ **WRONG**:
```go
func NewGoque(storage TaskStorage) *Goque {
    if storage == nil {
        panic("storage is nil")  // Don't panic!
    }
    return &Goque{taskStorage: storage}
}
```

✅ **RIGHT**:
```go
func NewGoque(storage TaskStorage) (*Goque, error) {
    if storage == nil {
        return nil, errors.New("storage is nil")
    }
    return &Goque{taskStorage: storage}, nil
}
```

**Exception**: Panics in tests are OK.

---

### 20. No Breaking Changes Without Discussion

**Rule**: Major changes require discussion and approval

**Major changes include**:
- Public API changes
- Database schema changes
- Architecture changes
- Dependency changes

**Process**:
1. Open issue
2. Discuss approach
3. Get approval
4. Implement
5. PR review

---

### 21. Update Documentation With Code Changes

**Rule**: When making changes that affect behavior, API, or architecture, update relevant documentation in a separate commit immediately after the code change

**Rationale**: Keep documentation synchronized with code while maintaining atomic commits

**Documentation to update**:
- `README.md` - If public API, features, or usage examples change
- `agent/project_details/` - If architecture, patterns, conventions, or agreements change
- `agent/quick_load/goque.toon` - If critical rules, principles, architecture, or workflows change
- `agent/known-issues.md` - **CRITICAL: Review and update with EVERY commit**:
  - **Remove ALL resolved issues** - No "Past Issues (Resolved)" section allowed
  - Remove issues fixed in current commit
  - Remove outdated workarounds
  - Add new issues discovered during development
  - Keep ONLY current/active issues
  - Resolved issues are tracked in git history, not documentation
- GoDoc comments - Always when changing public symbols
- Migration guides - For breaking changes
- Examples - If API usage changes

**When architecture, agreements, or known issues change**:
- Always update both `agent/project_details/` (detailed) AND `agent/quick_load/goque.toon` (summary)
- Review for contradictions - if found, keep the broader interpretation
- Remove duplicate or contradicting rules - consolidate into single clear rule

❌ **WRONG**:
```bash
# Commit only code changes and forget documentation
git add internal/processor/
git commit -m "feat: add priority queue"
# Documentation update forgotten!
```

❌ **ALSO WRONG**:
```bash
# Mix code and documentation in same commit
git add internal/processor/
git add README.md
git add agent/architecture.md
git commit -m "feat: add priority queue support"
```

✅ **RIGHT**:
```bash
# First commit: code changes
git add internal/processor/
git commit -m "feat: add priority queue support

- Implement priority-based task processing
- Add priority field to task entity
- Update database schema with migration"

# Second commit: documentation
git add README.md
git add agent/architecture.md
git commit -m "docs: document priority queue feature

- Update README with priority usage examples
- Update architecture docs with priority flow
- Add examples for priority configuration"
```

**When to update**:
- ✅ New feature added → Update README, examples
- ✅ Public API changed → Update README, GoDoc, agent/project_details/conventions.md, agent/quick_load/goque.toon
- ✅ Architecture changed → Update agent/project_details/architecture.md, agent/project_details/principles.md, agent/quick_load/goque.toon
- ✅ New pattern introduced → Update agent/project_details/conventions.md
- ✅ Critical rule added/changed → Update agent/project_details/critical-rules.md AND agent/quick_load/goque.toon
- ✅ Workflow changed → Update agent/project_details/workflow.md AND agent/quick_load/goque.toon
- ✅ Agreement or convention changed → Update relevant files in agent/project_details/ AND agent/quick_load/goque.toon
- ✅ Known issue found/fixed → **Always review agent/project_details/known-issues.md**
- ✅ Bug fix with behavior change → Update README if user-facing

**Special: agent/known-issues.md maintenance**:
```bash
# When fixing a bug that was a known issue
git commit -m "fix: resolve race condition in processor"

# THEN immediately clean up known-issues.md
git add agent/known-issues.md
git commit -m "docs: remove resolved race condition from known-issues

Remove race condition entry as it's now fixed.
Resolved issues are tracked in git history."

# When discovering a new issue during development
git add agent/known-issues.md
git commit -m "docs: add database connection exhaustion to known-issues

Document potential issue with high concurrency and provide workaround."
```

**No exceptions** - Documentation must follow immediately after code changes.

---

### 22. Never Commit Sensitive Information

**Rule**: Always scan commits for sensitive data before committing. Replace any found sensitive information with placeholder text.

**Rationale**: Prevent security breaches, protect user privacy, maintain compliance

**Sensitive information includes**:
- API keys, tokens, secrets
- Passwords, private keys
- Database credentials
- User personal data (emails, phone numbers, addresses, names)
- Internal URLs, IP addresses
- OAuth client secrets
- JWT secrets
- AWS/Cloud credentials

❌ **WRONG**:
```go
// Committing real secrets
const apiKey = "sk_live_51HqJ8K2eZvKYlo2C..."
const dbPassword = "MySecretP@ssw0rd123"

// Committing user data
email := "john.doe@example.com"
phone := "+1-555-123-4567"
```

✅ **RIGHT**:
```go
// Use placeholders
const apiKey = "<your-api-key>"
const dbPassword = "<your-db-password>"

// Mask user data in examples
email := "user@example.com"
phone := "+1-555-XXX-XXXX"
```

**Before committing, check**:
```bash
# Review what you're committing
git diff --cached

# Search for common patterns
git diff --cached | grep -i "password\|token\|secret\|key"

# Use tools like git-secrets or gitleaks
git secrets --scan
```

**If you accidentally commit sensitive data**:
1. **DO NOT** just remove it in next commit (it's still in history!)
2. Use `git filter-branch` or `BFG Repo-Cleaner` to remove from history
3. Rotate compromised credentials immediately
4. Notify security team

**Environment variables**:
```go
// ✅ RIGHT: Read from environment
apiKey := os.Getenv("API_KEY")

// ❌ WRONG: Hardcode in code
apiKey := "sk_live_51HqJ8K..."
```

**Configuration files**:
- `.env.local` - ✅ In .gitignore
- `.env.example` - ✅ Safe to commit (no real values)
- `config.yaml` - ❌ Only if no secrets

**No exceptions** - Security and privacy are non-negotiable.

---

### 23. Never Commit Directly to main/master Branch

**Rule**: Always create a feature branch for any changes. Never commit directly to main or master branch.

**Rationale**: Protect main branch, enable code review, maintain clean history, allow CI/CD validation before merge

**Workflow**:

✅ **RIGHT**:
```bash
# 1. Check current branch and create feature branch if needed
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
    git checkout -b feature/add-new-feature
else
    echo "Working on existing branch: $CURRENT_BRANCH"
fi

# 2. Make changes
# ... edit files ...

# 3. Commit to feature branch
git add .
git commit -m "feat: add new feature"

# 4. Push feature branch
git push origin feature/add-new-feature

# 5. Create Pull Request on GitHub
gh pr create --title "Add new feature" --body "Description..."

# 6. After review and approval, merge via GitHub UI
# main branch stays protected
```

**Branch Strategy**:
- **If on main/master** → Create new feature branch before committing
- **If on feature branch** → Continue working on current branch
- **Exception**: If user explicitly requests new branch, create it

❌ **WRONG**:
```bash
# Working directly on main
git checkout main
git add .
git commit -m "feat: add new feature"  # ❌ Direct commit to main!
git push origin main  # ❌ Direct push to main!
```

**Branch Naming Conventions**:

Quick rules:
- **Maximum 3 words** after the prefix
- Use hyphens (not underscores or camelCase)
- Lowercase only
- Format: `<type>/<word1>-<word2>-<word3>`

Examples:
```bash
✅ feature/add-priority              # 2 words
✅ fix/race-condition                # 2 words
✅ docs/update-readme                # 2 words
❌ docs/add-branch-protection-rule-from-main  # 5 words - too long
```

**See [Branch Naming Guide](branch-naming.md) for complete rules, examples, and strategies for shortening long names.**

**When You Accidentally Commit to main**:

If you accidentally committed to main locally (not pushed):
```bash
# 1. Create a new branch with your changes
git branch feature/my-changes

# 2. Reset main to origin
git checkout main
git reset --hard origin/main

# 3. Switch to your feature branch
git checkout feature/my-changes

# 4. Continue working and push feature branch
git push origin feature/my-changes
```

If you accidentally pushed to main:
```bash
# 1. Immediately notify the team
# 2. Don't try to force push to rewrite history
# 3. Create a revert commit if needed
git revert <commit-hash>
git push origin main

# 4. Then apply changes properly via feature branch
```

**GitHub Branch Protection** (Repository Settings):

Main branch should have protection rules:
- Require pull request before merging
- Require approvals (at least 1)
- Require status checks to pass (CI/CD)
- Require branches to be up to date
- No force pushes
- No deletions

**AI Agent Checklist**:

Before making any commit:
- [ ] Check current branch: `git branch --show-current`
- [ ] If on main/master → Create new feature branch NOW
- [ ] If on feature branch → Continue on current branch
- [ ] Branch name has maximum 3 words after prefix? (e.g., `docs/add-rule` ✅, not `docs/add-branch-protection-rule-from-main` ❌)
- [ ] Exception: If user explicitly requests new branch, create it regardless

**No exceptions** - Always use feature branches, never commit directly to main/master.

---

## 🛡️ Safety Checklist

Before committing, verify:

- [ ] **On feature branch, NOT main/master** (`git branch --show-current`)
- [ ] Tests pass with race detector (`make test-cov`)
- [ ] Linter passes (`make lint`)
- [ ] Code formatted (`make fmt`)
- [ ] Errors are wrapped with `%w`
- [ ] Context is first parameter
- [ ] Exported symbols documented
- [ ] No raw SQL strings
- [ ] Database models regenerated (if schema changed)
- [ ] Mocks regenerated (if interfaces changed)
- [ ] **No sensitive information** (secrets, tokens, user personal data)
- [ ] **Documentation updated in separate commit** (README, agent/, GoDoc if needed)
- [ ] **agent/known-issues.md reviewed and cleaned** (EVERY commit):
  - [ ] ALL resolved issues removed
  - [ ] No "Past Issues" section
  - [ ] Only current issues remain

---

## 🚫 Never Do This

1. **Never commit directly to main/master** - Always use feature branches
2. **Never commit commented-out code** - Delete it, it's in git history
3. **Never commit debug prints** - Use proper logging
4. **Never commit secrets, tokens, or user personal data** - Replace with `<your-secret>` or `<user-data>`
5. **Never commit `.env.local`** - It's gitignored for a reason
6. **Never commit merge conflicts** - Resolve them properly
7. **Never force push to main** - Protected for a reason
8. **Never skip tests** - They're there for a reason
9. **Never disable linter** - Fix the issue instead
10. **Never use `// nolint` without justification** - Add comment explaining why
11. **Never leak goroutines** - Always be stoppable
12. **Never leave resolved issues in agent/known-issues.md** - Delete "Past Issues" sections immediately

---

## ⚠️ Breaking These Rules

If you find yourself needing to break a rule:

1. **Stop** - Think if there's another way
2. **Ask** - Discuss with team
3. **Document** - If approved, document why
4. **Test** - Extra testing for exceptions

**Most "exceptions" are actually wrong approaches**.
