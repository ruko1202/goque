package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	namespace   = ""
	constLabels = prometheus.Labels{}
)

// SetServiceName sets the service name label for all metrics.
func SetServiceName(name string) {
	constLabels["service"] = name
}
