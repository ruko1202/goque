package test

// Benchmarks for the three stages a task goes through: enqueue, claim (fetch)
// and end-to-end processing.
//
// Each benchmark times only its own stage. Nothing runs in the background of
// the enqueue and fetch benchmarks, so their ns/op, B/op and allocs/op belong
// to the measured call alone. The processing benchmark times from Run until the
// last task's final status is persisted, so its numbers are per processed task
// and include fetch, both status writes and the worker pool.
//
// Every run uses its own task type and deletes its rows afterwards, so runs
// never fetch each other's tasks and the table does not grow between them.
//
//	make bench                                  # all benchmarks, all DBs
//	DB_DRIVER=pgx make bench BENCH=Processing   # one DB, one benchmark

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/storages/dbtx"
	"github.com/ruko1202/goque/test/testutils"
)

const (
	benchPayload = `{"user_id":42,"action":"send_email"}`

	// benchSeedChunk is how many tasks are inserted per transaction when a
	// benchmark pre-fills the queue: one commit per chunk instead of per task.
	benchSeedChunk = 1000

	// Processing benchmark settings. The tick is short so the fetch interval
	// does not cap throughput; maxTasks is the library default.
	benchFetchTick         = time.Millisecond
	benchFetchMaxTasks     = int64(100)
	benchProcessingTimeout = 5 * time.Minute
)

var (
	benchFetchBatches = []int{1, 100}
	benchWorkerCounts = []int{1, 10, 50}

	benchAllStatuses = []entity.TaskStatus{
		entity.TaskStatusNew,
		entity.TaskStatusPending,
		entity.TaskStatusProcessing,
		entity.TaskStatusDone,
		entity.TaskStatusCanceled,
		entity.TaskStatusError,
		entity.TaskStatusAttemptsLeft,
	}
)

// BenchmarkEnqueue measures one AddTaskToQueue call from a single goroutine:
// the latency of a single INSERT round trip plus goque's own overhead.
func BenchmarkEnqueue(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkEnqueue)
}

func benchmarkEnqueue(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()

	ctx := benchContext()
	queueManager := goque.NewTaskQueueManager(storage)
	taskType := benchTaskType(b, storage)

	b.ReportAllocs()
	for b.Loop() {
		if err := queueManager.AddTaskToQueue(ctx, goque.NewTask(taskType, benchPayload)); err != nil {
			b.Fatal(err)
		}
	}
	reportTasksPerSecond(b, b.N)
}

// BenchmarkEnqueueParallel measures AddTaskToQueue from GOMAXPROCS goroutines,
// which is how a service with concurrent request handlers enqueues.
// tasks/s is the total insert throughput the database sustains.
func BenchmarkEnqueueParallel(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkEnqueueParallel)
}

func benchmarkEnqueueParallel(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()

	ctx := benchContext()
	queueManager := goque.NewTaskQueueManager(storage)
	taskType := benchTaskType(b, storage)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if err := queueManager.AddTaskToQueue(ctx, goque.NewTask(taskType, benchPayload)); err != nil {
				b.Error(err)
				return
			}
		}
	})
	reportTasksPerSecond(b, b.N)
}

// BenchmarkFetch measures one GetTasksForProcessing call claiming a batch of
// tasks from a pre-filled queue. One op is one call; tasks/s is the claim rate.
func BenchmarkFetch(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, func(b *testing.B, storage storages.AdvancedTaskStorage) {
		b.Helper()

		for _, batch := range benchFetchBatches {
			b.Run(fmt.Sprintf("batch_%d", batch), func(b *testing.B) {
				benchmarkFetch(b, storage, batch)
			})
		}
	})
}

func benchmarkFetch(b *testing.B, storage storages.AdvancedTaskStorage, batch int) {
	b.Helper()

	ctx := benchContext()
	taskType := benchTaskType(b, storage)
	seedTasks(ctx, b, storage, taskType, b.N*batch)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		tasks, err := storage.GetTasksForProcessing(ctx, taskType, int64(batch))
		if err != nil {
			b.Fatal(err)
		}
		if len(tasks) != batch {
			b.Fatalf("fetched %d tasks, want %d", len(tasks), batch)
		}
	}
	b.StopTimer()
	reportTasksPerSecond(b, b.N*batch)
}

