package app

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SetupRoutes configures all HTTP routes for the application.
func SetupRoutes(e *echo.Echo, app *Application) {
	// Health check
	e.GET("/health", app.HealthCheckHandler)

	// API routes
	api := e.Group("/api")
	{
		// Tasks
		api.POST("/tasks", app.CreateTaskHandler)
		api.GET("/tasks", app.ListTasksHandler)
		api.GET("/tasks/:id", app.GetTaskHandler)
	}

	// Serve static files
	e.Static("/static", "web/static")

	// Serve dashboard
	e.GET("/", func(c echo.Context) error {
		return c.File("web/templates/dashboard.html")
	})

	// 404 handler
	e.RouteNotFound("/*", func(c echo.Context) error {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "route not found"})
	})
}
