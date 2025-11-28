package app

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"example/internal/models"

	"github.com/ruko1202/goque"
)

// CreateTaskHandler handles POST /api/tasks requests.
func (a *Application) CreateTaskHandler(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.CreateTaskRequest
	if err := c.Bind(&req); err != nil {
		xlog.Warn(ctx, "failed to bind request body", zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	xlog.Info(ctx, "creating task", zap.String("type", req.Type), zap.String("external_id", req.ExternalID))

	// Validate task type
	if err := models.ValidateTaskType(req.Type); err != nil {
		xlog.Warn(ctx, "invalid task type", zap.String("type", req.Type), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Generate external ID if not provided
	if req.ExternalID == "" {
		req.ExternalID = uuid.New().String()
	}

	// Create task entity
	task := goque.NewTaskWithExternalID(req.Type, string(req.Payload), req.ExternalID)

	// Add task to queue using TaskQueueManager
	if err := a.queueManager.AddTaskToQueue(ctx, task); err != nil {
		xlog.Error(ctx, "failed to create task",
			zap.String("type", req.Type),
			zap.String("external_id", req.ExternalID),
			zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create task"})
	}

	xlog.Info(ctx, "task created successfully",
		zap.String("task_id", task.ID.String()),
		zap.String("type", req.Type),
		zap.String("external_id", req.ExternalID))

	return c.JSON(http.StatusCreated, toTaskResponse(task))
}
