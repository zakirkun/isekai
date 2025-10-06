package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/zakirkun/isekai/internal/metrics"
)

// MetricsMiddleware tracks HTTP metrics
func MetricsMiddleware(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			m.ActiveConnections.Inc()
			defer m.ActiveConnections.Dec()

			// Wrap response writer to capture status code
			wrapped := &metricsResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Seconds()
			status := strconv.Itoa(wrapped.statusCode)

			// Record metrics
			m.RequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
			m.RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		})
	}
}

type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *metricsResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
