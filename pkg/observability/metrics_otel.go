package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// OTelMetrics holds OpenTelemetry metric instruments
type OTelMetrics struct {
	// HTTP metrics
	httpRequestsTotal    metric.Int64Counter
	httpRequestDuration  metric.Float64Histogram
	httpRequestSize      metric.Int64Histogram
	httpResponseSize     metric.Int64Histogram

	// Database metrics
	dbConnectionsActive  metric.Int64UpDownCounter
	dbConnectionsIdle    metric.Int64UpDownCounter
	dbConnectionsMax     metric.Int64Gauge
	dbQueryDuration      metric.Float64Histogram
	dbQueriesTotal       metric.Int64Counter

	// Cache metrics
	cacheHitsTotal       metric.Int64Counter
	cacheMissesTotal     metric.Int64Counter
	cacheEvictionsTotal  metric.Int64Counter
	cacheSize            metric.Int64UpDownCounter

	// Storage metrics
	storageOperations    metric.Int64Counter
	storageDuration      metric.Float64Histogram
	storageBytes         metric.Int64Histogram
}

// NewOTelMetrics creates a new OTel metrics instance
func NewOTelMetrics() (*OTelMetrics, error) {
	meter := otel.Meter("github.com/platinummonkey/spoke")

	m := &OTelMetrics{}
	var err error

	// HTTP metrics
	m.httpRequestsTotal, err = meter.Int64Counter(
		"http.server.requests",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_requests_total counter: %w", err)
	}

	m.httpRequestDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_request_duration histogram: %w", err)
	}

	m.httpRequestSize, err = meter.Int64Histogram(
		"http.server.request.size",
		metric.WithDescription("HTTP request size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_request_size histogram: %w", err)
	}

	m.httpResponseSize, err = meter.Int64Histogram(
		"http.server.response.size",
		metric.WithDescription("HTTP response size in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_response_size histogram: %w", err)
	}

	// Database metrics
	m.dbConnectionsActive, err = meter.Int64UpDownCounter(
		"db.connections.active",
		metric.WithDescription("Number of active database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db_connections_active gauge: %w", err)
	}

	m.dbConnectionsIdle, err = meter.Int64UpDownCounter(
		"db.connections.idle",
		metric.WithDescription("Number of idle database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db_connections_idle gauge: %w", err)
	}

	m.dbConnectionsMax, err = meter.Int64Gauge(
		"db.connections.max",
		metric.WithDescription("Maximum number of database connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db_connections_max gauge: %w", err)
	}

	m.dbQueryDuration, err = meter.Float64Histogram(
		"db.query.duration",
		metric.WithDescription("Database query duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db_query_duration histogram: %w", err)
	}

	m.dbQueriesTotal, err = meter.Int64Counter(
		"db.queries.total",
		metric.WithDescription("Total number of database queries"),
		metric.WithUnit("{query}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create db_queries_total counter: %w", err)
	}

	// Cache metrics
	m.cacheHitsTotal, err = meter.Int64Counter(
		"cache.hits.total",
		metric.WithDescription("Total number of cache hits"),
		metric.WithUnit("{hit}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_hits_total counter: %w", err)
	}

	m.cacheMissesTotal, err = meter.Int64Counter(
		"cache.misses.total",
		metric.WithDescription("Total number of cache misses"),
		metric.WithUnit("{miss}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_misses_total counter: %w", err)
	}

	m.cacheEvictionsTotal, err = meter.Int64Counter(
		"cache.evictions.total",
		metric.WithDescription("Total number of cache evictions"),
		metric.WithUnit("{eviction}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_evictions_total counter: %w", err)
	}

	m.cacheSize, err = meter.Int64UpDownCounter(
		"cache.size",
		metric.WithDescription("Current cache size"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_size gauge: %w", err)
	}

	// Storage metrics
	m.storageOperations, err = meter.Int64Counter(
		"storage.operations.total",
		metric.WithDescription("Total number of storage operations"),
		metric.WithUnit("{operation}"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage_operations counter: %w", err)
	}

	m.storageDuration, err = meter.Float64Histogram(
		"storage.operation.duration",
		metric.WithDescription("Storage operation duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage_duration histogram: %w", err)
	}

	m.storageBytes, err = meter.Int64Histogram(
		"storage.bytes",
		metric.WithDescription("Storage operation bytes transferred"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage_bytes histogram: %w", err)
	}

	return m, nil
}

// RecordHTTPRequest records an HTTP request metric
func (m *OTelMetrics) RecordHTTPRequest(ctx context.Context, method, route string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	attrs := []attribute.KeyValue{
		attribute.String("http.method", method),
		attribute.String("http.route", route),
		attribute.Int("http.status_code", statusCode),
	}

	m.httpRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if requestSize > 0 {
		m.httpRequestSize.Record(ctx, requestSize, metric.WithAttributes(attrs...))
	}
	if responseSize > 0 {
		m.httpResponseSize.Record(ctx, responseSize, metric.WithAttributes(attrs...))
	}
}

// RecordDBQuery records a database query metric
func (m *OTelMetrics) RecordDBQuery(ctx context.Context, operation string, duration time.Duration, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("db.operation", operation),
	}
	if err != nil {
		attrs = append(attrs, attribute.String("error", "true"))
	} else {
		attrs = append(attrs, attribute.String("error", "false"))
	}

	m.dbQueriesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.dbQueryDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// UpdateDBConnectionStats updates database connection pool statistics
func (m *OTelMetrics) UpdateDBConnectionStats(ctx context.Context, active, idle, max int) {
	m.dbConnectionsActive.Add(ctx, int64(active))
	m.dbConnectionsIdle.Add(ctx, int64(idle))
}

// RecordCacheHit records a cache hit
func (m *OTelMetrics) RecordCacheHit(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.type", cacheType),
	}
	m.cacheHitsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss
func (m *OTelMetrics) RecordCacheMiss(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.type", cacheType),
	}
	m.cacheMissesTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheEviction records a cache eviction
func (m *OTelMetrics) RecordCacheEviction(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.type", cacheType),
	}
	m.cacheEvictionsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// UpdateCacheSize updates the cache size metric
func (m *OTelMetrics) UpdateCacheSize(ctx context.Context, cacheType string, size int64) {
	attrs := []attribute.KeyValue{
		attribute.String("cache.type", cacheType),
	}
	m.cacheSize.Add(ctx, size, metric.WithAttributes(attrs...))
}

// RecordStorageOperation records a storage operation metric
func (m *OTelMetrics) RecordStorageOperation(ctx context.Context, operation, storageType string, duration time.Duration, bytes int64, err error) {
	attrs := []attribute.KeyValue{
		attribute.String("storage.operation", operation),
		attribute.String("storage.type", storageType),
	}
	if err != nil {
		attrs = append(attrs, attribute.String("error", "true"))
	} else {
		attrs = append(attrs, attribute.String("error", "false"))
	}

	m.storageOperations.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.storageDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))

	if bytes > 0 {
		m.storageBytes.Record(ctx, bytes, metric.WithAttributes(attrs...))
	}
}
