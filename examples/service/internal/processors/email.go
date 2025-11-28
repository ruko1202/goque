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

// EmailProcessor processes email sending tasks.
type EmailProcessor struct{}

// NewEmailProcessor creates a new email processor.
func NewEmailProcessor() *EmailProcessor {
	return &EmailProcessor{}
}

// ProcessTask implements the TaskProcessor interface for email tasks.
func (p *EmailProcessor) ProcessTask(ctx context.Context, task *goque.Task) error {
	ctx = xlog.WithOperation(ctx, "EmailProcessor.ProcessTask",
		zap.String("task_id", task.ID.String()),
	)
	var payload models.EmailPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal email payload: %w", err)
	}

	ctx = xlog.WithFields(ctx,
		zap.Any("payload", payload),
		zap.String("subject", payload.Subject),
		zap.String("to", payload.To),
	)

	xlog.Info(ctx, "Processing email task")
	// Simulate email sending with random processing time (1-3 seconds)
	processingTime := time.Duration(1+time.Now().UnixNano()%2) * time.Second
	select {
	case <-time.After(processingTime):
		if rand.Intn(10)%3 == 0 {
			return fmt.Errorf("notification processing timed out")
		}
		xlog.Info(ctx, "Email sent successfully",
			zap.Duration("processingTime", processingTime),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("email task canceled: %w", ctx.Err())
	}
}
