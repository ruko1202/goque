package processors

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

func mockProcess(ctx context.Context, process string, processingTime time.Duration, errorRatePercent int64) error {
	select {
	case <-time.After(processingTime):
		if rand.Int63n(100) < errorRatePercent {
			return fmt.Errorf("%s processing timed out", process)
		}
		xlog.Info(ctx, "process finish successfully",
			zap.Duration("processingTime", processingTime),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("%s canceled: %w", process, ctx.Err())
	}
}
