package test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ruko1202/xlog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/ruko1202/goque/internal/entity"
	"github.com/ruko1202/goque/internal/storages"
	"github.com/ruko1202/goque/internal/utils/xtime"
	"github.com/ruko1202/goque/test/testutils"
)

func TestCureTasks(t *testing.T) {
	testutils.RunMultiDBTests(t, taskStorages, testCureTasks)
}

//nolint:thelper
func testCureTasks(t *testing.T, storage storages.AdvancedTaskStorage) {
	t.Parallel()
	ctx := context.Background()

	t.Run("ok", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		task := makeTaskWithStatus(ctx, t, storage, "test cure task"+uuid.NewString(), entity.TaskStatusPending)
		task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Minute))
		updateTask(ctx, t, storage, task)

		tasks, err := storage.CureTasks(ctx, task.Type, []entity.TaskStatus{
			entity.TaskStatusPending,
		}, time.Millisecond, "comment")
		require.NoError(t, err)
		require.Len(t, tasks, 1)

		actualTask, err := storage.GetTask(ctx, task.ID)
		require.NoError(t, err)
		task.Status = entity.TaskStatusError
		task.Errors = lo.ToPtr(fmt.Sprintf("attempt %d: comment\n", task.Attempts))
		testutils.EqualTask(t, task, actualTask)
	})

	// Pins the race fix: two concurrent CureTasks calls against the same
	// stuck pool must, in total, return each task exactly once. Without
	// FOR UPDATE SKIP LOCKED on MySQL the SELECT/UPDATE pair would let
	// both healers see and cure the same row, returning it twice — and
	// the metric/log layer (base.go) would double-count it.
	t.Run("concurrent healers don't double-cure", func(t *testing.T) {
		t.Parallel()
		ctx := xlog.ContextWithLogger(ctx, xlog.NewZapAdapter(zaptest.NewLogger(t)))

		const (
			numTasks   = 20
			numHealers = 5
		)
		taskType := "concurrent cure " + uuid.NewString()

		// Seed N stuck pending tasks with updated_at well in the past.
		expectedIDs := make(map[uuid.UUID]struct{}, numTasks)
		for range numTasks {
			task := makeTaskWithStatus(ctx, t, storage, taskType, entity.TaskStatusPending)
			task.UpdatedAt = lo.ToPtr(xtime.Now().Add(-time.Hour))
			updateTask(ctx, t, storage, task)
			expectedIDs[task.ID] = struct{}{}
		}

		// Fire N healers in parallel. Each one races to the same pool.
		var (
			wg          sync.WaitGroup
			totalCured  atomic.Int64
			mu          sync.Mutex
			curedByID   = make(map[uuid.UUID]int, numTasks)
			startSignal = make(chan struct{})
		)
		wg.Add(numHealers)
		for range numHealers {
			go func() {
				defer wg.Done()
				<-startSignal // line them up at the gate
				cured, err := storage.CureTasks(ctx, taskType,
					[]entity.TaskStatus{entity.TaskStatusPending},
					time.Millisecond, "stuck",
				)
				require.NoError(t, err)
				totalCured.Add(int64(len(cured)))

				mu.Lock()
				for _, c := range cured {
					curedByID[c.ID]++
				}
				mu.Unlock()
			}()
		}
		close(startSignal)
		wg.Wait()

		// Every seeded task must be cured exactly once across all healers.
		// If FOR UPDATE SKIP LOCKED is missing on MySQL the sum exceeds
		// numTasks and at least one ID has count >1.
		require.Equal(t, int64(numTasks), totalCured.Load(),
			"sum of cured tasks across healers must equal seeded count")
		require.Len(t, curedByID, numTasks, "every seeded task must appear in some healer's result")
		for id, count := range curedByID {
			require.Equal(t, 1, count, "task %s cured %d times", id, count)
		}
	})
}
