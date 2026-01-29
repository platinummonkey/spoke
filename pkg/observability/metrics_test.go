package observability

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewMetrics(t *testing.T) {
	t.Run("creates and registers all metrics", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		if metrics == nil {
			t.Fatal("NewMetrics returned nil")
		}

		// Verify HTTP metrics are initialized
		if metrics.HTTPRequestsTotal == nil {
			t.Error("HTTPRequestsTotal is nil")
		}
		if metrics.HTTPRequestDuration == nil {
			t.Error("HTTPRequestDuration is nil")
		}
		if metrics.HTTPRequestSize == nil {
			t.Error("HTTPRequestSize is nil")
		}
		if metrics.HTTPResponseSize == nil {
			t.Error("HTTPResponseSize is nil")
		}

		// Verify Storage metrics are initialized
		if metrics.StorageOperationsTotal == nil {
			t.Error("StorageOperationsTotal is nil")
		}
		if metrics.StorageOperationDuration == nil {
			t.Error("StorageOperationDuration is nil")
		}
		if metrics.StorageErrorsTotal == nil {
			t.Error("StorageErrorsTotal is nil")
		}

		// Verify Compilation metrics are initialized
		if metrics.CompilationTotal == nil {
			t.Error("CompilationTotal is nil")
		}
		if metrics.CompilationDuration == nil {
			t.Error("CompilationDuration is nil")
		}
		if metrics.CompilationErrorsTotal == nil {
			t.Error("CompilationErrorsTotal is nil")
		}

		// Verify Cache metrics are initialized
		if metrics.CacheHitsTotal == nil {
			t.Error("CacheHitsTotal is nil")
		}
		if metrics.CacheMissesTotal == nil {
			t.Error("CacheMissesTotal is nil")
		}
		if metrics.CacheEvictionsTotal == nil {
			t.Error("CacheEvictionsTotal is nil")
		}
		if metrics.CacheSizeBytes == nil {
			t.Error("CacheSizeBytes is nil")
		}

		// Verify Database metrics are initialized
		if metrics.DBConnectionsActive == nil {
			t.Error("DBConnectionsActive is nil")
		}
		if metrics.DBConnectionsIdle == nil {
			t.Error("DBConnectionsIdle is nil")
		}
		if metrics.DBConnectionsWaitCount == nil {
			t.Error("DBConnectionsWaitCount is nil")
		}
		if metrics.DBConnectionsWaitDuration == nil {
			t.Error("DBConnectionsWaitDuration is nil")
		}

		// Verify Redis metrics are initialized
		if metrics.RedisConnectionsActive == nil {
			t.Error("RedisConnectionsActive is nil")
		}
		if metrics.RedisCommandsTotal == nil {
			t.Error("RedisCommandsTotal is nil")
		}
		if metrics.RedisCommandDuration == nil {
			t.Error("RedisCommandDuration is nil")
		}

		// Verify Business metrics are initialized
		if metrics.ModulesTotal == nil {
			t.Error("ModulesTotal is nil")
		}
		if metrics.VersionsTotal == nil {
			t.Error("VersionsTotal is nil")
		}
		if metrics.ActiveUsersTotal == nil {
			t.Error("ActiveUsersTotal is nil")
		}
		if metrics.APITokensActive == nil {
			t.Error("APITokensActive is nil")
		}
	})

	t.Run("metrics are registered with registry", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Initialize some metrics to make them appear in Gather()
		metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Add(0)
		metrics.StorageOperationsTotal.WithLabelValues("read", "s3", "success").Add(0)
		metrics.CompilationTotal.WithLabelValues("go", "success").Add(0)
		metrics.CacheHitsTotal.WithLabelValues("memory", "module").Add(0)
		metrics.DBConnectionsActive.Set(0)
		metrics.RedisConnectionsActive.Set(0)
		metrics.ModulesTotal.Set(0)

		// Gather metrics from registry to verify registration
		families, err := registry.Gather()
		if err != nil {
			t.Fatalf("Failed to gather metrics: %v", err)
		}

		if len(families) == 0 {
			t.Error("No metrics registered in registry")
		}

		// Verify some key metrics are present
		metricNames := make(map[string]bool)
		for _, family := range families {
			metricNames[family.GetName()] = true
		}

		expectedMetrics := []string{
			"spoke_http_requests_total",
			"spoke_storage_operations_total",
			"spoke_compilation_total",
			"spoke_cache_hits_total",
			"spoke_db_connections_active",
			"spoke_redis_connections_active",
			"spoke_modules_total",
		}

		for _, name := range expectedMetrics {
			if !metricNames[name] {
				t.Errorf("Expected metric %s not found in registry", name)
			}
		}
	})

	t.Run("panics on duplicate registration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		NewMetrics(registry)

		// Attempting to register again should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic on duplicate registration, but didn't panic")
			}
		}()

		NewMetrics(registry)
	})
}

