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
	ctx = xlog.WithOperation(ctx, "EmailProcessor",
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
	// Simulate email sending with random processing time (100ms-3 seconds)
	processingTime := time.Duration(100+rand.Intn(2_900)) * time.Millisecond
	return mockProcess(ctx, "email notification", processingTime, 25)
}
