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

// WebhookProcessor processes webhook tasks.
type WebhookProcessor struct{}

// NewWebhookProcessor creates a new webhook processor.
func NewWebhookProcessor() *WebhookProcessor {
	return &WebhookProcessor{}
}

// ProcessTask implements the TaskProcessor interface for webhook tasks.
func (p *WebhookProcessor) ProcessTask(ctx context.Context, task *goque.Task) error {
	ctx = xlog.WithOperation(ctx, "ReportProcessor.ProcessTask",
		zap.String("task_id", task.ID.String()),
	)

	var payload models.WebhookPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	ctx = xlog.WithFields(ctx,
		zap.Any("payload", payload),
		zap.String("url", payload.URL),
	)

	xlog.Info(ctx, "Processing webhook task")
	// Simulate webhook call with random processing time (1-4 seconds)
	processingTime := time.Duration(1+time.Now().UnixNano()%3) * time.Second
	select {
	case <-time.After(processingTime):
		if rand.Intn(10)%3 == 0 {
			return fmt.Errorf("report processing timed out")
		}
		xlog.Info(ctx, "Webhook called successfully",
			zap.Duration("processingTime", processingTime),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("webhook task canceled: %w", ctx.Err())
	}
}
