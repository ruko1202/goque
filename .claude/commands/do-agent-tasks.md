# Manage Agent Tasks (TOON Format)

command:
  name: do-agent-tasks
  purpose: Manage lifecycle of agent tasks from discovery to completion
  version: 1.0

workflow[8]{step,action,description}:
  0,Load context,Execute /load-context to prepare workspace knowledge
  1,Discover new tasks,Find new task files in agent/tasks/ directory (ignore agent/tasks/todo)
  2,Detect updated tasks,Compare task files with main branch to find modifications
  3,Read task registry,Load agent/tasks/0. tasks.toon to get current task status
  4,Update task registry,Update task statuses and add new tasks to registry
  5,Execute updated tasks,Re-run tasks that were completed but have been modified
  6,Execute new tasks,Run tasks with status NEW
  7,Mark completed,Set status to COMPLETED for successfully finished tasks

discover_new_tasks:
  action: List all *.md files in agent/tasks/ excluding 0. tasks.toon
  method: Use Glob tool with pattern agent/tasks/*.md
  filter: Exclude agent/tasks/0. tasks.toon from results
  output: List of task file paths

detect_updated_tasks:
  action: Detect tasks that were modified after completion
  primary_method: Compare file modification time with completed timestamp
  implementation:
    1. Read registry to get completed timestamps for each task
    2. Get file modification time for each task file (stat or ls -l)
    3. Compare modification time with completed timestamp from registry
    4. If file mtime > completed timestamp, mark as updated
  output: List of task file paths that were modified after completion

read_task_registry:
  file: agent/tasks/0. tasks.toon
  action: Read current task registry in TOON format
  parse: Extract task entries from tasks table with number title status created updated completed fields
  output: Structured list of known tasks with their statuses

task_statuses[4]{status,meaning,next_action}:
  NEW,Task just discovered,Execute task
  UPDATED,Task was completed but modified,Re-execute task
  COMPLETED,Task finished successfully,Skip (no action)
  IN_PROGRESS,Task currently being worked on,Continue or skip

update_task_registry:
  for_completed_but_modified:
    find: Tasks in registry with status COMPLETED that appear in detect_updated_tasks
    update: Add field updated with current date
    change_status: COMPLETED → UPDATED

  for_new_tasks:
    find: Task files from discover_new_tasks not in registry
    add_fields: created with current date, status with NEW
    append: Add new task entries to registry file

  file_format:
    structure: Use TOON table format
    entry_template: |
      tasks[N]{number,title,status,created,updated,completed}:
        {number},{title},{status},{created},{updated},{completed}

execute_tasks:
  priority[2]{order,task_type}:
    1,UPDATED tasks (re-execute modified completed tasks)
    2,NEW tasks (execute fresh tasks)

  for_each_task:
    read_task_file: Read agent/tasks/{task_number}. {task_title}.md
    parse_requirements: Extract Original Requirements section
    check_completion: Review if requirements are already satisfied
    execute_requirements: Implement each unchecked requirement
    update_task_file: Mark requirements as completed using [x] checkboxes
    update_registry: Set status to COMPLETED and add completed date

task_file_format:
  template: |
    # Task: {Title}

    ## Status: {emoji} {STATUS}

    ## Original Requirements:
    - [ ] Requirement 1
    - [ ] Requirement 2
    - [x] Completed requirement

    ## Completed Work:
    (Description of work done)

    ## Commit:
    {commit_hash} - {commit_message}

    ## Date Completed: {YYYY-MM-DD}

  status_emojis{status,emoji}:
    NEW,🆕
    UPDATED,🔄
    IN_PROGRESS,⏳
    COMPLETED,✅

execution_rules[8]{rule}:
  Follow all 23 critical rules from goque.toon
  Work on feature branch never on main
  Run make test-cov and make lint before commits
  Create separate commits for code and documentation
  Update task file with completion details
  Update task registry after each task completion
  Squash related commits per commit_squashing rules
  Mark requirements as [x] when completed

safety_checks[5]{check}:
  Verify on feature branch before starting
  Confirm task not already COMPLETED before executing
  Validate all requirements are met before marking COMPLETED
  Update both task file and registry atomically
  Commit changes after each task completion

output_format:
  summary: Provide brief summary of tasks found
  table{task,status,action}: Show task number title current_status and planned_action
  execution: Report on each task as it's executed
  completion: Confirm registry updates and final status

error_handling:
  missing_registry: Create new agent/tasks/0. tasks.toon if it doesn't exist
  invalid_format: Log warning and skip malformed task files
  execution_failure: Mark task as IN_PROGRESS and document error
  git_diff_error: Fall back to treating all non-registry files as new tasks

example_execution:
  input: User runs /do-agent-tasks
  discover: Found tasks 1 2 3 4 5
  registry: Tasks 1 2 3 completed, task 4 in registry
  git_diff: Task 2 modified since completion
  actions[3]:
    Task 2 status COMPLETED → UPDATED (modified)
    Task 4 status NEW (not in registry)
    Task 5 status NEW (not in registry)
  execute: Run task 2 4 5 in order
  update: Mark each as COMPLETED when done
