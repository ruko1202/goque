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
- **Metrics**: http://localhost:8080/metrics
- **Grafana**: http://localhost:3000 (admin/admin)
- **VictoriaMetrics**: http://localhost:8428
- **Loki**: http://localhost:3100

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

## Monitoring and Metrics

The example service includes comprehensive monitoring with Prometheus metrics, VictoriaMetrics for storage, and Grafana for visualization.

### Available Metrics

The service exposes Prometheus metrics at `/metrics` endpoint:

- **goque_processed_tasks_total** - Counter of processed tasks by type and status
- **goque_processed_tasks_with_error_total** - Counter of errors by type and operation
- **goque_task_processing_duration_seconds** - Histogram of task processing duration
- **goque_task_payload_size_bytes** - Histogram of task payload sizes

### Grafana Dashboard

Access Grafana at http://localhost:3000 (credentials: admin/admin)

The pre-configured dashboard includes:

1. **Overview Stats**:
   - Task processing rate
   - Error rate
   - P95 processing duration
   - Total tasks processed

2. **Time Series Graphs**:
   - Task processing rate by type and status
   - Processing duration percentiles (p50, p95, p99)
   - Error rate by task type and operation
   - Payload size percentiles

3. **Auto-refresh** every 10 seconds

### VictoriaMetrics

VictoriaMetrics is used as the time-series database for metrics storage:

- Web UI: http://localhost:8428
- Data retention: 30 days
- Scraped by vmagent every 10 seconds

### Architecture

```
Application (/metrics) → vmagent → VictoriaMetrics ─┐
Application (logs)     → promtail → Loki             ├─→ Grafana
                                                      │
                                                      └─→ Dashboards
```

## Log Aggregation with Loki

The example service includes centralized log aggregation using Grafana Loki.

### Log Collection

**Promtail** collects logs from the application container:
- Automatically discovers Docker containers
- Extracts JSON log fields (level, timestamp, logger) as labels
- Preserves full JSON log content for querying
- Sends to Loki for storage

### Grafana Logs Dashboard

Access the logs dashboard in Grafana (http://localhost:3000):

**"Goque Application Logs" dashboard includes**:

1. **Log Volume Graph** - Logs per minute by level (info, warn, error)
2. **Live Logs Viewer** - Real-time log stream with JSON parsing
3. **Statistics**:
   - Info logs count (5m window)
   - Warning logs count (5m window)
   - Error logs count (5m window)
   - Total logs count (5m window)
4. **Error Logs Panel** - Filtered view showing only errors

### Log Levels

The application uses structured logging with these levels:
- **info** - Normal operations, task processing
- **warn** - Warning conditions, big payloads
- **error** - Error conditions, failed operations

### Example Queries

```logql
# All logs from the service
{container="goque_example_service"} | json

# Only errors
{container="goque_example_service"} | json | level="error"

# Search for specific text
{container="goque_example_service"} | json |= "task processing"

# Count logs by level
sum by (level) (count_over_time({container="goque_example_service"} | json [5m]))
```

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
