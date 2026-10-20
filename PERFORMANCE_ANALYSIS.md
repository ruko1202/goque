# Goque Performance Report

*2026-10-20 · branch `test/fix-benchmarks` · [`test/benchmark_test.go`](test/benchmark_test.go) · replaces the 2026-03-19 report ([what was wrong with it](#a4-corrections-to-the-2026-03-19-report))*

**Bottom line:** goque's own overhead is small; the database sets the speed.
Each processed task costs two single-row writes plus a share of one fetch, so
throughput is bound by database round trips and commit fsyncs, not by goque's
CPU. On a laptop with Docker databases, PostgreSQL processes ~4.8k tasks/s with
50 workers while the client uses ~0.4 of one core. Running the benchmarks under
concurrency found two MySQL bugs, both fixed.

| | Enqueue latency | Processing, 1 worker | 10 workers | 50 workers |
|---|---:|---:|---:|---:|
| **PostgreSQL** | 0.44 ms | 1,040 tasks/s | 4,110 | 4,810 |
| **MySQL** | 1.26 ms | 380 tasks/s | 1,070 | 1,920 |
| **SQLite** | 0.06 ms | 9,840 tasks/s | 9,120 | 8,170 |

SQLite leads only because it skips the network (~0.25 ms per round trip to
Docker here). It is single-writer and single-process, not a production choice.
Absolute numbers are specific to this machine ([environment](#a1-environment-and-baselines));
compare them only against runs on the same setup.

---

## What to do

1. **Size the connection pool for your workers.** Keep `MaxIdleConns` at least at
   the total number of workers plus a few for the fetcher, healer and cleaner.
   Below that, `database/sql` keeps closing and reopening connections: on
   PostgreSQL with 50 workers throughput fell from 4.8k to 1.7k tasks/s. It
   mattered more than any other setting measured here.
2. **Add workers until the database saturates, not beyond.** PostgreSQL gained 4×
   up to 10 workers and only 17% more at 50; past that point more workers only add
   load on the database. Where that point lies depends on the database server, so
   measure it on yours.
3. **Leave allocations alone for now.** go-jet and logging dominate the bytes
   allocated per task but not CPU ([allocations](#allocations)); reducing them
   would ease GC, not raise throughput.
4. **If throughput must grow, cut round trips per task.** Merging the `processing`
   and `done` writes, or acknowledging finished tasks in batches, are the levers
   that would move these numbers. This is a hypothesis, not measured, and both
   change how version fencing and the healer see a running task.

---

## Why the numbers look like this

**Enqueue is one INSERT.** On PostgreSQL 441 µs is the raw insert (0.34 ms) plus
~0.1 ms for goque: building the SQL, spans, metrics and the driver's parameter
handling. SQLite's 62 µs is the whole goque and driver path with no network,
which bounds goque's own enqueue cost from above.

**Processing is two writes per task.** Each task needs two fenced single-row
writes, so one worker runs at about half the enqueue rate: PostgreSQL 958 µs/task
≈ 2 × 441 µs.

**Fetching in batches is nearly free per task.** A claim is one transaction of
four round trips (`BEGIN`, `SELECT … FOR UPDATE SKIP LOCKED`, `UPDATE`, `COMMIT`),
so one task costs ~1.5 ms on PostgreSQL and a batch of 100 only 11 ms.

**Scaling stops at the database.** PostgreSQL gains 4× from 1 to 10 workers and
+17% from 10 to 50: the database VM has 2 vCPUs. MySQL keeps scaling to 50
workers because concurrent commits share fsyncs (group commit); each single
commit waits for two fsyncs (redo log and binlog), which is why MySQL is slowest.
SQLite does not scale: one connection serializes every statement.

**The client is mostly idle.** In a CPU profile of `Processing/pgx/workers_10` the
client was busy 37% of wall time on one core of 14; go-jet took 4% of that and GC
10%.

---

## Bugs the benchmarks found

| Bug | Impact | Status |
|---|---|---|
| MySQL deadlock between status updates (`Error 1213`) | ~1 task in a few thousand with 10+ workers sat in `processing` until the healer cured it, losing an attempt | **Fixed:** goque's MySQL transactions run at READ COMMITTED |
| MySQL timestamps rounded to the second | a new task could not be fetched for up to 0.5 s | **Fixed:** migration to `TIMESTAMP(6)` |
| Test pool had `MaxOpenConns` and `MaxIdleConns` swapped | benchmarks ran with the wrong pool | **Fixed** |

<details>
<summary>MySQL deadlock: cause and verification</summary>

`SHOW ENGINE INNODB STATUS` showed two `UPDATE goque_task` statements waiting on
each other in `goque_task_type_status_next_attempt_at_idx`. Under REPEATABLE READ
the fetch's `SELECT … FOR UPDATE SKIP LOCKED` range scan also next-key-locks the
first index record past the range, which belongs to a task a worker is moving to
`processing` or `done` at that moment. The worker's `UPDATE` waited for that
record while the fetch's batch `UPDATE` waited for an insert intention lock in the
worker's gap. Nothing retried the error.

goque's own MySQL transactions (fetch, healer, cleaner, `ResetAttempts`) now run
at READ COMMITTED, which locks only the rows a scan returns. Version fencing is
unchanged. Before the fix, `-benchtime=4s -count=8` failed 5 tasks in ~135k;
after it, `-count=12` failed none in ~200k, and the MySQL numbers in this report
had no failures either. The server needs `binlog_format` `ROW` (the default) or
`MIXED`: with `STATEMENT`, MySQL rejects writes in a READ COMMITTED transaction
(`Error 1665`). PostgreSQL and SQLite were not affected.

</details>

<details>
<summary>MySQL timestamps: cause and fix</summary>

`next_attempt_at`, `created_at` and `updated_at` were `TIMESTAMP` without
fractional seconds, and MySQL rounds (not truncates) the fraction: a task enqueued
at 12:00:00.6 was stored as 12:00:01 and could not be fetched until then.
Migration `20261020120000_task_timestamp_precision` changes them to
`TIMESTAMP(6)`; a storage test now fetches a task enqueued with
`next_attempt_at = now` at once.

</details>

---

## Detailed results

Median of 5 runs, `-benchtime=1s`; the full spread is in the raw output of
`make bench`. MySQL rows were measured after both MySQL fixes. Against the run
before them, some MySQL medians moved by 25–95% in both directions, including
`EnqueueParallel`, whose code did not change: on this setup that is noise, so do
not read the difference as a speed-up or slowdown.

### Processing (end to end)

| tasks/s | 1 worker | 10 workers | 50 workers |
|---|---:|---:|---:|
| SQLite | 9,840 | 9,120 | 8,170 |
| PostgreSQL | 1,040 | 4,110 | 4,810 |
| MySQL | 380 | 1,070 | 1,920 |

Per processed task: 23–26 KB and 400–440 allocations on all three databases.

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

### Connection pool

PostgreSQL, 50 workers, `MaxOpenConns=50`:

| `MaxIdleConns` | tasks/s | B/op | allocs/op |
|---:|---:|---:|---:|
| 25 | 1,690 | 46.0 KB | 522 |
| 50 | 4,810 | 24.6 KB | 440 |

When `MaxIdleConns` is lower than the number of connections in use at once,
`database/sql` closes every connection returned above the idle limit and opens a
new one on the next request.

### Allocations

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
runs per operation even when the logger discards everything. All of this costs GC
time, not throughput: the client CPU is not the bottleneck.

---

## Appendix

### A1. Environment and baselines

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

### A2. What each benchmark measures

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

### A3. Reproducing

```bash
make docker-up
make bench BENCH_COUNT=5 | tee new.txt          # all benchmarks, all databases
DB_DRIVER=pgx make bench BENCH=Processing       # one database, one benchmark
benchstat old.txt new.txt                       # compare two runs
./scripts/profile.sh cpu                        # CPU profile of BenchmarkProcessing
```

On macOS the Docker VM can answer slowly right after it was idle: one warm-up run
was 10× slower than the runs after it. Use `-count` and benchstat, not a single
run.

### A4. Corrections to the 2026-03-19 report

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
  run under concurrency. Running it found the MySQL deadlock and the pool issue.
