# Load Goque Project Context

Load the full project context by reading all agent documentation files in parallel.

Read the following files from the agent/ directory using multiple Read tool calls in a single message:

**Core Documentation (MUST READ):**
- agent/README.md - Navigation hub and quick start guide
- agent/overview.md - Project purpose, goals, and high-level overview
- agent/architecture.md - Technical architecture, components, and design patterns
- agent/critical-rules.md - 21 non-negotiable rules (SQL injection, error handling, etc.)
- agent/conventions.md - Code style, naming patterns, and interface design
- agent/principles.md - 26 design principles (reliability, simplicity, composability)

**Workflow & Process:**
- agent/workflow.md - Development workflow, Git processes, testing
- agent/checklist.md - Development and PR checklist
- agent/commits.md - Commit message conventions

**Reference:**
- agent/tech-stack.md - Technologies, libraries, and tools used
- agent/known-issues.md - Known issues and workarounds
- agent/links.md - Useful links and resources

After reading, provide a brief confirmation that context has been loaded and you understand:
- Project purpose and goals
- Critical development rules
- Architecture and design patterns
- Development workflow and standards
