# Development Checklist

## Pre-Development Checklist

Before starting work:

- [ ] Issue exists describing the work
- [ ] Issue has been discussed and approach agreed upon
- [ ] Latest code pulled from main
- [ ] **New feature branch created from main** (NOT working on main/master!)
- [ ] **Verified current branch** (`git branch --show-current` shows feature branch)
- [ ] **Branch name has max 3 words** after prefix - see [Branch Naming](branch-naming.md)
- [ ] Database is running and accessible
- [ ] Migrations are up to date (`make db-status`)
- [ ] All tests pass on main (`make test-cov`)

## During Development Checklist

While coding:

### Code Quality
- [ ] Following Go conventions (see conventions.md)
- [ ] Following project principles (see principles.md)
- [ ] Not violating critical rules (see critical-rules.md)
- [ ] Code is simple and readable
- [ ] No unnecessary complexity
- [ ] No code duplication (DRY)

### Error Handling
- [ ] All errors are handled
- [ ] Errors are wrapped with `%w`
- [ ] Error messages are descriptive
- [ ] No silent failures
- [ ] No panics in library code

### Context Usage
- [ ] Context is first parameter
- [ ] Context is propagated down call stack
- [ ] Context cancellation is respected
- [ ] Timeouts are configured where appropriate

### Concurrency
- [ ] No unbounded goroutines
- [ ] Using worker pools (ants)
- [ ] No data races
- [ ] Proper synchronization
- [ ] All goroutines are stoppable

### Database
- [ ] Using go-jet (no raw SQL)
- [ ] Using `FOR UPDATE SKIP LOCKED` for task fetching
- [ ] Queries are efficient
- [ ] Proper indexes exist
- [ ] Transactions used where needed

### Testing
- [ ] Tests written for new code
- [ ] Tests cover edge cases
- [ ] Tests are isolated
- [ ] Tests don't depend on execution order
- [ ] Tests use mocks where appropriate
- [ ] Tests have meaningful names

### Documentation (CRITICAL - Must complete in separate commit!)
- [ ] All exported symbols have GoDoc comments
- [ ] Complex logic has explanatory comments
- [ ] Comments explain "why" not "what"
- [ ] **README.md updated in separate docs commit** if:
  - [ ] Public API changed
  - [ ] New feature added
  - [ ] Usage examples affected
  - [ ] Configuration options changed
- [ ] **agent/ docs updated in separate docs commit** if:
  - [ ] Architecture changed (→ agent/architecture.md)
  - [ ] New convention introduced (→ agent/conventions.md)
  - [ ] Design principle applied (→ agent/principles.md)
  - [ ] Critical rule added (→ agent/critical-rules.md)
  - [ ] Known issue found/fixed (→ agent/known-issues.md)
  - [ ] Workflow changed (→ agent/workflow.md)
- [ ] **agent/known-issues.md reviewed and cleaned** (CRITICAL):
  - [ ] Removed ALL resolved issues (no "Past Issues" section)
  - [ ] Removed issues fixed in current commit
  - [ ] Removed outdated workarounds
  - [ ] Added new issues if discovered
  - [ ] Kept ONLY current/active issues
- [ ] Migration guide created for breaking changes
- [ ] Examples updated if API usage changed

## Pre-Commit Checklist

Before committing:

### Branch Check (CRITICAL!)
- [ ] **Verify NOT on main/master**: `git branch --show-current`
- [ ] Working on a feature branch with descriptive name
- [ ] **Branch name has max 3 words** after prefix - see [Branch Naming](branch-naming.md)
- [ ] If on main/master by mistake, create feature branch immediately

### Run Checks
- [ ] All tests pass with race detector: `make test-cov`
- [ ] Linter passes: `make lint`
- [ ] Code formatted: `make fmt`

### Database Changes
If database schema changed:
- [ ] Migration created: `make db-migrate-create name="..."`
- [ ] Migration has both UP and DOWN
- [ ] Migration tested (up and down)
- [ ] Models regenerated: `make db-models`

### Interface Changes
If interfaces changed:
- [ ] Mocks regenerated: `make mocks`
- [ ] All implementations updated
- [ ] Tests updated