func TestMetrics_HTTPMetrics(t *testing.T) {
	t.Run("increment HTTP request counter", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200").Inc()

		count := testutil.CollectAndCount(metrics.HTTPRequestsTotal)
		if count != 1 {
			t.Errorf("Expected 1 metric, got %d", count)
		}

		// Verify the value
		expected := `
# HELP spoke_http_requests_total Total number of HTTP requests
# TYPE spoke_http_requests_total counter
spoke_http_requests_total{method="GET",path="/api/test",status="200"} 1
`
		if err := testutil.CollectAndCompare(metrics.HTTPRequestsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("observe HTTP request duration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.HTTPRequestDuration.WithLabelValues("POST", "/api/create").Observe(0.5)
		metrics.HTTPRequestDuration.WithLabelValues("POST", "/api/create").Observe(1.5)

		count := testutil.CollectAndCount(metrics.HTTPRequestDuration)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})

	t.Run("observe HTTP request size", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.HTTPRequestSize.WithLabelValues("POST", "/api/upload").Observe(1024)
		metrics.HTTPRequestSize.WithLabelValues("POST", "/api/upload").Observe(2048)

		count := testutil.CollectAndCount(metrics.HTTPRequestSize)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})

	t.Run("observe HTTP response size", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.HTTPResponseSize.WithLabelValues("GET", "/api/data").Observe(4096)

		count := testutil.CollectAndCount(metrics.HTTPResponseSize)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})
}

