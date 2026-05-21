package processors

import (
	"context"
	"math/rand"
	"time"

	"example/internal/models"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque"
)

// EmailProcessor processes email sending tasks.
type EmailProcessor struct{}

// NewEmailProcessor creates a new email processor.
func NewEmailProcessor() *EmailProcessor {
	return &EmailProcessor{}
}

// ProcessTask implements the TaskProcessor interface for email tasks.
func (p *EmailProcessor) ProcessTask(ctx context.Context, task *goque.TypedTask[models.EmailPayload]) error {
	ctx, span := xlog.WithOperationSpan(ctx, "EmailProcessor")
	defer span.End()

	ctx = xlog.WithFields(ctx,
		xfield.Any("payload", task.Payload),
		xfield.String("subject", task.Payload.Subject),
		xfield.String("to", task.Payload.To),
	)

	xlog.Info(ctx, "Processing email task")
	// Simulate email sending with random processing time (100ms-3 seconds)
	processingTime := time.Duration(100+rand.Intn(2_900)) * time.Millisecond //nolint:gosec
	return mockProcess(ctx, "email notification", processingTime, 25)
}
