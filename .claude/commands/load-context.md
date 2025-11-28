# Load Goque Project Context (TOON Format)

command:
  name: load-context
  purpose: Load project context using token-efficient TOON format
  version: 2.0

default_method:
  name: TOON Format
  file: agent/quick_load/goque.toon
  action: Read agent/quick_load/goque.toon

  metrics{metric,value}:
    Token savings,88% fewer tokens
    Token count,~4K tokens vs ~34K markdown
    File count,1 file vs 13 markdown files
    Load speed,Single read operation
    Coverage,100% critical information

  benefits[5]:
    Single file with all essential information
    Structured tables for easy AI parsing
    All 23 critical rules with rationale
    All 26 design principles with implementation
    Complete workflows and safety checklists

  contains[11]:
    Project overview features and architecture (5 layers)
    All 23 critical rules with rationale
    All 26 design principles with implementation notes
    7 task statuses + 8 state transitions with triggers
    Tech stack (3 databases 8 dependencies 5 dev tools)
    Naming conventions for all code elements
    Error handling database and testing rules
    Branch strategy workflow and commit conventions
    12 make commands with descriptions
    Safety checklist (13 checks) and never-do list (12 items)
    Quick start guide for AI agents (5 steps)

alternative_method:
  name: Detailed Markdown
  location: agent/project_details/
  action: Read multiple files from agent/project_details/ in parallel

  use_when[5]:
    First-time deep dive into the project
    Need detailed code examples with explanations
    Troubleshooting specific problems
    Need external resources and learning materials
    Want detailed workflow checklists

  core_files[6]{file,description}:
    README.md,Navigation hub and quick start guide
    overview.md,Project purpose goals and high-level overview
    architecture.md,Technical architecture components and design patterns
    critical-rules.md,23 non-negotiable rules with detailed explanations
    conventions.md,Code style naming patterns and interface design
    principles.md,26 design principles with rationale

  workflow_files[4]{file,description}:
    workflow.md,Development workflow Git processes testing
    checklist.md,Development and PR checklist
    commits.md,Commit message conventions
    branch-naming.md,Branch naming rules and strategies (max 3 words)

  reference_files[3]{file,description}:
    tech-stack.md,Technologies libraries and tools used
    known-issues.md,Known issues and workarounds
    links.md,Useful links and resources

recommendation:
  default: Use TOON format (agent/quick_load/goque.toon)
  reason: 88% token savings with 100% critical information
  switch_to_markdown_when: First time learning or need detailed examples

confirmation_required:
  after_loading: Provide brief confirmation that context is loaded
  must_understand[4]:
    Project purpose and goals
    Critical development rules (23 rules)
    Architecture and design patterns
    Development workflow and standards
