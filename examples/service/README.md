# Goque Example Service

A production-quality example service demonstrating real-world usage of the Goque task queue library with best practices.

## Quick Start with Docker

The fastest way to run the example:

```bash
# Start all services (PostgreSQL + Application)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

Once running, open your browser to:
- **Dashboard**: http://localhost:8080
- **API**: http://localhost:8080/api/tasks
- **Health**: http://localhost:8080/health

## Task Types

The example implements four different task types to demonstrate various use cases:

### 1. Email Tasks
Simulates email sending with retry logic (1-3 seconds processing time)

```json
{
  "type": "email",
  "payload": {
    "to": "user@example.com",
    "subject": "Test Email",
    "body": "Email content"
  }
}
```

### 2. Notification Tasks
Push notifications to users (500ms-2s processing time)

```json
{
  "type": "notification",
  "payload": {
    "user_id": "user-123",
    "title": "New Message",
    "message": "You have a new notification"
  }
}
```

### 3. Report Tasks
Long-running report generation (5-10 seconds processing time)

```json
{
  "type": "report",
  "payload": {
    "report_type": "sales",
    "date_from": "2024-01-01",
    "date_to": "2024-01-31",
    "format": "pdf"
  }
}
```

### 4. Webhook Tasks
External API integration (1-4 seconds processing time)

```json
{
  "type": "webhook",
  "payload": {
    "url": "https://api.example.com/webhook",
    "method": "POST",
    "headers": {
      "Content-Type": "application/json"
    },
    "body": "{\"event\": \"test\"}"
  }
}
```

## Web Dashboard

The dashboard provides:

- **Real-time Updates** - Auto-refreshes every 5 seconds
- **Status Filters** - Filter by task status (new, pending, processing, done, error, canceled)
- **Type Filters** - Filter by task type (email, notification, report, webhook)
- **Pagination** - Navigate through large task lists
- **Task Creation** - Create new tasks directly from the UI
- **Statistics** - View task counts by status

## Task Generator

The built-in task generator creates random tasks periodically to demonstrate the queue in action:

- Generates 1-5 random tasks every 10 seconds (configurable)
- Creates tasks of all types with realistic payloads
- Can be enabled/disabled via configuration

## Configuration

Configuration can be provided via:

1. **config.yaml file** (in current directory or ./config/)
2. **Environment variables** with `GOQUE_` prefix

Example environment variables:

```bash
GOQUE_SERVER_HOST=localhost
GOQUE_SERVER_PORT=8080
GOQUE_DATABASE_DRIVER=postgres
GOQUE_DATABASE_DSN="postgres://user:pass@localhost/db"
GOQUE_QUEUE_WORKERS=5
GOQUE_QUEUE_MAXATTEMPTS=3
GOQUE_TASKGENERATOR_ENABLED=true
GOQUE_TASKGENERATOR_INTERVAL=10s
```

## License

This example is part of the Goque project and uses the same license.
