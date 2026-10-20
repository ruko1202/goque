# Goque Performance Report

**Date:** 2026-10-20
**Code:** branch `test/fix-benchmarks` (on top of `0b5b8a9`)
**Benchmarks:** [`test/benchmark_test.go`](test/benchmark_test.go), run with `make bench BENCH_COUNT=5`

This report replaces the one from 2026-03-19. That report measured something
other than what it claimed; see [Corrections](#corrections-to-the-2026-03-19-report).

---

## Summary

- **Throughput is bound by database round trips and commit fsyncs, not by goque's
  CPU.** Processing 3.6k tasks/s on PostgreSQL keeps the client at ~0.4 of one core.
- **One processed task costs two single-row writes** (`processing`, then `done`)
  plus a share of one fetch transaction. With one worker that is ~1.0k tasks/s on
  PostgreSQL and ~350 on MySQL here; with 50 workers ~4.8k and ~2.1k.
- **MySQL deadlocks under concurrent workers.** Every full run with 10+ workers hit
  `Error 1213` on roughly one task in a few thousand. goque does not retry it, so
  the task stays in `processing` until the healer cures it, which costs an attempt.
- **`MaxIdleConns` below the number of workers reopens connections all the time.**
  On PostgreSQL with 50 workers it cut throughput from 4.8k to 1.7k tasks/s.
- **go-jet accounts for 45% of the bytes allocated per task, but only 4% of CPU.**
  It is not a throughput lever today.

SQLite has the best numbers below only because it skips the network: on this
machine every round trip to Docker costs ~0.25 ms. It is still single-writer and
single-process (see the README's SQLite caveats), not a production choice.

---

## Environment

| | |
|---|---|
| Host | Apple M4 Pro (14 cores), macOS, Go 1.25.0 |
| Docker | colima VM, **2 vCPUs**, 8 GiB |
| PostgreSQL | 17.10, `synchronous_commit=on`, driver `pgx/v5` |
| MySQL | 8.4.9, `innodb_flush_log_at_trx_commit=1`, `sync_binlog=1`, binlog on |
| SQLite | `mattn/go-sqlite3` v1.14.32, file DB, `_journal_mode=WAL&_txlock=immediate&_busy_timeout=5000`, 1 connection |
| Pools (PG, MySQL) | `MaxOpenConns=50`, `MaxIdleConns=50` |

Baselines measured with plain `database/sql`, without goque, to separate goque
from the environment:

| Probe | PostgreSQL | MySQL |
|---|---:|---:|
| `SELECT 1` round trip | 0.24 ms | 0.24 ms |
| Single-row `INSERT` (autocommit) | 0.34 ms | 1.4 ms |

MySQL's insert is slow because each commit waits for two fsyncs (redo log and binlog).

---

## What each benchmark measures

| Benchmark | One op is | Timed | Not timed |
|---|---|---|---|
| `Enqueue` | one `AddTaskToQueue` from one goroutine | the call | — |
| `EnqueueParallel` | one `AddTaskToQueue` from GOMAXPROCS goroutines | the calls | — |
| `Fetch/batch_N` | one `GetTasksForProcessing` claiming N tasks | the call | seeding the queue |
| `Processing/workers_N` | one task processed end to end, noop processor | `Goque.Run` until the last task is persisted as `done` | seeding, `Stop` |

Nothing runs in the background of `Enqueue` and `Fetch`, so their B/op and
allocs/op belong to the measured call alone. `Processing` uses a 1 ms fetch tick
(so the tick does not cap throughput), `maxTasks=100`, and moves the healer and
cleaner out of the way. Every run uses its own task type and deletes its rows.

---

## Results

Median of 5 runs, `-benchtime=1s`. The full spread is in the raw output of
`make bench`.

### Enqueue

| | latency | tasks/s | B/op | allocs/op |
|---|---:|---:|---:|---:|
| SQLite | 62 µs | 16,100 | 10.1 KB | 201 |
| PostgreSQL | 441 µs | 2,270 | 9.8 KB | 221 |
| MySQL | 1,208 µs | 830 | 9.9 KB | 195 |

### EnqueueParallel (14 goroutines)

| | tasks/s | B/op | allocs/op |
|---|---:|---:|---:|
| SQLite | 13,900 | 10.3 KB | 204 |
| PostgreSQL | 11,500 | 9.9 KB | 221 |
| MySQL | 2,290 | 9.9 KB | 195 |

### Fetch

| | batch_1 latency | batch_1 tasks/s | batch_100 latency | batch_100 tasks/s |
|---|---:|---:|---:|---:|
| SQLite | 102 µs | 9,800 | 1.5 ms | 68,700 |
| PostgreSQL | 1.56 ms | 640 | 11.3 ms | 8,800 |
| MySQL | 2.86 ms | 350 | 20.9 ms | 4,800 |

Allocations: ~21 KB / 420 allocs for one task, ~400 KB / 7,500 allocs for 100.

### Processing (end to end)

| tasks/s | 1 worker | 10 workers | 50 workers |
|---|---:|---:|---:|
| SQLite | 9,840 | 9,120 | 8,170 |
| PostgreSQL | 1,040 | 4,110 | 4,810 |
| MySQL | 350 | 1,430 | 2,110 |

Per processed task: 23–26 KB and 400–440 allocations on all three databases.

---

## Reading the numbers

**Enqueue.** On PostgreSQL 441 µs is the raw insert (0.34 ms) plus ~0.1 ms for
goque: building the SQL, spans, metrics and the driver's parameter handling. The
SQLite figure of 62 µs is the whole goque and driver path with no network, which
bounds goque's own enqueue cost from above.

**Fetch.** A claim is one transaction of four round trips: `BEGIN`, `SELECT … FOR
UPDATE SKIP LOCKED`, `UPDATE` of the claimed rows, `COMMIT`. That is why a single
task costs ~1.5 ms on PostgreSQL while a batch of 100 costs only 11 ms.

**Processing, one worker.** Each task needs two fenced single-row writes, so one
worker runs at about half the enqueue rate: PostgreSQL 958 µs/task ≈ 2 × 441 µs.

**Processing, scaling.** PostgreSQL gains 4× from 1 to 10 workers and only +17%
from 10 to 50: the database VM has 2 vCPUs. MySQL keeps scaling to 50 workers
because concurrent commits share fsyncs (group commit). SQLite does not scale and
loses a little: one connection serializes every statement, and more workers only
add contention for it.

**CPU.** In a CPU profile of `Processing/pgx/workers_10` the client was busy 37%
of wall time on one core of 14; go-jet took 4% of that and GC 10%.

---

## Findings

### 1. MySQL: deadlock between status updates (bug)

With 10 or 50 workers, about one task in a few thousand finishes with:

```
Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
```

`SHOW ENGINE INNODB STATUS` shows two `UPDATE goque_task` statements waiting on
each other in `goque_task_type_status_next_attempt_at_idx`: one holds a next-key
lock and waits for a record, the other holds that record and waits for an insert
intention lock. A status change moves the row inside that secondary index, and
under REPEATABLE READ the gap locks taken there conflict between concurrent
writers.

goque neither retries 1213 nor avoids the gap locks, so the failed write leaves
the task in `processing` until the healer cures it (one attempt consumed). Delivery
stays at-least-once, but a task can be delayed by the healer threshold and run twice.

Possible fixes: retry the statement on 1213/40001, or run goque's MySQL writes
under READ COMMITTED, which takes no gap locks. PostgreSQL and SQLite were not
affected in any run.

### 2. Connection pool: `MaxIdleConns` must cover the workers

When `MaxIdleConns` is lower than the number of connections in use at once,
`database/sql` closes every connection returned above the idle limit and opens a
new one on the next request. PostgreSQL, 50 workers, `MaxOpenConns=50`:

| `MaxIdleConns` | tasks/s | B/op | allocs/op |
|---:|---:|---:|---:|
| 25 | 1,690 | 46.0 KB | 522 |
| 50 | 4,810 | 24.6 KB | 440 |

Set `MaxIdleConns` to at least the total number of workers plus a few
connections for the fetcher, healer and cleaner. The test helpers had
`MaxOpenConns` and `MaxIdleConns` swapped; that is fixed as well.

### 3. MySQL: timestamps are rounded to the second

`goque_task.next_attempt_at`, `created_at` and `updated_at` are `TIMESTAMP`
without fractional seconds, and MySQL rounds (not truncates) the fraction. A task
enqueued at 12:00:00.6 is stored as 12:00:01 and cannot be fetched until then. The
benchmarks seed tasks with a past `next_attempt_at` to avoid it. A `TIMESTAMP(6)`
migration would fix it.

### 4. Allocations

Bytes allocated per processed task (`Processing/pgx/workers_10`, alloc_space):

| Component | Share |
|---|---:|
| go-jet SQL building | 45% |
| ↳ `strconv.syntaxError` from `jet.shouldQuoteIdentifier` | 7% |
| logging (zap + xlog), with a no-op logger | 14% |
| pgx driver | 12% |
| sqlx | 4% |

`shouldQuoteIdentifier` calls `strconv.ParseInt` on every identifier and discards
the error, allocating it each time. This is in go-jet and unchanged in its latest
release (v2.16.1). The logging share comes from `zap.Logger.clone`, which `With`
runs per operation even when the logger discards everything.

All of this costs GC time, not throughput: the client CPU is not the bottleneck
(see [CPU](#reading-the-numbers)).

---

## Corrections to the 2026-03-19 report

- **It reported enqueue latency as end-to-end processing.** `BenchmarkTaskProcessingSimple`
  timed only the `AddTaskToQueue` loop. Processing ran in the background and was
  never waited for.
- **Its B/op and allocs/op included that background processing**, because
  `ReportAllocs` counts every allocation in the process.
- **Its "throughput" was 1 / latency of a single goroutine**, not what goque can
  process.
- **SQLite's 3.2 ms was not CGO overhead.** It came from the default rollback
  journal (several fsyncs per commit) and from the single connection being shared
  with the background processor. With WAL, SQLite enqueues in 62 µs.
- **"MySQL has no `SKIP LOCKED`" is false.** MySQL 8 supports it and goque uses it.
- **The "~18% improvement" had no before/after measurement**, and its numbers did
  not match the profile files they were said to come from.
- **"No critical bottlenecks, production-ready":** the processing path was never
  run under concurrency. Running it exposed findings 1 and 2.

---

## Reproducing

```bash
make docker-up
make bench BENCH_COUNT=5 | tee new.txt          # all benchmarks, all databases
DB_DRIVER=pgx make bench BENCH=Processing       # one database, one benchmark
benchstat old.txt new.txt                       # compare two runs
./scripts/profile.sh cpu                        # CPU profile of BenchmarkProcessing
```

Absolute numbers depend on the machine and on how Docker is run; compare runs
only on the same setup. On macOS the Docker VM can answer slowly right after it
was idle: one warm-up run was 10× slower than the runs after it. Use `-count` and
benchstat, not a single run.
