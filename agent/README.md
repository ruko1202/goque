# Agent Documentation

This directory contains comprehensive documentation for AI agents working on the Goque project.

## Quick Navigation

- [Overview](overview.md) - High-level project overview and purpose
- [Architecture](architecture.md) - Technical architecture and component design
- [Tech Stack](tech-stack.md) - Technologies, libraries, and tools
- [Conventions](conventions.md) - Code conventions and patterns
- [Principles](principles.md) - Design principles and best practices
- [Critical Rules](critical-rules.md) - Must-follow rules and constraints
- [Branch Naming](branch-naming.md) - Branch naming conventions and rules
- [Workflow](workflow.md) - Development workflow and processes
- [Checklist](checklist.md) - Development and PR checklist
- [Commits](commits.md) - Commit message conventions
- [Known Issues](known-issues.md) - Known issues and workarounds
- [Links](links.md) - Useful links and resources

## Quick Start for AI Agents

1. Read [Overview](overview.md) to understand what Goque is
2. Review [Architecture](architecture.md) to understand the system design
3. Check [Critical Rules](critical-rules.md) before making any changes
4. Follow [Conventions](conventions.md) when writing code
5. Use [Checklist](checklist.md) before submitting changes

## ⚠️ Critical Rules Summary

### Rule #23: Never Commit to main/master
**ALWAYS work on feature branches. NEVER commit directly to main or master.**

```bash
# Before starting ANY work
git checkout main
git pull
git checkout -b feature/your-feature-name  # Max 3 words after prefix!

# Always verify
git branch --show-current  # Must NOT be 'main' or 'master'
```

**Branch Naming**:
- Maximum 3 words after prefix (e.g., `docs/add-rule` ✅)
- Use hyphens, lowercase only
- See [Branch Naming Guide](branch-naming.md) for complete rules and examples

See [Critical Rules #23](critical-rules.md#23-never-commit-directly-to-mainmaster-branch) for details.

### Rule #21: Documentation Updates
**ALWAYS update documentation when making code changes!**

When you change code that affects:
- **Public API** → Update README.md + GoDoc comments
- **Features** → Update README.md examples
- **Architecture** → Update agent/architecture.md
- **Patterns/Conventions** → Update agent/conventions.md
- **Critical rules** → Update agent/critical-rules.md
- **Known issues** → Update agent/known-issues.md

**Documentation must be updated in a separate commit immediately after code changes.**

See [Critical Rules #21](critical-rules.md#21-update-documentation-with-code-changes) for details.

## Project Summary

**Goque** is a robust, PostgreSQL-backed task queue system for Go with:
- Worker pool management
- Automatic retry logic
- Task lifecycle management
- Graceful shutdown support
- Built-in task healer

## Key Components

- **Goque Manager** - Main queue coordinator
- **Queue Processor** - Task processing engine
- **Task Storage** - PostgreSQL-backed persistence
- **Internal Processors** - Healer and Cleaner
- **Queue Manager** - Task submission interface
