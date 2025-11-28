package models

import (
	"encoding/json"
	"fmt"
)

// TaskType represents the type of task to be processed.
type TaskType = string

const (
	TaskTypeEmail        TaskType = "email"
	TaskTypeNotification TaskType = "notification"
	TaskTypeReport       TaskType = "report"
	TaskTypeWebhook      TaskType = "webhook"
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
	switch TaskType(taskType) {
	case TaskTypeEmail, TaskTypeNotification, TaskTypeReport, TaskTypeWebhook:
		return nil
	default:
		return fmt.Errorf("invalid task type: %s", taskType)
	}
}
