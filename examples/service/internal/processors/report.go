package processors

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"example/internal/models"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

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
	ctx = xlog.WithOperation(ctx, "ReportProcessor",
		xfield.String("task_id", task.ID.String()),
	)

	var payload models.ReportPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal report payload: %w", err)
	}

	ctx = xlog.WithFields(ctx,
		xfield.String("report_type", payload.ReportType),
		xfield.String("format", payload.Format),
	)

	xlog.Info(ctx, "Processing report task")
	// Simulate long-running report generation (2-10 seconds)
	processingTime := time.Duration(2_000+rand.Intn(8_000)) * time.Millisecond
	return mockProcess(ctx, "report", processingTime, 5)

}