func TestMetrics_StorageMetrics(t *testing.T) {
	t.Run("record storage operations", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.StorageOperationsTotal.WithLabelValues("read", "s3", "success").Inc()
		metrics.StorageOperationsTotal.WithLabelValues("write", "s3", "success").Inc()

		expected := `
# HELP spoke_storage_operations_total Total number of storage operations
# TYPE spoke_storage_operations_total counter
spoke_storage_operations_total{backend="s3",operation="read",status="success"} 1
spoke_storage_operations_total{backend="s3",operation="write",status="success"} 1
`
		if err := testutil.CollectAndCompare(metrics.StorageOperationsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("observe storage operation duration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.StorageOperationDuration.WithLabelValues("read", "local").Observe(0.01)

		count := testutil.CollectAndCount(metrics.StorageOperationDuration)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})

	t.Run("record storage errors", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.StorageErrorsTotal.WithLabelValues("write", "s3", "timeout").Inc()

		expected := `
# HELP spoke_storage_errors_total Total number of storage errors
# TYPE spoke_storage_errors_total counter
spoke_storage_errors_total{backend="s3",error_type="timeout",operation="write"} 1
`
		if err := testutil.CollectAndCompare(metrics.StorageErrorsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})
}

func TestMetrics_CompilationMetrics(t *testing.T) {
	t.Run("record compilation count", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CompilationTotal.WithLabelValues("go", "success").Inc()
		metrics.CompilationTotal.WithLabelValues("rust", "failure").Inc()

		expected := `
# HELP spoke_compilation_total Total number of compilations
# TYPE spoke_compilation_total counter
spoke_compilation_total{language="go",status="success"} 1
spoke_compilation_total{language="rust",status="failure"} 1
`
		if err := testutil.CollectAndCompare(metrics.CompilationTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("observe compilation duration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CompilationDuration.WithLabelValues("go").Observe(5.0)
		metrics.CompilationDuration.WithLabelValues("rust").Observe(30.0)

		count := testutil.CollectAndCount(metrics.CompilationDuration)
		if count != 2 {
			t.Errorf("Expected 2 metric families, got %d", count)
		}
	})

	t.Run("record compilation errors", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CompilationErrorsTotal.WithLabelValues("go", "syntax").Inc()

		expected := `
# HELP spoke_compilation_errors_total Total number of compilation errors
# TYPE spoke_compilation_errors_total counter
spoke_compilation_errors_total{error_type="syntax",language="go"} 1
`
		if err := testutil.CollectAndCompare(metrics.CompilationErrorsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})
}

func TestMetrics_CacheMetrics(t *testing.T) {
	t.Run("record cache hits", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CacheHitsTotal.WithLabelValues("memory", "module").Inc()

		expected := `
# HELP spoke_cache_hits_total Total number of cache hits
# TYPE spoke_cache_hits_total counter
spoke_cache_hits_total{cache_type="memory",key_type="module"} 1
`
		if err := testutil.CollectAndCompare(metrics.CacheHitsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("record cache misses", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CacheMissesTotal.WithLabelValues("redis", "version").Inc()

		expected := `
# HELP spoke_cache_misses_total Total number of cache misses
# TYPE spoke_cache_misses_total counter
spoke_cache_misses_total{cache_type="redis",key_type="version"} 1
`
		if err := testutil.CollectAndCompare(metrics.CacheMissesTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("record cache evictions", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CacheEvictionsTotal.WithLabelValues("memory", "size_limit").Inc()

		expected := `
# HELP spoke_cache_evictions_total Total number of cache evictions
# TYPE spoke_cache_evictions_total counter
spoke_cache_evictions_total{cache_type="memory",reason="size_limit"} 1
`
		if err := testutil.CollectAndCompare(metrics.CacheEvictionsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("set cache size", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.CacheSizeBytes.WithLabelValues("memory").Set(1024 * 1024)

		expected := `
# HELP spoke_cache_size_bytes Current cache size in bytes
# TYPE spoke_cache_size_bytes gauge
spoke_cache_size_bytes{cache_type="memory"} 1.048576e+06
`
		if err := testutil.CollectAndCompare(metrics.CacheSizeBytes, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})
}

func TestMetrics_DatabaseMetrics(t *testing.T) {
	t.Run("set database connections", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.DBConnectionsActive.Set(10)
		metrics.DBConnectionsIdle.Set(5)
		metrics.DBConnectionsWaitCount.Set(2)
		metrics.DBConnectionsWaitDuration.Set(0.05)

		// Verify metrics can be collected
		count := testutil.CollectAndCount(metrics.DBConnectionsActive)
		if count != 1 {
			t.Errorf("Expected 1 metric, got %d", count)
		}

		// Test increment and decrement
		metrics.DBConnectionsActive.Inc()
		metrics.DBConnectionsIdle.Dec()

		expected := `
# HELP spoke_db_connections_active Number of active database connections
# TYPE spoke_db_connections_active gauge
spoke_db_connections_active 11
`
		if err := testutil.CollectAndCompare(metrics.DBConnectionsActive, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})
}

func TestMetrics_RedisMetrics(t *testing.T) {
	t.Run("set redis connections", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.RedisConnectionsActive.Set(8)

		expected := `
# HELP spoke_redis_connections_active Number of active Redis connections
# TYPE spoke_redis_connections_active gauge
spoke_redis_connections_active 8
`
		if err := testutil.CollectAndCompare(metrics.RedisConnectionsActive, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("record redis commands", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.RedisCommandsTotal.WithLabelValues("GET", "success").Inc()
		metrics.RedisCommandsTotal.WithLabelValues("SET", "success").Inc()

		expected := `
# HELP spoke_redis_commands_total Total number of Redis commands
# TYPE spoke_redis_commands_total counter
spoke_redis_commands_total{command="GET",status="success"} 1
spoke_redis_commands_total{command="SET",status="success"} 1
`
		if err := testutil.CollectAndCompare(metrics.RedisCommandsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("observe redis command duration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.RedisCommandDuration.WithLabelValues("GET").Observe(0.001)

		count := testutil.CollectAndCount(metrics.RedisCommandDuration)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})
}

func TestMetrics_BusinessMetrics(t *testing.T) {
	t.Run("set business metrics", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.ModulesTotal.Set(100)
		metrics.VersionsTotal.Set(500)
		metrics.ActiveUsersTotal.Set(25)
		metrics.APITokensActive.Set(10)

		expected := `
# HELP spoke_modules_total Total number of modules
# TYPE spoke_modules_total gauge
spoke_modules_total 100
`
		if err := testutil.CollectAndCompare(metrics.ModulesTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}

		expected = `
# HELP spoke_versions_total Total number of versions
# TYPE spoke_versions_total gauge
spoke_versions_total 500
`
		if err := testutil.CollectAndCompare(metrics.VersionsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		rw.WriteHeader(http.StatusCreated)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("Expected status code %d, got %d", http.StatusCreated, rw.statusCode)
		}

		if recorder.Code != http.StatusCreated {
			t.Errorf("Expected recorder status code %d, got %d", http.StatusCreated, recorder.Code)
		}
	})

	t.Run("captures bytes written", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		data := []byte("Hello, World!")
		n, err := rw.Write(data)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if n != len(data) {
			t.Errorf("Expected %d bytes written, got %d", len(data), n)
		}

		if rw.bytesWritten != len(data) {
			t.Errorf("Expected %d bytes tracked, got %d", len(data), rw.bytesWritten)
		}
	})

	t.Run("accumulates bytes across multiple writes", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		rw.Write([]byte("Hello, "))
		rw.Write([]byte("World!"))

		expected := len("Hello, ") + len("World!")
		if rw.bytesWritten != expected {
			t.Errorf("Expected %d bytes written, got %d", expected, rw.bytesWritten)
		}
	})

	t.Run("defaults to 200 status code", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		// Write without calling WriteHeader
		rw.Write([]byte("test"))

		if rw.statusCode != http.StatusOK {
			t.Errorf("Expected default status code %d, got %d", http.StatusOK, rw.statusCode)
		}
	})
}

func TestHTTPMetricsMiddleware(t *testing.T) {
	t.Run("records HTTP metrics", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		// Verify counter was incremented
		expected := `
# HELP spoke_http_requests_total Total number of HTTP requests
# TYPE spoke_http_requests_total counter
spoke_http_requests_total{method="GET",path="/test",status="200"} 1
`
		if err := testutil.CollectAndCompare(metrics.HTTPRequestsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected counter value: %v", err)
		}

		// Verify duration was recorded
		count := testutil.CollectAndCount(metrics.HTTPRequestDuration)
		if count != 1 {
			t.Errorf("Expected 1 duration metric, got %d", count)
		}

		// Verify response size was recorded
		count = testutil.CollectAndCount(metrics.HTTPResponseSize)
		if count != 1 {
			t.Errorf("Expected 1 response size metric, got %d", count)
		}
	})

	t.Run("records different status codes", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		testCases := []struct {
			statusCode int
			path       string
		}{
			{http.StatusOK, "/ok"},
			{http.StatusNotFound, "/notfound"},
			{http.StatusInternalServerError, "/error"},
		}

		middleware := HTTPMetricsMiddleware(metrics)

		for _, tc := range testCases {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			wrappedHandler := middleware(handler)
			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)
		}

		// Verify all status codes were recorded
		count := testutil.CollectAndCount(metrics.HTTPRequestsTotal)
		if count != 3 {
			t.Errorf("Expected 3 metrics, got %d", count)
		}
	})

	t.Run("records request size with content length", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(handler)

		body := strings.NewReader("test body content")
		req := httptest.NewRequest("POST", "/upload", body)
		req.ContentLength = int64(body.Len())
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		// Verify request size was recorded
		count := testutil.CollectAndCount(metrics.HTTPRequestSize)
		if count != 1 {
			t.Errorf("Expected 1 request size metric, got %d", count)
		}
	})

	t.Run("skips request size when content length is 0", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		// Request size should not be recorded for GET without body
		count := testutil.CollectAndCount(metrics.HTTPRequestSize)
		if count != 0 {
			t.Errorf("Expected 0 request size metrics, got %d", count)
		}
	})

	t.Run("measures request duration", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(handler)

		req := httptest.NewRequest("GET", "/slow", nil)
		rec := httptest.NewRecorder()

		start := time.Now()
		wrappedHandler.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		if elapsed < 10*time.Millisecond {
			t.Error("Expected handler to take at least 10ms")
		}

		// Verify duration was recorded
		count := testutil.CollectAndCount(metrics.HTTPRequestDuration)
		if count != 1 {
			t.Errorf("Expected 1 duration metric, got %d", count)
		}
	})

	t.Run("handles multiple requests", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(handler)

		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			rec := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(rec, req)
		}

		expected := `
# HELP spoke_http_requests_total Total number of HTTP requests
# TYPE spoke_http_requests_total counter
spoke_http_requests_total{method="GET",path="/test",status="200"} 5
`
		if err := testutil.CollectAndCompare(metrics.HTTPRequestsTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected counter value: %v", err)
		}
	})
}

