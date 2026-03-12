package app

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
)

// HealthCheckHandler handles GET /health requests.
func (a *Application) HealthCheckHandler(c echo.Context) error {
	ctx, span := xlog.WithOperationSpan(c.Request().Context(), "app.HealthCheckHandler")
	defer span.End()

	xlog.Debug(ctx, "health check passed")
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
	})
}
