# Commit Message Conventions

## Format

```
<type>(<scope>): <subject>

<body>

<footer>
```

## Type

Must be one of the following:

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation only changes
- **style**: Code style changes (formatting, missing semi-colons, etc.)
- **refactor**: Code change that neither fixes a bug nor adds a feature
- **perf**: Performance improvement
- **test**: Adding or updating tests
- **chore**: Changes to build process or auxiliary tools
- **ci**: Changes to CI configuration
- **build**: Changes to build system or dependencies
- **revert**: Revert a previous commit

## Scope (Optional)

The scope should be the name of the affected component:

- `processor` - Queue processor changes
- `storage` - Storage layer changes
- `healer` - Healer processor changes
- `cleaner` - Cleaner processor changes
- `entity` - Entity/domain model changes
- `migration` - Database migration changes
- `api` - Public API changes
- `test` - Test infrastructure changes
- `docs` - Documentation changes

## Subject

The subject contains a succinct description of the change:

- Use the imperative, present tense: "change" not "changed" nor "changes"
- Don't capitalize the first letter
- No period (.) at the end
- Maximum 50 characters

## Body (Optional but recommended)

The body should include:

- Motivation for the change
- Contrast with previous behavior
- Implementation details (if complex)

Rules:
- Use imperative, present tense
- Wrap at 72 characters
- Separate from subject with blank line

## Footer (Optional)

The footer should contain:

- Breaking changes (start with `BREAKING CHANGE:`)
- Issue references (e.g., `Closes #123`, `Fixes #456`)
- Co-authors (if pair programming)

## Examples

### Simple Feature

```
feat: add task priority support

Add priority field to tasks to allow important tasks to be processed
first. Tasks with higher priority values are fetched before lower
priority tasks.

Closes #123
```

### Bug Fix

```
fix: prevent duplicate task processing

Add FOR UPDATE SKIP LOCKED to task fetch query to prevent multiple
workers from fetching the same task. This fixes a race condition
that could occur when multiple workers fetch tasks simultaneously.

Fixes #456
```

### Breaking Change

```
feat(api): change processor registration API

BREAKING CHANGE: RegisterProcessor now returns error instead of panicking
when processor type is already registered.

Before:
  goque.RegisterProcessor("email", processor)

After:
  err := goque.RegisterProcessor("email", processor)
  if err != nil {
    // handle error
  }

This allows library users to handle registration errors gracefully
instead of experiencing panics.

Closes #789
```

### Documentation

```
docs: add architecture documentation

Add comprehensive architecture documentation in agent/architecture.md
describing system components, data flow, and design patterns.
```

### Refactoring

```
refactor(processor): simplify option handling

Consolidate processor options into single struct instead of using
multiple separate fields. This makes it easier to add new options
without changing function signatures.

No behavior changes.
```

### Performance

```
perf(storage): batch task status updates

Update task status in batches instead of one-by-one to reduce
database round trips. Improves throughput by ~30% when processing
1000+ tasks.

Benchmarks:
  Before: 1000 updates in 5s
  After:  1000 updates in 3.5s
```

### Test

```
test(processor): add test for concurrent task processing

Add integration test that verifies multiple workers don't process
the same task when fetching concurrently. Uses 10 workers to
simulate realistic concurrency.
```

### Chore

```
chore: update dependencies

Update go-jet to v2.14.0 and testify to v1.11.1 for latest
bug fixes and features.
```

### Revert

```
revert: revert "feat: add task priority support"

This reverts commit abc123def456.

Reason: Priority implementation causes performance regression
in task fetching. Need to redesign approach.
```

## Multi-line Examples

### Complex Feature

```
feat(processor): add configurable retry backoff strategies

Add support for custom retry backoff strategies through new
WithNextAttemptAtFunc option. This allows users to implement
exponential backoff, constant delay, or any custom retry logic.

Changes:
- Add NextAttemptAtFunc type definition
- Add WithNextAttemptAtFunc option
- Update processor to use custom function if provided
- Fall back to default linear backoff if not provided
- Add comprehensive tests for different strategies

Example usage:
  processor := NewGoqueProcessor(
    storage,
    "email",
    emailProcessor,
    WithNextAttemptAtFunc(func(task *entity.Task) time.Time {
      // Exponential backoff: 1s, 2s, 4s, 8s, ...
      delay := time.Second * time.Duration(math.Pow(2, float64(task.Attempts)))
      return time.Now().Add(delay)
    }),
  )

Closes #234
```

### Complex Bug Fix

```
fix(storage): fix race condition in task fetching

Fix race condition where multiple workers could fetch and process
the same task when fetching occurs simultaneously. This was caused
by missing FOR UPDATE SKIP LOCKED clause in the fetch query.

Root cause:
- Original query: SELECT ... WHERE status='new' LIMIT 10
- Multiple workers execute this simultaneously
- All workers see same tasks before any are marked as processing
- Same task gets processed multiple times

Solution:
- Add FOR UPDATE SKIP LOCKED to query
- First worker locks the row
- Other workers skip locked rows and fetch different tasks
- Each task processed exactly once

Also add integration test that spawns 10 workers and verifies
each task is processed exactly once when processing 100 tasks.

Fixes #567
```

## Tips

### Good Commit Messages

