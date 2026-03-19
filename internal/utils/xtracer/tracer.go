package xtracer

import (
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	_mu                   sync.RWMutex
	_globalTracerProvider trace.TracerProvider = noop.NewTracerProvider()
	tracer                                     = initTracer(_globalTracerProvider)
)

// SetTracerProvider sets the global OpenTelemetry tracer provider.
// This function is thread-safe and updates the global tracer instance.
func SetTracerProvider(tp trace.TracerProvider) {
	_mu.Lock()
	defer _mu.Unlock()

	_globalTracerProvider = tp
	tracer = initTracer(tp)
}

// GetTracer returns the global tracer instance configured with the current tracer provider.
// This function is thread-safe and can be called concurrently.
func GetTracer() trace.Tracer {
	_mu.RLock()
	defer _mu.RUnlock()

	return tracer
}

func initTracer(tp trace.TracerProvider) trace.Tracer {
	return tp.Tracer(
		PkgName,
		trace.WithInstrumentationVersion(GetVersion()),
	)
}
