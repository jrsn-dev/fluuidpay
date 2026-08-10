package platform

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HttpRequestsTotal counts all incoming HTTP requests.
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_http_requests_total",
			Help: "Total number of HTTP requests processed, partitioned by status code and method.",
		},
		[]string{"method", "path", "status"},
	)

	// HttpRequestDuration measures the duration of HTTP requests.
	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_http_request_duration_seconds",
			Help:    "Histogram of response latencies.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	// PaymentOperationsTotal counts payment business operations (create, cancel, etc).
	PaymentOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_operations_total",
			Help: "Total number of payment operations partitioned by operation and status.",
		},
		[]string{"operation", "status"},
	)
)

// MetricsMiddleware records HTTP metrics for Prometheus.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())

		// We use chi patterns if available, otherwise just the raw path.
		// Note: In production, high cardinality on raw paths (e.g., /payments/{id})
		// is dangerous. You should ideally extract the route pattern.
		path := r.URL.Path

		HttpRequestsTotal.WithLabelValues(r.Method, path, status).Inc()
		HttpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

// MetricsHandler returns the HTTP handler for Prometheus metrics scraping.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