func TestRegisterMetricsEndpoint(t *testing.T) {
	t.Run("registers metrics endpoint", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Set some metric values
		metrics.ModulesTotal.Set(42)
		metrics.HTTPRequestsTotal.WithLabelValues("GET", "/api", "200").Inc()

		mux := http.NewServeMux()
		RegisterMetricsEndpoint(mux, registry)

		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		body := rec.Body.String()

		// Verify metrics are exposed
		if !strings.Contains(body, "spoke_modules_total") {
			t.Error("Expected spoke_modules_total in metrics output")
		}

		if !strings.Contains(body, "spoke_modules_total 42") {
			t.Error("Expected spoke_modules_total value to be 42")
		}

		if !strings.Contains(body, "spoke_http_requests_total") {
			t.Error("Expected spoke_http_requests_total in metrics output")
		}
	})

	t.Run("metrics endpoint returns prometheus format", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		NewMetrics(registry)

		mux := http.NewServeMux()
		RegisterMetricsEndpoint(mux, registry)

		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Errorf("Expected Content-Type to contain text/plain, got %s", contentType)
		}

		body := rec.Body.String()

		// Verify Prometheus format markers
		if !strings.Contains(body, "# HELP") {
			t.Error("Expected # HELP lines in output")
		}

		if !strings.Contains(body, "# TYPE") {
			t.Error("Expected # TYPE lines in output")
		}
	})

	t.Run("metrics endpoint can be called multiple times", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)
		metrics.VersionsTotal.Set(10)

		mux := http.NewServeMux()
		RegisterMetricsEndpoint(mux, registry)

		// Call multiple times
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest("GET", "/metrics", nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status code %d, got %d", i, http.StatusOK, rec.Code)
			}

			body := rec.Body.String()
			if !strings.Contains(body, "spoke_versions_total 10") {
				t.Errorf("Request %d: Expected spoke_versions_total value to be 10", i)
			}
		}
	})

	t.Run("metrics endpoint only responds to /metrics path", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		NewMetrics(registry)

		mux := http.NewServeMux()
		RegisterMetricsEndpoint(mux, registry)

		req := httptest.NewRequest("GET", "/other", nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d for non-metrics path, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

func TestMetrics_Integration(t *testing.T) {
	t.Run("full workflow with middleware and exposition", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Create an application handler
		appHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!"))
		})

		// Wrap with metrics middleware
		middleware := HTTPMetricsMiddleware(metrics)
		wrappedHandler := middleware(appHandler)

		// Create mux and register both app and metrics endpoints
		mux := http.NewServeMux()
		mux.Handle("/api/hello", wrappedHandler)
		RegisterMetricsEndpoint(mux, registry)

		// Make a request to the app
		req := httptest.NewRequest("GET", "/api/hello", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}

		// Fetch metrics
		metricsReq := httptest.NewRequest("GET", "/metrics", nil)
		metricsRec := httptest.NewRecorder()
		mux.ServeHTTP(metricsRec, metricsReq)

		if metricsRec.Code != http.StatusOK {
			t.Errorf("Expected metrics status code %d, got %d", http.StatusOK, metricsRec.Code)
		}

		body := metricsRec.Body.String()

		// Verify the app request was recorded in metrics
		if !strings.Contains(body, "spoke_http_requests_total") {
			t.Error("Expected spoke_http_requests_total in metrics")
		}

		if !strings.Contains(body, `method="GET"`) {
			t.Error("Expected GET method label in metrics")
		}

		if !strings.Contains(body, `path="/api/hello"`) {
			t.Error("Expected /api/hello path label in metrics")
		}

		if !strings.Contains(body, `status="200"`) {
			t.Error("Expected 200 status label in metrics")
		}
	})

	t.Run("records multiple label combinations", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Record multiple storage operations
		metrics.StorageOperationsTotal.WithLabelValues("read", "s3", "success").Add(10)
		metrics.StorageOperationsTotal.WithLabelValues("write", "s3", "success").Add(5)
		metrics.StorageOperationsTotal.WithLabelValues("read", "local", "success").Add(20)
		metrics.StorageOperationsTotal.WithLabelValues("write", "s3", "error").Add(2)

		mux := http.NewServeMux()
		RegisterMetricsEndpoint(mux, registry)

		req := httptest.NewRequest("GET", "/metrics", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		body := rec.Body.String()

		// Verify all label combinations are present
		expectedPatterns := []string{
			`spoke_storage_operations_total{backend="s3",operation="read",status="success"} 10`,
			`spoke_storage_operations_total{backend="s3",operation="write",status="success"} 5`,
			`spoke_storage_operations_total{backend="local",operation="read",status="success"} 20`,
			`spoke_storage_operations_total{backend="s3",operation="write",status="error"} 2`,
		}

		for _, pattern := range expectedPatterns {
			if !strings.Contains(body, pattern) {
				t.Errorf("Expected pattern %q not found in metrics output", pattern)
			}
		}
	})
}

