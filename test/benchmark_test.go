package test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/ruko1202/goque"
	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/goquectx"
	"github.com/ruko1202/goque/test/testutils"
)

// toJSONBench converts object to JSON for benchmarks.
func toJSONBench(tb testing.TB, obj any) string {
	tb.Helper()
	b, err := json.Marshal(obj)
	require.NoError(tb, err)
	return string(b)
}

// pushToQueueBench adds task to queue for benchmarks.
func pushToQueueBench(ctx context.Context, tb testing.TB, queueManager goque.TaskQueueManager, task *goque.Task) {
	tb.Helper()
	err := queueManager.AddTaskToQueue(ctx, task)
	require.NoError(tb, err)
}

// BenchmarkTaskPush benchmarks only task creation and insertion into the queue.
// Measures database write performance and memory allocations.
func BenchmarkTaskPush(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkTaskPush)
}

func benchmarkTaskPush(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()
	ctx := context.Background()
	ctx = xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zap.NewNop()))

	queueManager := goque.NewTaskQueueManager(storage)
	taskType := "push_benchmark_" + uuid.NewString()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task := goque.NewTask(
			taskType,
			toJSONBench(b, fmt.Sprintf("payload_%d", i)),
		)
		pushToQueueBench(ctx, b, queueManager, task)
	}
}

// BenchmarkConcurrentTaskPush benchmarks concurrent task insertion.
// Tests lock contention and concurrent database writes.
func BenchmarkConcurrentTaskPush(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkConcurrentTaskPush)
}

func benchmarkConcurrentTaskPush(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()
	ctx := context.Background()
	ctx = xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zap.NewNop()))

	queueManager := goque.NewTaskQueueManager(storage)
	taskType := "concurrent_push_" + uuid.NewString()

	workerCount := 10
	tasksPerWorker := b.N / workerCount

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for w := 0; w < workerCount; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < tasksPerWorker; i++ {
				task := goque.NewTask(
					taskType,
					toJSONBench(b, fmt.Sprintf("worker_%d_task_%d", workerID, i)),
				)
				pushToQueueBench(ctx, b, queueManager, task)
			}
		}(w)
	}

	wg.Wait()
}

// BenchmarkTaskFetch benchmarks task fetching from the queue.
// Measures database read performance with locking (FOR UPDATE SKIP LOCKED).
func BenchmarkTaskFetch(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkTaskFetch)
}

func benchmarkTaskFetch(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()
	ctx := context.Background()
	ctx = xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zap.NewNop()))

	queueManager := goque.NewTaskQueueManager(storage)
	taskType := "fetch_benchmark_" + uuid.NewString()

	// Pre-populate queue with tasks
	for i := 0; i < b.N; i++ {
		task := goque.NewTask(
			taskType,
			toJSONBench(b, fmt.Sprintf("payload_%d", i)),
		)
		pushToQueueBench(ctx, b, queueManager, task)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tasks, err := storage.GetTasksForProcessing(ctx, taskType, 1)
		require.NoError(b, err)
		if len(tasks) > 0 {
			task := tasks[0]
			task.Status = entity.TaskStatusDone
			err = storage.UpdateTask(ctx, task.ID, task)
			require.NoError(b, err)
		}
	}
}

// BenchmarkWorkerPool benchmarks worker pool performance with different worker counts.
func BenchmarkWorkerPool(b *testing.B) {
	workerCounts := []int{1, 5, 10, 20, 50}

	for _, workerCount := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", workerCount), func(b *testing.B) {
			// nolint:thelper // Anonymous function passed to RunMultiDBBenchmarks
			testutils.RunMultiDBBenchmarks(b, taskStorages, func(b *testing.B, storage storages.AdvancedTaskStorage) {
				benchmarkWorkerPool(b, storage, workerCount)
			})
		})
	}
}

func benchmarkWorkerPool(b *testing.B, storage storages.AdvancedTaskStorage, workerCount int) {
	b.Helper()
	ctx := context.Background()
	ctx = xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zap.NewNop()))
	ctx = goquectx.WithValue(ctx, "benchmark", b.Name())

	queueManager := goque.NewTaskQueueManager(storage)
	taskType := "worker_pool_" + uuid.NewString()

	// Processor that simulates work
	processor := goque.TaskProcessorFunc(func(_ context.Context, _ *goque.Task) error {
		time.Sleep(1 * time.Millisecond) // Simulate work
		return nil
	})

	goq := goque.NewGoque(storage)
	goq.RegisterProcessor(
		taskType,
		processor,
		goque.WithWorkersCount(workerCount),
		goque.WithTaskFetcherTick(10*time.Millisecond),
		goque.WithTaskFetcherMaxTasks(100),
	)

	err := goq.Run(ctx)
	require.NoError(b, err)
	defer goq.Stop()

	// Pre-populate tasks
	for i := 0; i < b.N; i++ {
		task := goque.NewTask(
			taskType,
			toJSONBench(b, fmt.Sprintf("payload_%d", i)),
		)
		pushToQueueBench(ctx, b, queueManager, task)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Wait for all tasks to be processed
	require.Eventually(b, func() bool {
		tasks, err := storage.GetTasksForProcessing(ctx, taskType, 1)
		require.NoError(b, err)
		return len(tasks) == 0
	}, 30*time.Second, 100*time.Millisecond)
}

// BenchmarkTaskProcessingSimple benchmarks simple task processing end-to-end.
func BenchmarkTaskProcessingSimple(b *testing.B) {
	testutils.RunMultiDBBenchmarks(b, taskStorages, benchmarkTaskProcessingSimple)
}

func benchmarkTaskProcessingSimple(b *testing.B, storage storages.AdvancedTaskStorage) {
	b.Helper()
	ctx := context.Background()
	ctx = xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zap.NewNop()))
	ctx = goquectx.WithValue(ctx, "benchmark", b.Name())

	queueManager := goque.NewTaskQueueManager(storage)
	taskType := "simple_benchmark_" + uuid.NewString()

	// Create Goque instance with noop processor
	goq := goque.NewGoque(storage)
	goq.RegisterProcessor(
		taskType,
		goque.NoopTaskProcessor(),
		goque.WithWorkersCount(10),
		goque.WithTaskFetcherTick(10*time.Millisecond),
		goque.WithTaskFetcherMaxTasks(100),
	)

	err := goq.Run(ctx)
	require.NoError(b, err)
	defer goq.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task := goque.NewTask(
			taskType,
			toJSONBench(b, fmt.Sprintf("payload_%d", i)),
		)
		pushToQueueBench(ctx, b, queueManager, task)
	}

	// Give some time for processing
	time.Sleep(100 * time.Millisecond)
}
