# Do agent tasks(TOON Format)

tasks_dir: agent/tasks
task_file: {tasks_dir}/0. tasks.toon

command:
    name: do-agent-tasks
    purpose: Do agent tasks from {tasks_dir}
    version: 1.0

workflow:
    - read_tasks
    - check_changed_processed_tasks
    - process_tasks


actions:
    read_tasks:
        - read task_file
        - make a plan for processing uncompleted task
        - do process_tasks action
    process_tasks:
        - processed tasks
        - mark processed tasks as `COMPLETED` in task_file
    check_changed_processed_tasks:
        - check changed files in tasks_dir with tasks is marked `COMPLETED`
        - do process_tasks