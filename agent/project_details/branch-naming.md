# Branch Naming Conventions

## Allowed Branch Types (Quick Reference)

| Type | Purpose | Example |
|------|---------|---------|
| **feature/** | New features or enhancements | `feature/add-priority` |
| **fix/** | Bug fixes | `fix/race-condition` |
| **docs/** | Documentation changes | `docs/update-readme` |
| **refactor/** | Code restructuring without behavior changes | `refactor/simplify-storage` |
| **test/** | Adding or updating tests | `test/add-integration` |
| **chore/** | Maintenance tasks, dependencies | `chore/update-deps` |
| **perf/** | Performance improvements | `perf/optimize-queries` |
| **style/** | Code style changes (formatting) | `style/fix-formatting` |
| **ci/** | CI/CD configuration changes | `ci/add-workflow` |
| **build/** | Build system changes | `build/update-makefile` |
| **revert/** | Reverting previous commits | `revert/bad-change` |

**Most Common**: `feature/`, `fix/`, `docs/`, `refactor/`, `chore/`

---

## Core Rules

1. **Never commit to main/master** - Always use feature branches
2. **Maximum 3 words** after the type prefix
3. **Use hyphens** to separate words
4. **Keep names short and descriptive**
5. **Format**: `<type>/<word1>-<word2>-<word3>`

## Branch Types

### Feature Branches
New features or enhancements

**Format**: `feature/<name>`

✅ **GOOD**:
```bash
feature/add-priority           # 2 words
feature/webhook-delivery       # 2 words
feature/add-user-auth          # 3 words
feature/task-retry             # 2 words
feature/batch-processing       # 2 words
```

❌ **BAD**:
```bash
feature/add-new-webhook-delivery-system        # 5 words - too long
feature/implement-user-authentication-module   # 4 words - too long
feature/addPriority                            # camelCase - use hyphens
feature/Add-Priority                           # PascalCase - lowercase only
```

### Bug Fixes
Fixing bugs or issues

**Format**: `fix/<name>`

✅ **GOOD**:
```bash
fix/race-condition             # 2 words
fix/memory-leak                # 2 words
fix/task-timeout               # 2 words
fix/worker-deadlock            # 2 words
fix/nil-pointer                # 2 words
```

❌ **BAD**:
```bash
fix/race-condition-in-worker-pool              # 5 words - too long
fix/resolve-memory-leak-in-processor           # 5 words - too long
fix/bug                                         # too vague
fix/issue-123                                   # use descriptive name, not issue number
```

### Refactoring
Code restructuring without behavior changes

**Format**: `refactor/<name>`

✅ **GOOD**:
```bash
refactor/simplify-storage      # 2 words
refactor/improve-api           # 2 words
refactor/extract-helpers       # 2 words
refactor/cleanup-processor     # 2 words
refactor/optimize-queries      # 2 words
```

❌ **BAD**:
```bash
refactor/improve-storage-interface-design      # 4 words - too long
refactor/refactor-code                          # redundant "refactor"
refactor/changes                                # too vague
```

### Documentation
Documentation changes

**Format**: `docs/<name>`

✅ **GOOD**:
```bash
docs/update-readme             # 2 words
docs/add-architecture          # 2 words
docs/fix-examples              # 2 words
docs/add-branch-rules          # 3 words
docs/update-api-guide          # 3 words
```

❌ **BAD**:
```bash
docs/add-branch-protection-rule-from-main      # 5 words - too long
docs/update-documentation-for-new-features     # 5 words - too long
docs/update                                     # too vague
```

### Chores
Maintenance tasks, dependency updates, etc.

**Format**: `chore/<name>`

✅ **GOOD**:
```bash
chore/update-dependencies      # 2 words
chore/bump-go-version          # 3 words
chore/fix-linter               # 2 words
chore/update-ci                # 2 words
chore/cleanup-tests            # 2 words
```

❌ **BAD**:
```bash
chore/update-all-project-dependencies-to-latest   # 6 words - too long
chore/maintenance                                  # too vague
```

### Performance
Performance improvements

**Format**: `perf/<name>`

✅ **GOOD**:
```bash
perf/optimize-queries          # 2 words
perf/improve-worker-pool       # 3 words
perf/cache-results             # 2 words
perf/reduce-allocations        # 2 words
```

### Tests
Adding or updating tests

**Format**: `test/<name>`

✅ **GOOD**:
```bash
test/add-integration           # 2 words
test/fix-race-tests            # 3 words
test/improve-coverage          # 2 words
test/add-benchmarks            # 2 words
```

## Naming Guidelines

### Be Descriptive
Names should clearly indicate what the branch does

✅ **GOOD**:
```bash
feature/webhook-delivery       # Clear: adds webhook delivery
fix/race-condition             # Clear: fixes race condition
docs/add-architecture          # Clear: adds architecture docs
```

❌ **BAD**:
```bash
feature/new-stuff              # Vague
fix/bug                        # Vague
docs/update                    # Vague
```

### Use Action Verbs
Start with action verbs when appropriate

✅ **GOOD**:
```bash
feature/add-priority           # Action: add
fix/resolve-deadlock           # Action: resolve
refactor/simplify-code         # Action: simplify
docs/update-readme             # Action: update
```

### Abbreviate When Needed
If name is too long, use abbreviations

✅ **GOOD**:
```bash
feature/auth-api               # Instead of: authentication-api-endpoints
feature/webhook-delivery       # Instead of: webhook-delivery-system
docs/api-guide                 # Instead of: api-usage-guide
```

### Avoid Numbers
Don't use issue numbers or version numbers in branch names

✅ **GOOD**:
```bash
fix/race-condition             # Descriptive
feature/add-priority           # Descriptive
```

❌ **BAD**:
```bash
fix/issue-123                  # Use description, not issue number
feature/v2-api                 # Don't use version numbers
```

### Use Lowercase Only
All branch names should be lowercase with hyphens

✅ **GOOD**:
```bash
feature/add-priority
fix/race-condition
docs/update-readme
```

❌ **BAD**:
```bash
feature/AddPriority            # PascalCase
feature/add_priority           # Underscores
FEATURE/ADD-PRIORITY           # Uppercase
```

## Word Count Examples

### 1 Word (OK)
```bash
feature/webhooks
fix/deadlock
docs/architecture
refactor/storage
```

### 2 Words (IDEAL)
```bash
feature/add-priority
fix/race-condition
docs/update-readme
refactor/simplify-code
```

### 3 Words (MAXIMUM)
```bash
feature/add-user-auth
fix/task-timeout-error
docs/add-branch-rules
refactor/improve-error-handling
```

### 4+ Words (TOO LONG)
```bash
feature/add-new-user-authentication           # 4 words ❌
fix/resolve-task-timeout-error-issue          # 5 words ❌
docs/add-branch-protection-rule-from-main     # 5 words ❌
refactor/improve-error-handling-in-processor  # 5 words ❌
```

## Shortening Long Names

### Strategy 1: Remove Redundant Words
```bash
# Before (4 words)
feature/add-new-webhook-system

# After (2 words)
feature/webhook-system         # "add" and "new" are implied
```

### Strategy 2: Use Abbreviations
```bash
# Before (4 words)
feature/authentication-api-endpoints

# After (2 words)
feature/auth-api               # "auth" instead of "authentication"
```

### Strategy 3: Simplify Description
```bash
# Before (5 words)
fix/resolve-race-condition-in-worker-pool

# After (2 words)
fix/worker-race                # Simplified, still clear
```

### Strategy 4: Focus on Core Concept
```bash
# Before (5 words)
docs/add-comprehensive-architecture-guide-for-agents

# After (2 words)
docs/add-architecture          # Core concept preserved
```

## Real-World Examples

### Good Branch Names from Project History

```bash
# From actual project
feature/add-priority-queue             # 3 words ✅
fix/race-condition                     # 2 words ✅
refactor/simplify-storage              # 2 words ✅
chore/update-dependencies              # 2 words ✅
```

### Bad Branch Names to Avoid

```bash
# Too long
docs/add-branch-protection-rule-from-main          # 5 words ❌

# Should be:
docs/add-branch-rules                              # 3 words ✅

# Too vague
feature/new-stuff                                   # Vague ❌

# Should be:
feature/webhook-delivery                            # Clear ✅

# Using issue numbers
fix/issue-123                                       # Not descriptive ❌

# Should be:
fix/memory-leak                                     # Descriptive ✅
```

## Checklist

Before creating a branch:

- [ ] Branch name has a type prefix (feature/, fix/, docs/, etc.)
- [ ] Name has maximum 3 words after the prefix
- [ ] Words are separated by hyphens (not underscores or camelCase)
- [ ] Name is lowercase only
- [ ] Name is descriptive and clear
- [ ] Name uses action verbs when appropriate
- [ ] Name doesn't include issue numbers or version numbers
- [ ] If name seems too long, it has been shortened

## Common Mistakes

### Mistake 1: Too Many Words
```bash
❌ feature/add-new-webhook-delivery-notification-system
✅ feature/webhook-notifications
```

### Mistake 2: Using Underscores
```bash
❌ feature/add_webhook_support
✅ feature/add-webhook-support
```

### Mistake 3: Using CamelCase
```bash
❌ feature/addWebhookSupport
✅ feature/add-webhook-support
```

### Mistake 4: Too Vague
```bash
❌ fix/bug
✅ fix/race-condition
```

### Mistake 5: Including Issue Numbers
```bash
❌ feature/issue-123-add-webhooks
✅ feature/add-webhooks
```

### Mistake 6: Uppercase Letters
```bash
❌ Feature/Add-Webhooks
✅ feature/add-webhooks
```

## When You Have a Long Name

If you find yourself needing more than 3 words, ask:

1. **Can I remove redundant words?**
   - "add-new-webhook-system" → "webhook-system"

2. **Can I use abbreviations?**
   - "authentication-api" → "auth-api"

3. **Can I simplify the description?**
   - "fix-race-condition-in-worker" → "fix-worker-race"

4. **Is the scope too broad?**
   - Maybe split into multiple branches
   - "add-auth-and-logging" → "add-auth" + "add-logging"

## Summary

**Golden Rules**:
1. Maximum 3 words after type prefix
2. Use hyphens, lowercase only
3. Be descriptive and clear
4. Avoid redundancy and vagueness

**Format**: `<type>/<word1>-<word2>-<word3>`

**Examples**:
- ✅ `feature/add-priority` (2 words)
- ✅ `fix/race-condition` (2 words)
- ✅ `docs/update-readme` (2 words)
- ✅ `refactor/simplify-code` (2 words)
- ❌ `feature/add-new-priority-queue-system` (5 words)
