package app

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

// GetTaskHandler handles GET /api/tasks/:id requests.
func (a *Application) GetTaskHandler(c echo.Context) error {
	ctx := c.Request().Context()

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		xlog.Warn(ctx, "invalid task ID format", zap.String("task_id", taskIDStr), zap.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid task ID"})
	}

	xlog.Debug(ctx, "fetching task", zap.String("task_id", taskID.String()))

	task, err := a.queueManager.GetTask(ctx, taskID)
	if err != nil {
		xlog.Warn(ctx, "task not found", zap.String("task_id", taskID.String()), zap.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	xlog.Debug(ctx, "task retrieved successfully",
		zap.String("task_id", taskID.String()),
		zap.String("type", string(task.Type)),
		zap.String("status", string(task.Status)))

	return c.JSON(http.StatusOK, toTaskResponse(task))
}
