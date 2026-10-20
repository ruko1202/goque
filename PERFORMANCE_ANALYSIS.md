# Goque Performance Report

**Date:** 2026-10-20
**Code:** branch `test/fix-benchmarks`, including the MySQL fixes from findings 1 and 3
**Benchmarks:** [`test/benchmark_test.go`](test/benchmark_test.go), run with `make bench BENCH_COUNT=5`

This report replaces the one from 2026-03-19. That report measured something
other than what it claimed; see [Corrections](#corrections-to-the-2026-03-19-report).

---

## Summary

- **Throughput is bound by database round trips and commit fsyncs, not by goque's
  CPU.** Processing 3.6k tasks/s on PostgreSQL keeps the client at ~0.4 of one core.
- **One processed task costs two single-row writes** (`processing`, then `done`)
  plus a share of one fetch transaction. With one worker that is ~1.0k tasks/s on
  PostgreSQL and ~380 on MySQL here; with 50 workers ~4.8k and ~1.9k.
- **MySQL deadlocked under concurrent workers; fixed.** With 10+ workers about one
  task in a few thousand failed with `Error 1213` and waited for the healer. goque's
  own MySQL transactions now run at READ COMMITTED, and no run since has failed.
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
`make bench`. MySQL rows were measured after the fixes in findings 1 and 3.
Against the run before them, some MySQL medians moved by 25–95% in both
directions, including `EnqueueParallel`, whose code did not change: on this
setup that is noise, so do not read the difference as a speed-up or slowdown.

### Enqueue

| | latency | tasks/s | B/op | allocs/op |
|---|---:|---:|---:|---:|
| SQLite | 62 µs | 16,100 | 10.1 KB | 201 |
| PostgreSQL | 441 µs | 2,270 | 9.8 KB | 221 |
| MySQL | 1,256 µs | 800 | 9.9 KB | 195 |

### EnqueueParallel (14 goroutines)

| | tasks/s | B/op | allocs/op |
|---|---:|---:|---:|
| SQLite | 13,900 | 10.3 KB | 204 |
| PostgreSQL | 11,500 | 9.9 KB | 221 |
| MySQL | 4,450 | 10.0 KB | 195 |

### Fetch

| | batch_1 latency | batch_1 tasks/s | batch_100 latency | batch_100 tasks/s |
|---|---:|---:|---:|---:|
| SQLite | 102 µs | 9,800 | 1.5 ms | 68,700 |
| PostgreSQL | 1.56 ms | 640 | 11.3 ms | 8,800 |
| MySQL | 3.58 ms | 280 | 21.0 ms | 4,760 |

Allocations: ~21 KB / 420 allocs for one task, ~400 KB / 7,500 allocs for 100.

### Processing (end to end)

| tasks/s | 1 worker | 10 workers | 50 workers |
|---|---:|---:|---:|
| SQLite | 9,840 | 9,120 | 8,170 |
| PostgreSQL | 1,040 | 4,110 | 4,810 |
| MySQL | 380 | 1,070 | 1,920 |

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

### 1. MySQL: deadlock between status updates (fixed)

With 10 or 50 workers, about one task in a few thousand finished with:

```
Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
```

`SHOW ENGINE INNODB STATUS` showed two `UPDATE goque_task` statements waiting on
each other in `goque_task_type_status_next_attempt_at_idx`. Under REPEATABLE READ
the fetch's `SELECT … FOR UPDATE SKIP LOCKED` range scan also next-key-locks the
first index record past the range, which belongs to a task a worker is moving to
`processing` or `done` at that moment. The worker's `UPDATE` waited for that
record while the fetch's batch `UPDATE` waited for an insert intention lock in the
worker's gap. Nothing retried the error, so the task stayed in `processing` until
the healer cured it, consuming an attempt.

**Fix:** goque's own MySQL transactions (fetch, healer, cleaner, `ResetAttempts`)
now run at READ COMMITTED, which locks only the rows a scan returns. Version
fencing is unchanged. Before the fix, `-benchtime=4s -count=8` failed 5 tasks in
~135k; after it, `-count=12` failed none in ~200k, and the MySQL numbers above
had no failures either. The server needs `binlog_format` `ROW` (the default) or
`MIXED`: with `STATEMENT`, MySQL rejects writes in a READ COMMITTED transaction
(`Error 1665`). PostgreSQL and SQLite were not affected.

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

### 3. MySQL: timestamps were rounded to the second (fixed)

`goque_task.next_attempt_at`, `created_at` and `updated_at` were `TIMESTAMP`
without fractional seconds, and MySQL rounds (not truncates) the fraction. A task
enqueued at 12:00:00.6 was stored as 12:00:01 and could not be fetched until then.
Migration `20261020120000_task_timestamp_precision` changes them to
`TIMESTAMP(6)`; a storage test now fetches a task enqueued with
`next_attempt_at = now` at once.

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
