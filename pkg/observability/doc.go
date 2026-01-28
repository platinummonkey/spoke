// Package observability provides structured logging, Prometheus metrics, and OpenTelemetry tracing.
//
// # Overview
//
// This package centralizes observability infrastructure including JSON logging, metrics
// collection, health checks, and distributed tracing integration.
//
// # Structured Logging
//
// Create logger:
//
//	logger := observability.NewLogger(observability.LevelInfo)
//	logger.Info("Server started", "port", 8080)
//
// Context-aware logging:
//
//	logger.WithField("request_id", reqID).Error("Request failed", err)
//
// # Prometheus Metrics
//
// Initialize metrics:
//
//	metrics := observability.InitMetrics()
//	metrics.HTTPRequestsTotal.WithLabelValues("GET", "/modules", "200").Inc()
//	metrics.HTTPRequestDuration.WithLabelValues("GET", "/modules").Observe(0.123)
//
// Business metrics:
//
//	metrics.ModulesTotal.Set(float64(count))
//	metrics.ActiveUsersGauge.Set(float64(activeUsers))
//
// # Health Checks
//
// Configure health checker:
//
//	checker := observability.NewHealthChecker(db, redisClient)
//	status := checker.Check(ctx)
//	fmt.Printf("Healthy: %v\n", status.Healthy)
//
// # OpenTelemetry
//
// Initialize tracing:
//
//	providers, err := observability.InitOTel(&observability.OTelConfig{
//		ServiceName:    "spoke-api",
//		ServiceVersion: "v1.0.0",
//		OTLPEndpoint:   "otel-collector:4317",
//	})
//	defer providers.Shutdown(ctx)
//
// # Related Packages
//
//   - pkg/config: Observability configuration
//   - pkg/middleware: Request logging middleware
package observability
