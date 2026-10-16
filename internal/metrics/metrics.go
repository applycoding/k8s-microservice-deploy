package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTPRequests counts completed requests by method, route template, and status.
var HTTPRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests processed, partitioned by method, route, and status.",
	},
	[]string{"method", "route", "status"},
)

// HTTPDuration records handler latency in seconds.
var HTTPDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route"},
)

// TripsCreated counts trips that were stored successfully.
var TripsCreated = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "trips_created_total",
		Help: "Trips successfully created.",
	},
)

// Observe records one HTTP request.
func Observe(method, route string, status int, elapsed time.Duration) {
	HTTPRequests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	HTTPDuration.WithLabelValues(method, route).Observe(elapsed.Seconds())
}