// BenchmarkProcessing measures end-to-end throughput: b.N tasks are already in
// the queue, the timer runs from Goque.Run until every task is persisted as
// done. The processor does no work, so this is goque's own ceiling per worker
// count: fetch, the processing and done writes, hooks and the worker pool.
func BenchmarkProcessing(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, func(b *testing.B, storage storages.AdvancedTaskStorage) {
		b.Helper()

		for _, workers := range benchWorkerCounts {
			b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) {
				benchmarkProcessing(b, storage, workers)
			})
		}
	})
}

func benchmarkProcessing(b *testing.B, storage storages.AdvancedTaskStorage, workers int) {
	b.Helper()

	ctx := benchContext()
	taskType := benchTaskType(b, storage)
	seedTasks(ctx, b, storage, taskType, b.N)

	var (
		finished atomic.Int64
		failed   atomic.Int64
		firstErr atomic.Pointer[error]
		allDone  = make(chan struct{})
	)

	goq := goque.NewGoque(storage)
	goq.RegisterProcessor(
		taskType,
		goque.NoopTaskProcessor(),
		goque.WithWorkersCount(goque.NewStaticValue(workers)),
		goque.WithTaskFetcherTick(goque.NewStaticValue(benchFetchTick)),
		goque.WithTaskFetcherMaxTasks(goque.NewStaticValue(benchFetchMaxTasks)),
		// Keep the healer and cleaner out of the measurement.
		goque.WithHealerPeriod(goque.NewStaticValue(time.Hour)),
		goque.WithCleanerPeriod(goque.NewStaticValue(time.Hour)),
		// The after-hook runs once the task's final status is persisted, so
		// the b.N-th call is the moment the whole queue is processed.
		goque.WithHooksAfterProcessing(func(_ context.Context, _ *goque.Task, err error) {
			if err != nil {
				failed.Add(1)
				firstErr.CompareAndSwap(nil, &err)
			}
			if finished.Add(1) == int64(b.N) {
				close(allDone)
			}
		}),
	)

	b.ReportAllocs()
	b.ResetTimer()
	require.NoError(b, goq.Run(ctx))
	// Registered after benchTaskType, so it runs before the rows are deleted.
	b.Cleanup(goq.Stop)

	select {
	case <-allDone:
	case <-time.After(benchProcessingTimeout):
		b.Fatalf("processed %d of %d tasks in %s", finished.Load(), b.N, benchProcessingTimeout)
	}
	b.StopTimer()

	// A failed task is reported, not fatal: the run still measures throughput,
	// and the failures (e.g. MySQL deadlocks under many workers) stay visible.
	if n := failed.Load(); n > 0 {
		b.Logf("%d of %d tasks finished with an error, first: %v", n, b.N, *firstErr.Load())
	}
	b.ReportMetric(float64(failed.Load()), "failed")
	reportTasksPerSecond(b, b.N)
}

func benchContext() context.Context {
	return xlog.ContextWithLogger(context.Background(), xlog.NewZapAdapter(zap.NewNop()))
}

// benchTaskType returns a task type unique to this run and deletes its tasks
// once the run finishes.
func benchTaskType(b *testing.B, storage storages.AdvancedTaskStorage) entity.TaskType {
	b.Helper()

	taskType := "bench_" + uuid.NewString()
	b.Cleanup(func() {
		_, err := storage.DeleteTasks(benchContext(), taskType, benchAllStatuses, 0)
		require.NoError(b, err)
	})

	return taskType
}

// seedTasks inserts n due tasks of taskType, benchSeedChunk per transaction.
func seedTasks(ctx context.Context, b *testing.B, storage storages.AdvancedTaskStorage, taskType entity.TaskType, n int) {
	b.Helper()

	for seeded := 0; seeded < n; seeded += benchSeedChunk {
		chunk := min(benchSeedChunk, n-seeded)
		err := dbtx.WithinTx(ctx, storage.GetDB(), func(ctx context.Context) error {
			for range chunk {
				if err := storage.AddTask(ctx, entity.NewTask(taskType, benchPayload)); err != nil {
					return err
				}
			}
			return nil
		})
		require.NoError(b, err)
	}
}

func reportTasksPerSecond(b *testing.B, tasks int) {
	b.Helper()

	if elapsed := b.Elapsed(); elapsed > 0 {
		b.ReportMetric(float64(tasks)/elapsed.Seconds(), "tasks/s")
	}
}
