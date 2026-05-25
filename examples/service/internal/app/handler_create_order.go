package app

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"

	"example/internal/models"

	"github.com/ruko1202/goque"
)

// CreateOrderHandler is a runnable demonstration of the transactional
// outbox pattern enabled by goque.WithTx.
//
// The handler does TWO writes that must be atomic:
//
//  1. INSERT a new row into the domain `orders` table.
//  2. Enqueue a goque task to send the customer a confirmation
//     ("order_confirmation" task type).
//
// Without the outbox: a crash between the two leaves either a paid
// order without a confirmation (bad UX) or a phantom confirmation for
// an order that was rolled back (worse).
//
// With the outbox: both rows are written through the SAME *sqlx.Tx —
// the goque task picks it up via `goque.WithTx(ctx, tx)`. tx.Commit
// makes them durable together; tx.Rollback discards both.
//
// POST /api/orders
//
//	{"customer": "alice@example.com", "amount_cents": 1999}
//
// Response: 201 with the order id and the enqueued task id (useful
// for `GET /api/tasks/{task_id}` to watch the confirmation get
// processed).
func (a *Application) CreateOrderHandler(c echo.Context) error {
	ctx, span := xlog.WithOperationSpan(c.Request().Context(), "app.CreateOrderHandler")
	defer span.End()

	var req models.CreateOrderRequest
	if err := c.Bind(&req); err != nil {
		xlog.Warn(ctx, "failed to bind request body", xfield.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}
	if req.Customer == "" || req.AmountCents <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "customer must be non-empty and amount_cents must be > 0",
		})
	}

	orderID := uuid.New()

	// 1) Open the tx that will wrap both the domain write and the
	//    queue enqueue. Defer-rollback is a no-op after a successful
	//    Commit (sqlx returns sql.ErrTxDone, which we ignore).
	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		xlog.Error(ctx, "begin tx", xfield.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
	defer func() { _ = tx.Rollback() }()

	// 2) Domain write — the row goes into the same tx.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO orders (id, customer, amount_cents, status) VALUES ($1, $2, $3, 'created')`,
		orderID, req.Customer, req.AmountCents,
	); err != nil {
		xlog.Error(ctx, "insert order", xfield.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create order"})
	}

	// 3) Enqueue the confirmation task — goque.WithTx wires our tx
	//    into ctx so AddTaskToQueue performs its INSERT through the
	//    same tx instead of opening its own.
	payload, _ := json.Marshal(models.OrderConfirmationPayload{
		OrderID:  orderID.String(),
		Customer: req.Customer,
	})
	task := goque.NewTaskWithExternalID(
		models.TaskTypeOrderConfirmation,
		string(payload),
		orderID.String(), // external_id == order id makes the enqueue idempotent on retry
	)
	if err := a.queueManager.AddTaskToQueue(goque.WithTx(ctx, tx), task); err != nil {
		xlog.Error(ctx, "enqueue confirmation task", xfield.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to enqueue confirmation"})
	}

	// 4) Commit — order row and queued task become durable together.
	//    If the process crashes between the two ExecContext calls
	//    above the tx is automatically rolled back by the DB and
	//    neither lands.
	if err := tx.Commit(); err != nil {
		xlog.Error(ctx, "commit tx", xfield.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to commit"})
	}

	xlog.Info(ctx, "order created and confirmation enqueued atomically",
		xfield.String("order_id", orderID.String()),
		xfield.String("task_id", task.ID.String()),
		xfield.String("customer", req.Customer),
	)

	return c.JSON(http.StatusCreated, models.OrderResponse{
		ID:             orderID.String(),
		Customer:       req.Customer,
		AmountCents:    req.AmountCents,
		Status:         "created",
		EnqueuedTaskID: task.ID.String(),
	})
}
