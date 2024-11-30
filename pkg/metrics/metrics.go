// pkg/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Counter for total requests
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests processed, labeled by status code and HTTP method.",
		},
		[]string{"code", "method"},
	)

	// Histogram for request duration
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"code", "method"},
	)
)

// InitMetrics registers the metrics with Prometheus
func InitMetrics() {
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
}
