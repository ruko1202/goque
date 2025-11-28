package app

import (
	"github.com/labstack/echo/v4"
	"github.com/ruko1202/xlog"
	"go.uber.org/zap"
)

// XlogMiddleware adds the xlog logger to the request context.
func XlogMiddleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get request ID if available
			requestID := c.Response().Header().Get(echo.HeaderXRequestID)

			// Create logger with request ID
			loggerWithFields := logger
			if requestID != "" {
				loggerWithFields = logger.With(zap.String("request_id", requestID))
			}

			// Add logger to request context
			ctx := xlog.ContextWithLogger(c.Request().Context(), loggerWithFields)

			// Replace request with the new context
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}
