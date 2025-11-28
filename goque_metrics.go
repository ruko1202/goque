package goque

import (
	"github.com/ruko1202/goque/internal/metrics"
)

// SetMetricsServiceName sets the service name label for Prometheus metrics.
// This should be called once during application initialization, before starting the queue manager.
//
// Example:
//
//	goque.SetMetricsServiceName("my-service")
func SetMetricsServiceName(name string) {
	metrics.SetServiceName(name)
}
