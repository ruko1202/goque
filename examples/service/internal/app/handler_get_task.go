package app

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"github.com/ruko1202/xlog/xfield"
)

// GetTaskHandler handles GET /api/tasks/:id requests.
func (a *Application) GetTaskHandler(c echo.Context) error {
	ctx, span := xlog.WithOperationSpan(c.Request().Context(), "app.GetTaskHandler")
	defer span.End()

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		xlog.Warn(ctx, "invalid task ID format", xfield.String("task_id", taskIDStr), xfield.Error(err))
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid task ID"})
	}

	xlog.Debug(ctx, "fetching task", xfield.String("task_id", taskID.String()))

	task, err := a.queueManager.GetTask(ctx, taskID)
	if err != nil {
		xlog.Warn(ctx, "task not found", xfield.String("task_id", taskID.String()), xfield.Error(err))
		return c.JSON(http.StatusNotFound, map[string]string{"error": "task not found"})
	}

	xlog.Debug(ctx, "task retrieved successfully",
		xfield.String("task_id", taskID.String()),
		xfield.String("type", task.Type),
		xfield.String("status", task.Status))

	return c.JSON(http.StatusOK, toTaskResponse(task))
}
