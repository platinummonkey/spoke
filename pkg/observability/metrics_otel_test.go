package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// setupTestMeterProvider creates a test meter provider with a manual reader
func setupTestMeterProvider(t *testing.T) (*metric.MeterProvider, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := metric.NewMeterProvider(metric.WithReader(reader))
	otel.SetMeterProvider(provider)
	return provider, reader
}

func TestNewOTelMetrics(t *testing.T) {
	t.Run("successful initialization", func(t *testing.T) {
		provider, _ := setupTestMeterProvider(t)
		defer func() {
			if err := provider.Shutdown(context.Background()); err != nil {
				t.Logf("Error shutting down provider: %v", err)
			}
		}()

		m, err := NewOTelMetrics()
		if err != nil {
			t.Fatalf("NewOTelMetrics() error = %v, want nil", err)
		}

		if m == nil {
			t.Fatal("NewOTelMetrics() returned nil metrics")
		}

		// Verify that all metric instruments are initialized
		if m.httpRequestsTotal == nil {
			t.Error("httpRequestsTotal is nil")
		}
		if m.httpRequestDuration == nil {
			t.Error("httpRequestDuration is nil")
		}
		if m.httpRequestSize == nil {
			t.Error("httpRequestSize is nil")
		}
		if m.httpResponseSize == nil {
			t.Error("httpResponseSize is nil")
		}
		if m.dbConnectionsActive == nil {
			t.Error("dbConnectionsActive is nil")
		}
		if m.dbConnectionsIdle == nil {
			t.Error("dbConnectionsIdle is nil")
		}
		if m.dbConnectionsMax == nil {
			t.Error("dbConnectionsMax is nil")
		}
		if m.dbQueryDuration == nil {
			t.Error("dbQueryDuration is nil")
		}
		if m.dbQueriesTotal == nil {
			t.Error("dbQueriesTotal is nil")
		}
		if m.cacheHitsTotal == nil {
			t.Error("cacheHitsTotal is nil")
		}
		if m.cacheMissesTotal == nil {
			t.Error("cacheMissesTotal is nil")
		}
		if m.cacheEvictionsTotal == nil {
			t.Error("cacheEvictionsTotal is nil")
		}
		if m.cacheSize == nil {
			t.Error("cacheSize is nil")
		}
		if m.storageOperations == nil {
			t.Error("storageOperations is nil")
		}
		if m.storageDuration == nil {
			t.Error("storageDuration is nil")
		}
		if m.storageBytes == nil {
			t.Error("storageBytes is nil")
		}
	})
}

func TestOTelMetrics_RecordHTTPRequest(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		route        string
		statusCode   int
		duration     time.Duration
		requestSize  int64
		responseSize int64
	}{
		{
			name:         "successful GET request",
			method:       "GET",
			route:        "/api/v1/users",
			statusCode:   200,
			duration:     100 * time.Millisecond,
			requestSize:  0,
			responseSize: 1024,
		},
		{
			name:         "POST request with request body",
			method:       "POST",
			route:        "/api/v1/users",
			statusCode:   201,
			duration:     250 * time.Millisecond,
			requestSize:  512,
			responseSize: 256,
		},
		{
			name:         "error response",
			method:       "GET",
			route:        "/api/v1/users/123",
			statusCode:   404,
			duration:     50 * time.Millisecond,
			requestSize:  0,
			responseSize: 128,
		},
		{
			name:         "zero sizes",
			method:       "DELETE",
			route:        "/api/v1/users/123",
			statusCode:   204,
			duration:     75 * time.Millisecond,
			requestSize:  0,
			responseSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordHTTPRequest(ctx, tt.method, tt.route, tt.statusCode, tt.duration, tt.requestSize, tt.responseSize)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metrics were recorded
			if len(rm.ScopeMetrics) == 0 {
				t.Error("No scope metrics recorded")
				return
			}

			foundCounter := false
			foundDuration := false
			foundRequestSize := false
			foundResponseSize := false

			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					switch m.Name {
					case "http.server.requests":
						foundCounter = true
						if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
							if len(sum.DataPoints) > 0 && sum.DataPoints[0].Value != 1 {
								t.Errorf("Expected counter value 1, got %d", sum.DataPoints[0].Value)
							}
						}
					case "http.server.duration":
						foundDuration = true
					case "http.server.request.size":
						if tt.requestSize > 0 {
							foundRequestSize = true
						}
					case "http.server.response.size":
						if tt.responseSize > 0 {
							foundResponseSize = true
						}
					}
				}
			}

			if !foundCounter {
				t.Error("HTTP request counter not recorded")
			}
			if !foundDuration {
				t.Error("HTTP request duration not recorded")
			}
			if tt.requestSize > 0 && !foundRequestSize {
				t.Error("HTTP request size not recorded when requestSize > 0")
			}
			if tt.responseSize > 0 && !foundResponseSize {
				t.Error("HTTP response size not recorded when responseSize > 0")
			}
		})
	}
}

