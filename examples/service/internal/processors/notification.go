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

// NotificationProcessor processes notification tasks.
type NotificationProcessor struct{}

// NewNotificationProcessor creates a new notification processor.
func NewNotificationProcessor() *NotificationProcessor {
	return &NotificationProcessor{}
}

// ProcessTask implements the TaskProcessor interface for notification tasks.
func (p *NotificationProcessor) ProcessTask(ctx context.Context, task *goque.Task) error {
	ctx = xlog.WithOperation(ctx, "NotificationProcessor.ProcessTask",
		zap.String("task_id", task.ID.String()),
	)

	var payload models.NotificationPayload
	if err := json.Unmarshal([]byte(task.Payload), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal notification payload: %w", err)
	}

	ctx = xlog.WithFields(ctx,
		zap.String("user_id", payload.UserID),
		zap.String("title", payload.Title),
	)

	xlog.Info(ctx, "Processing notification task")
	// Simulate notification sending with random processing time (500ms-2s)
	processingTime := time.Duration(500+time.Now().UnixNano()%1500) * time.Millisecond
	select {
	case <-time.After(processingTime):

		if rand.Intn(10)%3 == 0 {
			return fmt.Errorf("notification processing timed out")
		}
		xlog.Info(ctx, "Notification sent successfully to user",
			zap.Duration("processingTime", processingTime),
		)
		return nil
	case <-ctx.Done():
		return fmt.Errorf("notification task canceled: %w", ctx.Err())
	}
}
