package processors

import (
	"context"
	"fmt"

	"example/internal/models"

	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"github.com/ruko1202/goque"
)

// OrderConfirmationProcessor processes tasks enqueued by the
// transactional-outbox example endpoint (POST /api/orders). The task
// payload is OrderConfirmationPayload — small reference to the order
// so the processor can fetch any extra data it needs from the DB.
//
// In a real service this would render an email/SMS/push and call out
// to the delivery provider. Here we just log the simulated send so
// the example stays self-contained.
type OrderConfirmationProcessor struct{}

// NewOrderConfirmationProcessor constructs the processor.
func NewOrderConfirmationProcessor() *OrderConfirmationProcessor {
	return &OrderConfirmationProcessor{}
}

// ProcessTask implements goque.TaskProcessor. The processor is wired
// through NewTypedTaskProcessor in main.go so the JSON-decoded
// OrderConfirmationPayload arrives already in task.Payload.
func (p *OrderConfirmationProcessor) ProcessTask(
	ctx context.Context, task *goque.TypedTask[models.OrderConfirmationPayload],
) error {
	ctx, span := xlog.WithOperationSpan(ctx, "OrderConfirmationProcessor")
	defer span.End()

	if task.Payload.OrderID == "" || task.Payload.Customer == "" {
		// Cancel: payload malformed beyond rescue — no point retrying.
		return fmt.Errorf("invalid order confirmation payload: %+v", task.Payload)
	}

	xlog.Info(ctx, "sending order confirmation",
		xfield.String("order_id", task.Payload.OrderID),
		xfield.String("customer", task.Payload.Customer),
	)

	// Pretend we called out to an email/SMS provider here.
	// Side note: this processor is a perfect place to use
	// goque.WithTx itself — e.g. if the confirmation send needs to
	// update an `outbox_sent` table atomically with marking the task
	// done. Out of scope for the example.
	return nil
}