func TestOTelMetrics_RecordDBQuery(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		duration  time.Duration
		err       error
	}{
		{
			name:      "successful SELECT",
			operation: "SELECT",
			duration:  50 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "successful INSERT",
			operation: "INSERT",
			duration:  100 * time.Millisecond,
			err:       nil,
		},
		{
			name:      "failed UPDATE",
			operation: "UPDATE",
			duration:  75 * time.Millisecond,
			err:       errors.New("constraint violation"),
		},
		{
			name:      "failed DELETE",
			operation: "DELETE",
			duration:  25 * time.Millisecond,
			err:       errors.New("connection timeout"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordDBQuery(ctx, tt.operation, tt.duration, tt.err)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metrics were recorded
			foundCounter := false
			foundDuration := false

			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					switch m.Name {
					case "db.queries.total":
						foundCounter = true
						if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
							if len(sum.DataPoints) > 0 && sum.DataPoints[0].Value != 1 {
								t.Errorf("Expected counter value 1, got %d", sum.DataPoints[0].Value)
							}
						}
					case "db.query.duration":
						foundDuration = true
					}
				}
			}

			if !foundCounter {
				t.Error("DB queries counter not recorded")
			}
			if !foundDuration {
				t.Error("DB query duration not recorded")
			}
		})
	}
}

func TestOTelMetrics_UpdateDBConnectionStats(t *testing.T) {
	tests := []struct {
		name   string
		active int
		idle   int
		max    int
	}{
		{
			name:   "typical connection pool",
			active: 5,
			idle:   3,
			max:    10,
		},
		{
			name:   "fully utilized pool",
			active: 10,
			idle:   0,
			max:    10,
		},
		{
			name:   "idle pool",
			active: 0,
			idle:   10,
			max:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.UpdateDBConnectionStats(ctx, tt.active, tt.idle, tt.max)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metrics were recorded
			foundActive := false
			foundIdle := false

			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					switch m.Name {
					case "db.connections.active":
						foundActive = true
					case "db.connections.idle":
						foundIdle = true
					}
				}
			}

			if !foundActive {
				t.Error("DB connections active metric not recorded")
			}
			if !foundIdle {
				t.Error("DB connections idle metric not recorded")
			}
		})
	}
}

func TestOTelMetrics_RecordCacheHit(t *testing.T) {
	tests := []struct {
		name      string
		cacheType string
	}{
		{
			name:      "redis cache hit",
			cacheType: "redis",
		},
		{
			name:      "memory cache hit",
			cacheType: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordCacheHit(ctx, tt.cacheType)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metric was recorded
			found := false
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "cache.hits.total" {
						found = true
						break
					}
				}
			}

			if !found {
				t.Error("Cache hits counter not recorded")
			}
		})
	}
}

func TestOTelMetrics_RecordCacheMiss(t *testing.T) {
	tests := []struct {
		name      string
		cacheType string
	}{
		{
			name:      "redis cache miss",
			cacheType: "redis",
		},
		{
			name:      "memory cache miss",
			cacheType: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordCacheMiss(ctx, tt.cacheType)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metric was recorded
			found := false
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "cache.misses.total" {
						found = true
						break
					}
				}
			}

			if !found {
				t.Error("Cache misses counter not recorded")
			}
		})
	}
}

func TestOTelMetrics_RecordCacheEviction(t *testing.T) {
	tests := []struct {
		name      string
		cacheType string
	}{
		{
			name:      "redis cache eviction",
			cacheType: "redis",
		},
		{
			name:      "memory cache eviction",
			cacheType: "memory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordCacheEviction(ctx, tt.cacheType)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metric was recorded
			found := false
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "cache.evictions.total" {
						found = true
						break
					}
				}
			}

			if !found {
				t.Error("Cache evictions counter not recorded")
			}
		})
	}
}

func TestOTelMetrics_UpdateCacheSize(t *testing.T) {
	tests := []struct {
		name      string
		cacheType string
		size      int64
	}{
		{
			name:      "increase cache size",
			cacheType: "redis",
			size:      1024,
		},
		{
			name:      "decrease cache size",
			cacheType: "memory",
			size:      -512,
		},
		{
			name:      "zero cache size",
			cacheType: "memory",
			size:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.UpdateCacheSize(ctx, tt.cacheType, tt.size)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metric was recorded
			found := false
			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					if m.Name == "cache.size" {
						found = true
						break
					}
				}
			}

			if !found {
				t.Error("Cache size metric not recorded")
			}
		})
	}
}