### Commit Message
- [ ] Follows commit conventions (see commits.md)
- [ ] Has type prefix (feat:, fix:, etc.)
- [ ] Has clear description
- [ ] References issue number
- [ ] Describes what changed and why
- [ ] **Mentions documentation updates** (if any)

### Documentation Updates
If code changes affect documentation:
- [ ] **Documentation changes prepared for separate commit**
- [ ] README.md updated (if public API/features changed)
- [ ] agent/ docs updated (if architecture/patterns changed)
- [ ] **agent/known-issues.md reviewed and cleaned** (check EVERY commit):
  - [ ] Removed resolved issues
  - [ ] No "Past Issues" section
- [ ] GoDoc comments updated (if public symbols changed)
- [ ] Documentation commit follows code commit immediately

### Generated Code
- [ ] No manual edits to `internal/pkg/generated/`
- [ ] Generated code is committed
- [ ] Generated code is up to date

## Pre-PR Checklist

Before creating Pull Request:

### Code Review
- [ ] Self-reviewed the changes
- [ ] Removed debug code
- [ ] Removed commented-out code
- [ ] No TODOs or FIXMEs (or tracked as issues)
- [ ] No secrets or credentials committed

### Testing
- [ ] All tests pass locally
- [ ] Added tests for new functionality
- [ ] Added tests for bug fixes
- [ ] Edge cases covered
- [ ] Test coverage maintained or improved

### Documentation (CRITICAL - Must be complete in separate commit!)
- [ ] **README.md updated in separate docs commit** if:
  - [ ] Public API changed
  - [ ] New features added
  - [ ] Usage examples affected
  - [ ] Installation/setup changed
- [ ] **agent/ documentation updated in separate docs commit** if:
  - [ ] Architecture changed (agent/architecture.md)
  - [ ] Conventions changed (agent/conventions.md)
  - [ ] Principles applied (agent/principles.md)
  - [ ] Critical rules added (agent/critical-rules.md)
  - [ ] Known issues added/fixed (agent/known-issues.md)
  - [ ] Workflow changed (agent/workflow.md)
- [ ] **agent/known-issues.md verified clean** (CRITICAL for every PR):
  - [ ] NO "Past Issues (Resolved)" section exists
  - [ ] ALL resolved issues removed
  - [ ] Only current/active issues remain
  - [ ] Outdated workarounds removed
- [ ] **GoDoc comments** added/updated for all exported symbols
- [ ] **Migration guide** written for breaking changes (if any)
- [ ] **Examples** updated if API usage changed
- [ ] All documentation is clear and accurate
- [ ] Documentation commit follows code commit immediately

### API Changes
If public API changed:
- [ ] Changes discussed and approved
- [ ] Backward compatible (or major version bump planned)
- [ ] Migration guide provided
- [ ] Example usage updated

### Performance
- [ ] No obvious performance regressions
- [ ] Efficient algorithms used
- [ ] No unnecessary allocations
- [ ] Database queries optimized

### Security
- [ ] No SQL injection possible
- [ ] No race conditions
- [ ] Proper error handling (no information leakage)
- [ ] Dependencies vetted

## PR Description Checklist

Pull Request should include:

- [ ] Clear title following conventions
- [ ] Summary of changes
- [ ] Why the change is needed
- [ ] How the change works
- [ ] Link to related issue(s)
- [ ] Breaking changes highlighted (if any)
- [ ] Screenshots/examples (if UI/API changes)
- [ ] Test plan described

### PR Template Example:
```markdown
## Summary
Brief description of changes

## Motivation
Why this change is needed

## Changes
- Bullet point list of changes
- What was modified
- What was added
- What was removed

## Breaking Changes
- List any breaking changes
- Or state "None"

## Testing
- How it was tested
- What scenarios were covered

## Related Issues
Closes #123
```

## Code Review Checklist

When reviewing others' code:

### Functionality
- [ ] Code does what PR says it does
- [ ] No bugs or logic errors
- [ ] Edge cases handled
- [ ] Error handling is correct

### Design
- [ ] Follows architecture principles
- [ ] Uses appropriate patterns
- [ ] Properly abstracted
- [ ] No unnecessary complexity

