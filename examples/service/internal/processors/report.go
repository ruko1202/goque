package processors

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"example/internal/models"

	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"github.com/ruko1202/goque"
)

// ReportProcessor processes report generation tasks.
type ReportProcessor struct{}

// NewReportProcessor creates a new report processor.
func NewReportProcessor() *ReportProcessor {
	return &ReportProcessor{}
}

// ProcessTask implements the TaskProcessor interface for report tasks.
func (p *ReportProcessor) ProcessTask(ctx context.Context, task *goque.Task) error {
	ctx = xlog.WithOperation(ctx, "ReportProcessor.ProcessTask",
		zap.String("task_id", task.ID.String()),
	)

	var payload models.ReportPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal report payload: %w", err)
	}

	ctx = xlog.WithFields(ctx,
		zap.String("report_type", payload.ReportType),
		zap.String("format", payload.Format),
	)

	xlog.Info(ctx, "Processing notification task")
	// Simulate long-running report generation (5-10 seconds)
	processingTime := time.Duration(5+time.Now().UnixNano()%5) * time.Second
	select {
	case <-time.After(processingTime):
		if rand.Intn(10)%3 == 0 {
			return fmt.Errorf("report processing timed out")
		}
		xlog.Info(ctx, "Report generated successfully",
			zap.Duration("processingTime", processingTime),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("report task canceled: %w", ctx.Err())
	}
}