func TestOTelMetrics_RecordStorageOperation(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		storageType string
		duration    time.Duration
		bytes       int64
		err         error
	}{
		{
			name:        "successful read",
			operation:   "read",
			storageType: "s3",
			duration:    100 * time.Millisecond,
			bytes:       2048,
			err:         nil,
		},
		{
			name:        "successful write",
			operation:   "write",
			storageType: "s3",
			duration:    200 * time.Millisecond,
			bytes:       4096,
			err:         nil,
		},
		{
			name:        "failed read",
			operation:   "read",
			storageType: "gcs",
			duration:    50 * time.Millisecond,
			bytes:       0,
			err:         errors.New("object not found"),
		},
		{
			name:        "delete operation",
			operation:   "delete",
			storageType: "local",
			duration:    25 * time.Millisecond,
			bytes:       0,
			err:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, reader := setupTestMeterProvider(t)
			defer func() {
				if err := provider.Shutdown(context.Background()); err != nil {
					t.Logf("Error shutting down provider: %v", err)
				}
			}()

			m, err := NewOTelMetrics()
			if err != nil {
				t.Fatalf("NewOTelMetrics() error = %v", err)
			}

			ctx := context.Background()
			m.RecordStorageOperation(ctx, tt.operation, tt.storageType, tt.duration, tt.bytes, tt.err)

			// Collect metrics
			var rm metricdata.ResourceMetrics
			err = reader.Collect(ctx, &rm)
			if err != nil {
				t.Fatalf("Failed to collect metrics: %v", err)
			}

			// Verify metrics were recorded
			foundCounter := false
			foundDuration := false
			foundBytes := false

			for _, sm := range rm.ScopeMetrics {
				for _, m := range sm.Metrics {
					switch m.Name {
					case "storage.operations.total":
						foundCounter = true
					case "storage.operation.duration":
						foundDuration = true
					case "storage.bytes":
						if tt.bytes > 0 {
							foundBytes = true
						}
					}
				}
			}

			if !foundCounter {
				t.Error("Storage operations counter not recorded")
			}
			if !foundDuration {
				t.Error("Storage operation duration not recorded")
			}
			if tt.bytes > 0 && !foundBytes {
				t.Error("Storage bytes not recorded when bytes > 0")
			}
		})
	}
}

func TestOTelMetrics_MultipleOperations(t *testing.T) {
	t.Run("multiple HTTP requests", func(t *testing.T) {
		provider, reader := setupTestMeterProvider(t)
		defer func() {
			if err := provider.Shutdown(context.Background()); err != nil {
				t.Logf("Error shutting down provider: %v", err)
			}
		}()

		m, err := NewOTelMetrics()
		if err != nil {
			t.Fatalf("NewOTelMetrics() error = %v", err)
		}

		ctx := context.Background()

		// Record multiple requests
		for i := 0; i < 5; i++ {
			m.RecordHTTPRequest(ctx, "GET", "/api/v1/users", 200, 100*time.Millisecond, 0, 1024)
		}

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Verify counter incremented correctly
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				if m.Name == "http.server.requests" {
					if sum, ok := m.Data.(metricdata.Sum[int64]); ok {
						if len(sum.DataPoints) > 0 && sum.DataPoints[0].Value != 5 {
							t.Errorf("Expected counter value 5, got %d", sum.DataPoints[0].Value)
						}
					}
				}
			}
		}
	})

	t.Run("mixed cache operations", func(t *testing.T) {
		provider, reader := setupTestMeterProvider(t)
		defer func() {
			if err := provider.Shutdown(context.Background()); err != nil {
				t.Logf("Error shutting down provider: %v", err)
			}
		}()

		m, err := NewOTelMetrics()
		if err != nil {
			t.Fatalf("NewOTelMetrics() error = %v", err)
		}

		ctx := context.Background()

		// Record various cache operations
		m.RecordCacheHit(ctx, "redis")
		m.RecordCacheHit(ctx, "redis")
		m.RecordCacheMiss(ctx, "redis")
		m.RecordCacheEviction(ctx, "redis")
		m.UpdateCacheSize(ctx, "redis", 1024)

		// Collect metrics
		var rm metricdata.ResourceMetrics
		err = reader.Collect(ctx, &rm)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Verify all cache metrics were recorded
		foundHits := false
		foundMisses := false
		foundEvictions := false
		foundSize := false

		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				switch m.Name {
				case "cache.hits.total":
					foundHits = true
				case "cache.misses.total":
					foundMisses = true
				case "cache.evictions.total":
					foundEvictions = true
				case "cache.size":
					foundSize = true
				}
			}
		}

		if !foundHits {
			t.Error("Cache hits not recorded")
		}
		if !foundMisses {
			t.Error("Cache misses not recorded")
		}
		if !foundEvictions {
			t.Error("Cache evictions not recorded")
		}
		if !foundSize {
			t.Error("Cache size not recorded")
		}
	})
}
