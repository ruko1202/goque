package internalprocessors

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ruko1202/goque/internal/entity"
)

// TestDefaultStartupJitter_IsRandomized confirms the default jitter
// spreads first ticks across [0, period). Calling it 1k times and
// asserting that distinct values land in multiple buckets is enough
// to catch a regression that hard-codes the delay to 0.
func TestDefaultStartupJitter_IsRandomized(t *testing.T) {
	t.Parallel()

	const (
		period  = 100 * time.Millisecond
		samples = 1000
		buckets = 10
	)
	hits := make(map[int]int, buckets)
	for range samples {
		d := defaultStartupJitter(period)
		require.GreaterOrEqual(t, d, time.Duration(0))
		require.Less(t, d, period)
		// Bucket the result so we don't depend on exact values.
		hits[int(d*time.Duration(buckets)/period)]++
	}

	// Every bucket must see at least one sample — confirms the
	// distribution is meaningfully random, not collapsed to a corner.
	require.Len(t, hits, buckets, "jitter values must cover all buckets, got distribution: %v", hits)
}

func TestDefaultStartupJitter_ZeroPeriodReturnsZero(t *testing.T) {
	t.Parallel()
	// Defensive: a misconfigured processor with period<=0 must not
	// crash defaultStartupJitter (rand.N panics on zero argument).
	require.Equal(t, time.Duration(0), defaultStartupJitter(0))
	require.Equal(t, time.Duration(0), defaultStartupJitter(-1*time.Second))
}

// TestBaseProcessor_NonPositivePeriodExitsCleanly defends against a
// misconfigured processor (period <= 0) starting up: it must log and
// exit without panicking. Before the guard, time.NewTicker(0) would
// panic on the next line.
func TestBaseProcessor_NonPositivePeriodExitsCleanly(t *testing.T) {
	t.Parallel()

	bp := newBaseProcessor(
		"misconfig-test",
		entity.TaskType("t"),
		1*time.Second,
		0, // invalid
		func(_ context.Context, _ entity.TaskType) ([]*entity.Task, error) {
			t.Fatal("processQueueFunc must not be called on misconfigured processor")
			return nil, nil
		},
	)

	require.NotPanics(t, func() { bp.Run(t.Context()) })

	// The goroutine should close gracefulStoppedCh immediately
	// because the misconfig-guard returns before any tick loop.
	select {
	case <-bp.gracefulStoppedCh:
	case <-time.After(time.Second):
		t.Fatal("baseProcessor did not exit on non-positive period")
	}
}

// TestBaseProcessor_FirstTickHonorsJitter pins the contract that the
// jitter actually delays the first tick. We pick jitter (80ms) >>
// period (10ms) so that, without the jitter block, the first callback
// would arrive at t=period=10ms (well under the 78ms floor). With the
// jitter block honored the callback can't fire before t=jitter=80ms.
// This asymmetry is what makes the assertion discriminate "jitter on"
// vs "jitter removed" — collapse jitter to <=period and the test no
// longer differentiates the two.
func TestBaseProcessor_FirstTickHonorsJitter(t *testing.T) {
	t.Parallel()

	const (
		period = 10 * time.Millisecond
		jitter = 80 * time.Millisecond // intentionally >> period
	)

	var (
		callCount atomic.Int32
		gotFirst  = make(chan time.Time, 1)
	)
	bp := newBaseProcessor(
		"jitter-test",
		entity.TaskType("t"),
		1*time.Second,
		period,
		func(_ context.Context, _ entity.TaskType) ([]*entity.Task, error) {
			if callCount.Add(1) == 1 {
				select {
				case gotFirst <- time.Now():
				default:
				}
			}
			return nil, nil
		},
	)
	bp.startupJitter = func(time.Duration) time.Duration { return jitter }
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	start := time.Now()
	bp.Run(ctx)

	select {
	case firstCall := <-gotFirst:
		delay := firstCall.Sub(start)
		// At minimum: jitter delay must have elapsed before the
		// first call. Allow some scheduling slop on the upper bound.
		require.GreaterOrEqual(t, delay, jitter-2*time.Millisecond,
			"first call fired too early; jitter not honored (delay=%s)", delay)
	case <-time.After(jitter + 200*time.Millisecond):
		t.Fatal("first call never fired")
	}

	cancel()
	bp.Stop()
}

// TestBaseProcessor_JitterSpreadsConcurrentStarts is the
// thundering-herd regression test: N processors started in lockstep
// with the same period must NOT all fire their first tick at the same
// instant. We capture each first-tick timestamp and assert the spread
// across them exceeds 10% of the period — without jitter the spread
// would be sub-millisecond, well below this floor.
//
// Flake math: with uniform jitter ~ U[0, period) and N=20 processors,
// P(range <= 0.1 * period) = N * (0.1)^(N-1) ≈ 20 * 1e-19 ≈ 2e-18.
// Effectively zero. (At 30% threshold + N=10 the flake rate is ~2e-4,
// which is too high for a test that runs on every CI build.)
func TestBaseProcessor_JitterSpreadsConcurrentStarts(t *testing.T) {
	t.Parallel()

	const (
		numProcessors = 20
		period        = 200 * time.Millisecond
	)

	// Use the real (random) jitter for this test.
	var (
		mu       sync.Mutex
		firstAts = make([]time.Time, 0, numProcessors)
		fired    = make([]bool, numProcessors)
		wg       sync.WaitGroup
	)

	processors := make([]*baseProcessor, 0, numProcessors)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	wg.Add(numProcessors)
	for i := range numProcessors {
		bp := newBaseProcessor(
			"spread-test",
			entity.TaskType("t"),
			1*time.Second,
			period,
			func(_ context.Context, _ entity.TaskType) ([]*entity.Task, error) {
				mu.Lock()
				defer mu.Unlock()
				// Record only the first call per processor index — the
				// ticker keeps firing every `period`, we only care
				// about t0 per processor.
				if fired[i] {
					return nil, nil
				}
				fired[i] = true
				firstAts = append(firstAts, time.Now())
				wg.Done()
				return nil, nil
			},
		)
		processors = append(processors, bp)
	}

	// Start all of them as close to simultaneously as possible.
	for _, bp := range processors {
		bp.Run(ctx)
	}

	// Wait for every processor to have fired its first tick at
	// least once (or time out).
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(period + 500*time.Millisecond):
		t.Fatalf("not all processors fired in time, got %d/%d", len(firstAts), numProcessors)
	}

	cancel()
	for _, bp := range processors {
		bp.Stop()
	}

	require.Len(t, firstAts, numProcessors)

	// Compute spread between earliest and latest first-tick.
	earliest, latest := firstAts[0], firstAts[0]
	for _, t0 := range firstAts {
		if t0.Before(earliest) {
			earliest = t0
		}
		if t0.After(latest) {
			latest = t0
		}
	}
	spread := latest.Sub(earliest)

	// Without jitter the spread is sub-ms (all ticks land within
	// goroutine-scheduling noise of each other). With uniform
	// jitter across [0, 200ms] the expected spread for n=10 is
	// ~180ms; we use 60ms (30% of period) as a generous floor that
	// still differentiates "jittered" from "in-lockstep".
	require.Greater(t, spread, period/10,
		"first-tick spread too small (%s) — jitter not effective", spread)
}
