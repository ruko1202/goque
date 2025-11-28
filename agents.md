# Load Goque Agent Context

## Quick Load (Recommended)

For fastest context loading with minimal tokens, read the TOON-formatted documentation:

**📦 agent/quick_load/**
- **goque.toon** - Complete project context in Token-Oriented Object Notation
  - ⚡ **88% fewer tokens** than markdown (~4K tokens vs ~34K tokens)
  - 📦 Single file with all essential information
  - ✅ 100% coverage of critical rules, principles, and workflows
  - 🎯 Structured tables for easy parsing
  - Architecture, tech stack, critical rules, conventions, principles
  - Quick start guide and safety checklist

## Detailed Load (Alternative)

For comprehensive documentation with detailed explanations, read the following files from **agent/project_details/** directory:

### Core Documentation (MUST READ)

Read these files first to understand the project fundamentals:

- **README.md** - Navigation hub and quick start guide
- **overview.md** - Project purpose, goals, and high-level overview
- **architecture.md** - Technical architecture, components, and design patterns
- **critical-rules.md** - 23 non-negotiable rules (SQL injection, error handling, branch strategy, etc.)
- **conventions.md** - Code style, naming patterns, and interface design
- **principles.md** - 26 design principles (reliability, simplicity, composability)

### Workflow & Process

Essential workflow documentation:

- **workflow.md** - Development workflow, Git processes, testing requirements
- **checklist.md** - Development and PR checklist
- **commits.md** - Commit message conventions
- **branch-naming.md** - Branch naming rules and strategies

### Reference Documentation

Additional reference materials:

- **tech-stack.md** - Technologies, libraries, and tools used
- **known-issues.md** - Known issues and workarounds
- **links.md** - Useful links and resources

## What You'll Learn

After reading the documentation, you will understand:

✓ **Project Purpose**: Robust database-backed task queue for Go with PostgreSQL, MySQL, and SQLite support
✓ **Critical Rules**: 23 must-follow rules including context-first, error handling, go-jet usage, branch strategy
✓ **Architecture**: Task lifecycle, processor pattern, storage layer, multi-database support
✓ **Conventions**: Naming patterns, error handling, interface design
✓ **Principles**: Reliability, simplicity, composability, extensibility
✓ **Workflow**: Testing requirements, migration process, PR standards, branch protection

## Directory Structure

```
agent/
├── quick_load/          # TOON-formatted documentation (fast loading)
│   └── goque.toon       # Complete project context in TOON format
│
├── project_details/     # Detailed markdown documentation
│   ├── README.md        # Documentation navigation
│   ├── overview.md      # Project overview
│   ├── architecture.md  # System architecture
│   ├── critical-rules.md  # 23 critical rules
│   ├── conventions.md   # Code conventions
│   ├── principles.md    # Design principles
│   ├── workflow.md      # Development workflow
│   ├── checklist.md     # Development checklist
│   ├── commits.md       # Commit conventions
│   ├── branch-naming.md # Branch naming guide
│   ├── tech-stack.md    # Technology stack
│   ├── known-issues.md  # Known issues
│   └── links.md         # Useful links
│
└── tasks/               # Active development tasks
    └── ...

```

## Usage Tips

### Default: Use TOON Format
```
Read agent/quick_load/goque.toon
```
- ⚡ **88% token savings** - 4K tokens instead of 34K
- 📦 **One file** instead of 13 markdown files
- ✅ **100% critical info** - all rules, principles, workflows
- 🚀 **Fast loading** - single read operation
- 🎯 **Perfect for**: Regular work, quick reference, token efficiency

### Alternative: Use Markdown When You Need
```
Read all files in agent/project_details/ in parallel
```
- 📚 **Deep dive** - first-time project exploration
- 🔍 **Code examples** - detailed implementation examples
- 🛠️ **Troubleshooting** - known-issues.md with solutions
- 🔗 **Resources** - links.md with external materials
- 📝 **Checklists** - detailed development checklists

### Before Starting Work
1. **Load context**: Use TOON format (`agent/quick_load/goque.toon`)
2. **Check branch**: `git branch --show-current`
3. **Create feature branch** if on main/master
4. **Review critical rules** (23 non-negotiable rules in TOON)
5. **Start implementation** following conventions

## File Formats

- **TOON Format** (agent/quick_load/) - Token-Oriented Object Notation
  - **88% fewer tokens** than markdown (4K vs 34K tokens)
  - Structured tables with clear columns
  - All critical information in single file
  - Optimized for AI agent parsing

- **Markdown Format** (agent/project_details/) - Comprehensive documentation
  - Detailed explanations with rationale
  - Code examples and best practices
  - Troubleshooting guides
  - External resource links

**Choose based on your needs:**
- ✅ **Regular work, quick reference** → Use TOON (agent/quick_load/)
- ⚠️ **First-time learning, detailed examples** → Use Markdown (agent/project_details/)