func TestMetrics_EdgeCases(t *testing.T) {
	t.Run("large metric values", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		largeValue := float64(1000000000) // 1 billion
		metrics.ModulesTotal.Set(largeValue)

		expected := `
# HELP spoke_modules_total Total number of modules
# TYPE spoke_modules_total gauge
spoke_modules_total 1e+09
`
		if err := testutil.CollectAndCompare(metrics.ModulesTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("zero values", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		metrics.ActiveUsersTotal.Set(0)

		expected := `
# HELP spoke_active_users_total Total number of active users
# TYPE spoke_active_users_total gauge
spoke_active_users_total 0
`
		if err := testutil.CollectAndCompare(metrics.ActiveUsersTotal, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("negative gauge values", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// While unusual, gauges can technically be negative
		metrics.DBConnectionsActive.Set(10)
		metrics.DBConnectionsActive.Sub(15)

		expected := `
# HELP spoke_db_connections_active Number of active database connections
# TYPE spoke_db_connections_active gauge
spoke_db_connections_active -5
`
		if err := testutil.CollectAndCompare(metrics.DBConnectionsActive, strings.NewReader(expected)); err != nil {
			t.Errorf("Unexpected metric value: %v", err)
		}
	})

	t.Run("histogram with extreme values", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Record very small and very large durations
		metrics.CompilationDuration.WithLabelValues("go").Observe(0.001)
		metrics.CompilationDuration.WithLabelValues("go").Observe(299.999)

		count := testutil.CollectAndCount(metrics.CompilationDuration)
		if count != 1 {
			t.Errorf("Expected 1 metric family, got %d", count)
		}
	})

	t.Run("empty response body", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusNoContent,
		}

		rw.WriteHeader(http.StatusNoContent)

		if rw.bytesWritten != 0 {
			t.Errorf("Expected 0 bytes written, got %d", rw.bytesWritten)
		}
	})

	t.Run("special characters in labels", func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := NewMetrics(registry)

		// Labels with special characters
		metrics.HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/users/{id}", "200").Inc()

		count := testutil.CollectAndCount(metrics.HTTPRequestsTotal)
		if count != 1 {
			t.Errorf("Expected 1 metric, got %d", count)
		}
	})
}

