# Grafana dashboards

A ready-to-import Grafana dashboard for monitoring a Goque task queue.

| File | Dashboard | Data source |
|------|-----------|-------------|
| [`goque-metrics.json`](goque-metrics.json) | **Goque Queue Metrics** — task throughput, latency, retries, payload size, workers | Prometheus |

## Import

1. In Grafana: **Dashboards → New → Import**.
2. Upload the JSON file (or paste its contents).
3. When prompted, pick your **Prometheus** data source.

That's it — the dashboard uses template variables and discovers values from your data automatically.

## Metrics dashboard at a glance

**Template variables** (top of the dashboard):

- `$service` — service name (from the `service` label on `goque_*` metrics).
- `$processor` — task type / processor to focus on.
- `$rate_interval` — window for rate/quantile calculations (`10s … 10m`).

**Rows and panels:**

- **Common** — fleet-wide view across all processors:
  - New rate, Success, Error rate, Payload Decode Errors (stats)
  - Processing Duration q95 (stat)
  - Task Processing (throughput by status) and Task Processing Duration q95 (timeseries)
- **Processor: `$processor`** — drill-down for one task type:
  - State (new / done / errored), Retry attempts q95, Workers
  - Task Processing, Task Processing Duration q95, Task Payload Size q95
- **Internal processes** — operations performed by internal processors (cleaner, healer, etc.).

All q95 latency/size panels compute `histogram_quantile` over `rate(..._bucket[$rate_interval])`, so they reflect the selected window rather than counters accumulated since process start. Task types with no recent activity therefore read empty instead of showing a stale value.

## Metrics reference

The dashboards are built on the Prometheus metrics Goque exports (`goque_processed_tasks_total`, `goque_task_processing_duration_seconds`, `goque_task_retry_attempts`, `goque_task_payload_size_bytes`, `goque_processors_workers_count`, `goque_operations_total`, `goque_payload_decode_errors_total`). See the **Monitoring** section of the [top-level README](../README.md) for how to expose them.

## Try it locally

A full docker-compose stack (Goque example service + Prometheus + Grafana with this dashboard pre-provisioned) lives under [`examples/service`](../examples/service). From that directory:

```bash
docker compose up
```

Grafana comes up with the dashboard already loaded.

> **Note:** the copy pre-provisioned into the demo lives at
> `examples/service/monitoring/grafana/dashboards/goque-metrics.json`. The file
> here is the import-friendly copy referenced from the docs; when you change the
> dashboard, update both so they stay in sync.
