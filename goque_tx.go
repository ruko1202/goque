package goque

import (
	"github.com/ruko1202/goque/internal/storages/dbtx"
)

// WithTx returns a context that carries tx so that storage operations
// run inside it instead of the storage's own *sqlx.DB. This enables
// the transactional-outbox pattern: open a tx, write your domain rows,
// enqueue a task via goque, then commit — all atomically. If the tx
// rolls back, the enqueue rolls back with it.
//
// Calls that do not have a tx attached behave exactly as before and
// commit to the underlying database immediately.
//
// The caller owns the lifecycle of tx (Begin, Commit, Rollback). Goque
// only writes through it.
//
// If tx is nil, WithTx returns ctx unchanged and logs a WARN — storing
// a nil tx silently would degrade outbox guarantees to at-least-once
// with a window. To deliberately detach a tx from ctx use WithoutTx.
//
// Scope: tx-aware methods (participate in your tx)
//   - AddTaskToQueue
//   - GetTask, GetTasks
//   - UpdateTask
//   - CancelTask
//   - DeleteTasks, CureTasks  (PostgreSQL only; on MySQL/SQLite these
//     run in their own internally-managed tx for batched updates)
//
// Scope: NOT tx-aware (always run in their own internal tx)
//   - GetTasksForProcessing (uses FOR UPDATE SKIP LOCKED — must not
//     be entangled with caller's outbox tx)
//   - ResetAttempts (read+write loop with its own boundaries)
//   - DeleteTasks, CureTasks on MySQL/SQLite (see above)
//
// Do NOT use WithTx with AsyncAddTaskToQueue: the async enqueue runs
// in a goroutine that the caller does not wait on, so it races against
// the caller's Commit/Rollback and will either lose the write or panic
// on a closed tx. The async path detects this defensively, strips the
// tx, and logs a WARN — but the resulting enqueue is no longer atomic
// with your domain write. Stick to the synchronous AddTaskToQueue.
var WithTx = dbtx.WithTx

// WithoutTx returns a context with any *sqlx.Tx attached via WithTx
// removed. Use it when handing ctx to a code path that must NOT
// enroll in the caller's tx — for example, a goroutine that
// outlives the caller's Commit/Rollback (see AsyncAddTaskToQueue,
// which calls this defensively).
var WithoutTx = dbtx.WithoutTx

// TxFromContext returns the *sqlx.Tx previously attached with
// WithTx, plus a boolean indicating whether one was present.
// Useful from custom TaskProcessor implementations that need to
// enroll their own DB writes in the same tx as goque's enqueue.
var TxFromContext = dbtx.TxFromContext
