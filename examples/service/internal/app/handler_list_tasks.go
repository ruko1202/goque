package app

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"

	"example/internal/models"

	"github.com/ruko1202/goque"
)

// ListTasksHandler handles GET /api/tasks requests with pagination and filters.
// Note: This is a simplified version for the example. In production, you would want to
// implement proper filtering at the database level for better performance.
func (a *Application) ListTasksHandler(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	statusFilter := c.QueryParam("status")
	typeFilter := c.QueryParam("type")

	xlog.Debug(ctx, "listing tasks",
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
		zap.String("status_filter", statusFilter),
		zap.String("type_filter", typeFilter))

	// Build filter for TaskQueueManager
	filter := &goque.TaskFilter{}

	if statusFilter != "" {
		status := goque.TaskStatus(statusFilter)
		filter.Status = &status
	}

	if typeFilter != "" {
		taskType := goque.TaskType(typeFilter)
		filter.TaskType = &taskType
	}

	// Get tasks using TaskQueueManager with high limit (10000)
	// Note: This is a simplified version. In production, you would want to
	// implement proper pagination at the database level for better performance.
	filteredTasks, err := a.queueManager.GetTasks(ctx, filter, 10000)
	if err != nil {
		xlog.Error(ctx, "failed to get tasks from queue", zap.Error(err))
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list tasks"})
	}

	// Calculate pagination
	total := len(filteredTasks)
	offset := (page - 1) * pageSize
	end := offset + pageSize

	if offset > total {
		offset = total
	}
	if end > total {
		end = total
	}

	paginatedTasks := filteredTasks[offset:end]

	// Convert to response format
	taskResponses := make([]models.TaskResponse, len(paginatedTasks))
	for i, task := range paginatedTasks {
		taskResponses[i] = toTaskResponse(task)
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	xlog.Debug(ctx, "tasks listed successfully",
		zap.Int("total", total),
		zap.Int("page", page),
		zap.Int("page_size", pageSize),
		zap.Int("returned", len(paginatedTasks)))

	response := models.TaskListResponse{
		Tasks:      taskResponses,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	return c.JSON(http.StatusOK, response)
}
