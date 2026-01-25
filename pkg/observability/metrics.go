package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestSize     *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Storage metrics
	StorageOperationsTotal    *prometheus.CounterVec
	StorageOperationDuration  *prometheus.HistogramVec
	StorageErrorsTotal        *prometheus.CounterVec

	// Compilation metrics
	CompilationTotal          *prometheus.CounterVec
	CompilationDuration       *prometheus.HistogramVec
	CompilationErrorsTotal    *prometheus.CounterVec

	// Cache metrics
	CacheHitsTotal            *prometheus.CounterVec
	CacheMissesTotal          *prometheus.CounterVec
	CacheEvictionsTotal       *prometheus.CounterVec
	CacheSizeBytes            *prometheus.GaugeVec

	// Database metrics
	DBConnectionsActive       prometheus.Gauge
	DBConnectionsIdle         prometheus.Gauge
	DBConnectionsWaitCount    prometheus.Gauge
	DBConnectionsWaitDuration prometheus.Gauge

	// Redis metrics
	RedisConnectionsActive    prometheus.Gauge
	RedisCommandsTotal        *prometheus.CounterVec
	RedisCommandDuration      *prometheus.HistogramVec

	// Business metrics
	ModulesTotal              prometheus.Gauge
	VersionsTotal             prometheus.Gauge
	ActiveUsersTotal          prometheus.Gauge
	APITokensActive           prometheus.Gauge
}

// NewMetrics creates and registers all Prometheus metrics
func NewMetrics(registry *prometheus.Registry) *Metrics {
	m := &Metrics{
		// HTTP metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPRequestSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_http_request_size_bytes",
				Help:    "HTTP request size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: prometheus.ExponentialBuckets(100, 10, 8),
			},
			[]string{"method", "path"},
		),

		// Storage metrics
		StorageOperationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_storage_operations_total",
				Help: "Total number of storage operations",
			},
			[]string{"operation", "backend", "status"},
		),
		StorageOperationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_storage_operation_duration_seconds",
				Help:    "Storage operation duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "backend"},
		),
		StorageErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_storage_errors_total",
				Help: "Total number of storage errors",
			},
			[]string{"operation", "backend", "error_type"},
		),

		// Compilation metrics
		CompilationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_compilation_total",
				Help: "Total number of compilations",
			},
			[]string{"language", "status"},
		),
		CompilationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_compilation_duration_seconds",
				Help:    "Compilation duration in seconds",
				Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
			},
			[]string{"language"},
		),
		CompilationErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_compilation_errors_total",
				Help: "Total number of compilation errors",
			},
			[]string{"language", "error_type"},
		),

		// Cache metrics
		CacheHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_cache_hits_total",
				Help: "Total number of cache hits",
			},
			[]string{"cache_type", "key_type"},
		),
		CacheMissesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_cache_misses_total",
				Help: "Total number of cache misses",
			},
			[]string{"cache_type", "key_type"},
		),
		CacheEvictionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_cache_evictions_total",
				Help: "Total number of cache evictions",
			},
			[]string{"cache_type", "reason"},
		),
		CacheSizeBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "spoke_cache_size_bytes",
				Help: "Current cache size in bytes",
			},
			[]string{"cache_type"},
		),

		// Database metrics
		DBConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_db_connections_active",
				Help: "Number of active database connections",
			},
		),
		DBConnectionsIdle: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_db_connections_idle",
				Help: "Number of idle database connections",
			},
		),
		DBConnectionsWaitCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_db_connections_wait_count",
				Help: "Total number of connections waited for",
			},
		),
		DBConnectionsWaitDuration: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_db_connections_wait_duration_seconds",
				Help: "Total time spent waiting for connections",
			},
		),

		// Redis metrics
		RedisConnectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_redis_connections_active",
				Help: "Number of active Redis connections",
			},
		),
		RedisCommandsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "spoke_redis_commands_total",
				Help: "Total number of Redis commands",
			},
			[]string{"command", "status"},
		),
		RedisCommandDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "spoke_redis_command_duration_seconds",
				Help:    "Redis command duration in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"command"},
		),

		// Business metrics
		ModulesTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_modules_total",
				Help: "Total number of modules",
			},
		),
		VersionsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_versions_total",
				Help: "Total number of versions",
			},
		),
		ActiveUsersTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_active_users_total",
				Help: "Total number of active users",
			},
		),
		APITokensActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "spoke_api_tokens_active",
				Help: "Number of active API tokens",
			},
		),
	}

	// Register all metrics
	registry.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestSize,
		m.HTTPResponseSize,
		m.StorageOperationsTotal,
		m.StorageOperationDuration,
		m.StorageErrorsTotal,
		m.CompilationTotal,
		m.CompilationDuration,
		m.CompilationErrorsTotal,
		m.CacheHitsTotal,
		m.CacheMissesTotal,
		m.CacheEvictionsTotal,
		m.CacheSizeBytes,
		m.DBConnectionsActive,
		m.DBConnectionsIdle,
		m.DBConnectionsWaitCount,
		m.DBConnectionsWaitDuration,
		m.RedisConnectionsActive,
		m.RedisCommandsTotal,
		m.RedisCommandDuration,
		m.ModulesTotal,
		m.VersionsTotal,
		m.ActiveUsersTotal,
		m.APITokensActive,
	)

	return m
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// HTTPMetricsMiddleware instruments HTTP requests with Prometheus metrics
func HTTPMetricsMiddleware(metrics *Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status and size
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Record request size
			if r.ContentLength > 0 {
				metrics.HTTPRequestSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(r.ContentLength))
			}

			// Serve the request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.statusCode)

			metrics.HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
			metrics.HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
			metrics.HTTPResponseSize.WithLabelValues(r.Method, r.URL.Path).Observe(float64(rw.bytesWritten))
		})
	}
}

// RegisterMetricsEndpoint registers the /metrics endpoint
func RegisterMetricsEndpoint(mux *http.ServeMux, registry *prometheus.Registry) {
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}
