package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	RequestsTotal       *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	ActiveConnections   prometheus.Gauge
	CacheHits           prometheus.Counter
	CacheMisses         prometheus.Counter
	ProxyErrors         *prometheus.CounterVec
	DatabaseQueries     *prometheus.HistogramVec
	CircuitBreakerState *prometheus.GaugeVec
}

// New creates a new metrics instance
func New() *Metrics {
	return &Metrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "isekai_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "isekai_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "isekai_active_connections",
				Help: "Number of active connections",
			},
		),
		CacheHits: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "isekai_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		CacheMisses: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "isekai_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		ProxyErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "isekai_proxy_errors_total",
				Help: "Total number of proxy errors",
			},
			[]string{"target", "error_type"},
		),
		DatabaseQueries: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "isekai_database_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"query_type"},
		),
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "isekai_circuit_breaker_state",
				Help: "Circuit breaker state (0=closed, 1=half-open, 2=open)",
			},
			[]string{"target"},
		),
	}
}