func BenchmarkHTTPMetricsMiddleware(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := HTTPMetricsMiddleware(metrics)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkMetricsCollection(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test", "200").Inc()
		metrics.HTTPRequestDuration.WithLabelValues("GET", "/test").Observe(0.1)
		metrics.StorageOperationsTotal.WithLabelValues("read", "s3", "success").Inc()
		metrics.CacheHitsTotal.WithLabelValues("memory", "module").Inc()
	}
}

func BenchmarkResponseWriter(b *testing.B) {
	data := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		rw := &responseWriter{
			ResponseWriter: recorder,
			statusCode:     http.StatusOK,
		}

		rw.Write(data)
	}
}

func ExampleMetrics() {
	// Create a new registry and metrics
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Record some metrics
	metrics.HTTPRequestsTotal.WithLabelValues("GET", "/api/users", "200").Inc()
	metrics.HTTPRequestDuration.WithLabelValues("GET", "/api/users").Observe(0.123)
	metrics.StorageOperationsTotal.WithLabelValues("read", "s3", "success").Inc()
	metrics.CacheHitsTotal.WithLabelValues("memory", "user").Inc()

	// Set gauge values
	metrics.ModulesTotal.Set(100)
	metrics.ActiveUsersTotal.Set(42)

	// Create HTTP server with metrics
	mux := http.NewServeMux()
	RegisterMetricsEndpoint(mux, registry)

	// The metrics are now available at /metrics endpoint
}

func ExampleHTTPMetricsMiddleware() {
	registry := prometheus.NewRegistry()
	metrics := NewMetrics(registry)

	// Create your application handler
	appHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "Hello, World!")
	})

	// Wrap with metrics middleware
	middleware := HTTPMetricsMiddleware(metrics)
	instrumentedHandler := middleware(appHandler)

	// Use the instrumented handler
	mux := http.NewServeMux()
	mux.Handle("/", instrumentedHandler)
	RegisterMetricsEndpoint(mux, registry)

	// All requests will be automatically instrumented
}