### Code Quality
- [ ] Follows conventions
- [ ] Readable and maintainable
- [ ] Well-named variables/functions
- [ ] Appropriate comments

### Tests
- [ ] Tests exist and are sufficient
- [ ] Tests are clear and maintainable
- [ ] Tests cover edge cases
- [ ] Tests are not flaky

### Performance
- [ ] No obvious performance issues
- [ ] Efficient algorithms
- [ ] No unnecessary work

### Security
- [ ] No security vulnerabilities
- [ ] Proper input validation
- [ ] No race conditions

## Post-Merge Checklist

After PR is merged:

- [ ] Delete feature branch (locally and remotely)
- [ ] Verify CI passes on main
- [ ] Update local main branch
- [ ] Close related issue(s)
- [ ] Update project board (if used)
- [ ] Notify relevant people (if needed)

## Release Checklist

When preparing a release:

### Pre-Release
- [ ] All planned features merged
- [ ] All tests pass
- [ ] No known critical bugs
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Version number decided
- [ ] Release notes drafted

### Release
- [ ] Tag created
- [ ] GitHub release created
- [ ] Release notes published
- [ ] Announcement made

### Post-Release
- [ ] Monitor for issues
- [ ] Respond to user feedback
- [ ] Update examples/docs if needed

## Emergency Hotfix Checklist

For critical bugs in production:

### Immediate
- [ ] Identify the bug
- [ ] Assess severity
- [ ] Notify team
- [ ] Create hotfix branch from main

### Fix
- [ ] Write failing test
- [ ] Implement fix
- [ ] Verify test passes
- [ ] Run full test suite

### Deploy
- [ ] Fast-track PR review
- [ ] Merge to main
- [ ] Create patch release
- [ ] Deploy immediately
- [ ] Verify fix in production

### Follow-up
- [ ] Post-mortem if needed
- [ ] Update monitoring
- [ ] Prevent recurrence

## Migration Checklist

For database migrations:

### Before Migration
- [ ] Backup database
- [ ] Test migration on copy
- [ ] Verify rollback works
- [ ] Plan downtime (if needed)

### During Migration
- [ ] Run migration: `make db-up`
- [ ] Verify status: `make db-status`
- [ ] Check data integrity
- [ ] Regenerate models: `make db-models`

### After Migration
- [ ] Run tests: `make test-cov`
- [ ] Verify application works
- [ ] Monitor for issues
- [ ] Document any issues

## Dependency Update Checklist

When updating dependencies:

- [ ] Check for breaking changes
- [ ] Read changelog/release notes
- [ ] Update `go.mod`: `go get -u <package>`
- [ ] Run tests: `make test-cov`
- [ ] Check for deprecation warnings
- [ ] Update code if needed
- [ ] Test thoroughly
- [ ] Commit with clear message

## Refactoring Checklist

When refactoring:

### Before
- [ ] Tests exist and pass
- [ ] Understand current code
- [ ] Plan refactoring approach
- [ ] Discuss if major refactoring

### During
- [ ] Keep tests passing (green)
- [ ] Make small, incremental changes
- [ ] Commit frequently
- [ ] Don't change behavior

### After
- [ ] All tests still pass
- [ ] Code is cleaner
- [ ] Performance not worse
- [ ] Review before merging

## Performance Optimization Checklist

Before optimizing:

- [ ] Profile first (identify bottleneck)
- [ ] Measure baseline performance
- [ ] Set target performance
- [ ] Consider if optimization needed

While optimizing:

- [ ] Optimize bottleneck only
- [ ] Measure after each change
- [ ] Keep tests passing
- [ ] Don't sacrifice readability

After optimizing:

- [ ] Verify improvement
- [ ] Document performance characteristics
- [ ] Add performance tests
- [ ] Consider trade-offs

## Security Review Checklist

For security-sensitive changes:

- [ ] Input validation comprehensive
- [ ] No SQL injection possible
- [ ] No race conditions
- [ ] Proper error handling
- [ ] No information leakage
- [ ] Dependencies up to date
- [ ] No hardcoded secrets
- [ ] Audit logging added (if needed)
- [ ] Security review conducted
- [ ] Penetration test considered
