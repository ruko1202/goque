// Package xtracer provides OpenTelemetry tracing utilities for Goque.
// It manages the global tracer provider and tracer instances used throughout the library.
package xtracer

import "runtime/debug"

const (
	// PkgName is the package identifier used for OpenTelemetry instrumentation.
	PkgName        = "github.com/ruko1202/goque"
	defaultVersion = "v0.0.0"
)

// GetVersion returns the version of the Goque package.
// It reads the version from build info or returns the default version if not available.
func GetVersion() string {
	info, ok := debug.ReadBuildInfo()
	if ok {
		for _, dep := range info.Deps {
			if dep.Path == PkgName {
				return dep.Version
			}
		}
	}
	return defaultVersion
}
