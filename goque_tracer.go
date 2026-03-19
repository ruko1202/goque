package goque

import (
	"go.opentelemetry.io/otel/trace"

	"github.com/ruko1202/goque/internal/utils/xtracer"
)

// SetTracerProvider configures the TracerProvider used by Goque for distributed tracing.
//
// This function must be called BEFORE creating any Goque instances or TaskQueueManager instances,
// as they capture the tracer at creation time.
//
// By default, Goque uses a noop tracer (zero overhead). Call this function to enable
// OpenTelemetry distributed tracing in production environments.
//
// Example usage:
//
//	import (
//	    "github.com/ruko1202/goque"
//	    sdktrace "go.opentelemetry.io/otel/sdk/trace"
//	)
//
//	// Initialize TracerProvider
//	tracerProvider := sdktrace.NewTracerProvider(
//	    sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.01)), // 1% sampling
//	    sdktrace.WithBatcher(exporter),
//	)
//	defer tracerProvider.Shutdown(ctx)
//
//	// Configure Goque (BEFORE creating instances)
//	goque.SetTracerProvider(tracerProvider)
//
//	// Now create Goque instances
//	goq := goque.NewGoque(taskStorage)
func SetTracerProvider(tp trace.TracerProvider) {
	xtracer.SetTracerProvider(tp)
}
