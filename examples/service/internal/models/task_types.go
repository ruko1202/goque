// Package models contains the data structures used in the application.
package models

import (
	"encoding/json"
	"fmt"
)

// TaskType represents the type of task to be processed.
type TaskType = string

const (
	// TaskTypeEmail .
	TaskTypeEmail TaskType = "email"
	// TaskTypeNotification .
	TaskTypeNotification TaskType = "notification"
	// TaskTypeReport .
	TaskTypeReport TaskType = "report"
	// TaskTypeWebhook .
	TaskTypeWebhook TaskType = "webhook"
	// TaskTypeTaskGenerator .
	TaskTypeTaskGenerator TaskType = "task_generator"
	// TaskTypeOrderConfirmation is enqueued by the outbox example
	// endpoint (POST /api/orders). The processor sends the customer
	// a "your order is confirmed" notification — same pipeline as
	// any other notification, but enqueue is atomic with the
	// orders-row INSERT.
	TaskTypeOrderConfirmation TaskType = "order_confirmation"
)

// EmailPayload represents the payload for email tasks.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// NotificationPayload represents the payload for notification tasks.
type NotificationPayload struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

// ReportPayload represents the payload for report generation tasks.
type ReportPayload struct {
	ReportType string `json:"report_type"`
	DateFrom   string `json:"date_from"`
	DateTo     string `json:"date_to"`
	Format     string `json:"format"`
}

// WebhookPayload represents the payload for webhook tasks.
type WebhookPayload struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

// CreateTaskRequest represents a request to create a new task.
type CreateTaskRequest struct {
	Type       string          `json:"type"`
	ExternalID string          `json:"external_id,omitempty"`
	Payload    json.RawMessage `json:"payload"`
}

// OrderConfirmationPayload is the queue payload produced by the
// outbox example: a small reference to the order so the processor
// can render the confirmation message.
type OrderConfirmationPayload struct {
	OrderID  string `json:"order_id"`
	Customer string `json:"customer"`
}

// CreateOrderRequest is the body of POST /api/orders. The handler
// writes a row into the orders table AND enqueues a task to send
// the confirmation — both inside the same transaction.
type CreateOrderRequest struct {
	Customer    string `json:"customer"`
	AmountCents int    `json:"amount_cents"`
}

// OrderResponse is what POST /api/orders returns once the tx commits.
type OrderResponse struct {
	ID          string `json:"id"`
	Customer    string `json:"customer"`
	AmountCents int    `json:"amount_cents"`
	Status      string `json:"status"`
	// EnqueuedTaskID is the goque task that was created atomically
	// with this order. Useful for tracing: hit GET /api/tasks/{id}
	// to see its lifecycle.
	EnqueuedTaskID string `json:"enqueued_task_id"`
}

// TaskResponse represents a task in API responses.
type TaskResponse struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"`
	ExternalID    string  `json:"external_id,omitempty"`
	Status        string  `json:"status"`
	Attempts      int     `json:"attempts"`
	Payload       string  `json:"payload"`
	Error         string  `json:"error,omitempty"`
	NextAttemptAt *string `json:"next_attempt_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

// TaskListResponse represents a paginated list of tasks.
type TaskListResponse struct {
	Tasks      []TaskResponse `json:"tasks"`
	Total      int            `json:"total"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
	TotalPages int            `json:"total_pages"`
}

// ValidateTaskType checks if the given task type is valid.
func ValidateTaskType(taskType string) error {
	switch taskType {
	case TaskTypeEmail, TaskTypeNotification, TaskTypeReport, TaskTypeWebhook, TaskTypeTaskGenerator:
		return nil
	default:
		return fmt.Errorf("invalid task type: %s", taskType)
	}
}