✅ **GOOD**:
```
feat: add webhook delivery processor

Add built-in processor for delivering webhooks with automatic
retries. Includes exponential backoff and configurable timeout.

Closes #123
```

Characteristics:
- Clear what was added
- Explains purpose
- References issue

✅ **GOOD**:
```
fix: prevent goroutine leak in processor shutdown

Ensure all worker goroutines exit when processor is stopped by
properly closing the worker pool. Previously, goroutines could
hang indefinitely waiting for tasks.

Fixes #456
```

Characteristics:
- Clear what was fixed
- Explains the problem
- Describes solution
- References issue

### Bad Commit Messages

❌ **BAD**:
```
update code
```

Problems:
- Too vague
- No context
- What was updated?

❌ **BAD**:
```
fix bug
```

Problems:
- Which bug?
- How was it fixed?
- No issue reference

❌ **BAD**:
```
WIP
```

Problems:
- Work in progress shouldn't be committed
- Commit when work is complete

❌ **BAD**:
```
feat: Add awesome new feature that does many things including task processing and webhook delivery and also fixes some bugs and improves performance
```

Problems:
- Too long subject (>50 chars)
- Multiple changes in one commit
- Should be split into multiple commits

## Commit Size

### One Logical Change Per Commit

✅ **GOOD** - Separate commits:
```
Commit 1: feat: add priority field to task entity
Commit 2: feat: implement priority-based task fetching
Commit 3: test: add tests for task priority
Commit 4: docs: document task priority feature
```

❌ **BAD** - Single commit:
```
feat: add task priority support, fix bug in healer, update dependencies, refactor storage interface
```

### Atomic Commits

Each commit should:
- Compile successfully
- Pass all tests
- Contain a complete logical change
- Be revertible without breaking the system

## Commit Frequency

### When to Commit

Commit when:
- Feature is complete
- Bug is fixed
- Test is added
- Documentation is updated
- Refactoring step is done

Don't commit:
- In the middle of a change
- When tests are failing
- When code doesn't compile
- Work in progress (use git stash instead)

### Multiple Small Commits vs One Large Commit

Prefer multiple small commits over one large commit:

✅ **GOOD**:
```
feat(entity): add priority field to Task
feat(storage): implement priority-based fetching
test(storage): add tests for priority fetching
docs: document priority feature
```

Each commit is small, focused, and reviewable.

❌ **BAD**:
```
feat: add priority support (500 lines changed, 10 files modified)
```

Too large, hard to review, mixes concerns.

## Issue References

### Formats

```
Closes #123          # Closes issue #123
Fixes #456           # Fixes bug #456
Resolves #789        # Resolves issue #789
Relates to #234      # Related but doesn't close
See #567             # Reference for context
```

### Multiple Issues

```
Closes #123, #456
```

or

```
Closes #123
Closes #456
```

### GitHub Keywords

GitHub automatically closes issues when these keywords are used:

- close, closes, closed
- fix, fixes, fixed
- resolve, resolves, resolved

## Co-Authors

When pair programming:

```
feat: add webhook processor

Implemented by pair programming session.

Co-authored-by: John Doe <john@example.com>
```

## Signed Commits

If using GPG signing:

```bash
git commit -S -m "feat: add feature"
```

## Amending Commits

Fix last commit:

```bash
# Fix commit message
git commit --amend

# Add forgotten files
git add forgotten_file.go
git commit --amend --no-edit
```

**Warning**: Never amend commits that have been pushed and shared!

## Interactive Rebase

Clean up commits before pushing:

```bash
# Edit last 3 commits
git rebase -i HEAD~3

# Options:
# pick = use commit
# reword = use commit, but edit message
# edit = use commit, but stop for amending
# squash = merge into previous commit
# fixup = like squash, but discard commit message
# drop = remove commit
```

## Conventional Commits

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- Provides structured commit history
- Enables automatic changelog generation
- Enables automatic semantic versioning
- Makes history searchable

## Tools

### Commit Message Template

Create `.gitmessage` template:

```
# <type>(<scope>): <subject>
#
# <body>
#
# <footer>
#
# Type: feat, fix, docs, style, refactor, perf, test, chore
# Scope: processor, storage, healer, cleaner, entity, etc.
# Subject: imperative, present tense, no period, max 50 chars
# Body: explain what and why, wrap at 72 chars
# Footer: breaking changes, issue references
```

Configure Git:
```bash
git config commit.template .gitmessage
```

### Commitlint

Consider using commitlint to enforce conventions:

```bash
npm install -g @commitlint/cli @commitlint/config-conventional
```

## Examples from Real Commits

Based on project history:

### Refactoring
```
refactor(processor): improve task processing and cleanup logic

Simplify processor shutdown logic and improve worker pool cleanup.
Extract retry logic into separate function for better testability.

No behavior changes.
```

### Documentation
```
docs: reorganize documentation into agent directory

Move all agent-related documentation into agent/ directory for
better organization. Add comprehensive guides for AI agents
working on the project.
```

### Feature
```
feat: add linting documentation

Document linting process and golangci-lint configuration.
Explain how to run linter and fix common issues.
```

## Summary

Good commit messages:
- Follow conventions
- Are clear and descriptive
- Explain what and why
- Reference issues
- Use proper formatting
- Are atomic and focused

Bad commit messages:
- Are vague
- Lack context
- Mix multiple changes
- Don't reference issues
- Have poor formatting
