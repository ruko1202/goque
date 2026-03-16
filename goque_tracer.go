package goque

import (
	"runtime/debug"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	pkgName = "github.com/ruko1202/goque"
)

var tracer = otel.Tracer(
	pkgName,
	trace.WithInstrumentationVersion(getVersion()),
)

func getVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			if dep.Path == pkgName {
				return dep.Version
			}
		}
	}
	return "unknown"
}
