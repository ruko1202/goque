package app

import (
	"time"

	"example/internal/models"

	"github.com/ruko1202/goque"
)

// toTaskResponse converts a goque.Task to a TaskResponse.
func toTaskResponse(task *goque.Task) models.TaskResponse {
	var updatedAt string
	if task.UpdatedAt != nil {
		updatedAt = task.UpdatedAt.Format(time.RFC3339)
	} else {
		updatedAt = task.CreatedAt.Format(time.RFC3339)
	}

	resp := models.TaskResponse{
		ID:         task.ID.String(),
		Type:       string(task.Type),
		ExternalID: task.ExternalID,
		Status:     string(task.Status),
		Attempts:   int(task.Attempts),
		Payload:    task.Payload,
		CreatedAt:  task.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  updatedAt,
	}

	if task.Errors != nil && *task.Errors != "" {
		resp.Error = *task.Errors
	}

	if !task.NextAttemptAt.IsZero() {
		nextAttempt := task.NextAttemptAt.Format(time.RFC3339)
		resp.NextAttemptAt = &nextAttempt
	}

	return resp
}
