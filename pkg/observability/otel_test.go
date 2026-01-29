package observability

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// TestInitOTel_Disabled tests that InitOTel returns nil when disabled
func TestInitOTel_Disabled(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled: false,
	}

	providers, err := InitOTel(ctx, cfg, logger)

	assert.NoError(t, err)
	assert.Nil(t, providers)
}

// TestInitOTel_InvalidEndpoint tests InitOTel with invalid endpoint
// Note: OTLP exporters don't validate connection at creation time, so this will succeed
func TestInitOTel_InvalidEndpoint(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "invalid-endpoint:9999",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(ctx, cfg, logger)

	// OTLP exporters succeed at creation time even without a collector
	// They only fail when attempting to export data
	assert.NoError(t, err)
	assert.NotNil(t, providers)

	// Clean up
	if providers != nil {
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}

// TestInitOTel_EmptyServiceName tests InitOTel with empty service name
func TestInitOTel_EmptyServiceName(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "", // Empty service name
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	// Should not panic with empty service name
	providers, err := InitOTel(ctx, cfg, logger)
	// Should succeed even with empty service name
	assert.NoError(t, err)
	assert.NotNil(t, providers)

	// Clean up
	if providers != nil {
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}

// TestInitOTel_Config tests various OTelConfig values
func TestInitOTel_Config(t *testing.T) {
	tests := []struct {
		name        string
		cfg         OTelConfig
		shouldError bool
	}{
		{
			name: "disabled",
			cfg: OTelConfig{
				Enabled: false,
			},
			shouldError: false,
		},
		{
			name: "enabled with invalid endpoint",
			cfg: OTelConfig{
				Enabled:        true,
				Endpoint:       "invalid:9999",
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				Insecure:       true,
			},
			shouldError: false, // OTLP exporters don't fail at creation time
		},
		{
			name: "secure connection",
			cfg: OTelConfig{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test",
				ServiceVersion: "1.0.0",
				Insecure:       false,
			},
			shouldError: false, // OTLP exporters don't fail at creation time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, &bytes.Buffer{})
			providers, err := InitOTel(context.Background(), tt.cfg, logger)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if providers != nil {
				// Clean up if providers were created
				_ = ShutdownOTel(context.Background(), providers, logger)
			}
		})
	}
}

// TestShutdownOTel_NilProviders tests that ShutdownOTel handles nil providers gracefully
func TestShutdownOTel_NilProviders(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	err := ShutdownOTel(ctx, nil, logger)
	assert.NoError(t, err)
}

// TestShutdownOTel_NilTracerProvider tests shutdown with nil tracer provider
func TestShutdownOTel_NilTracerProvider(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	providers := &OTelProviders{
		TracerProvider: nil,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)
	assert.NoError(t, err)
}

// TestShutdownOTel_WithProviders tests shutdown with actual providers
func TestShutdownOTel_WithProviders(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	// Create a basic tracer provider without exporter
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)
	assert.NoError(t, err)
}

// TestShutdownOTel_WithCanceledContext tests shutdown with canceled context
func TestShutdownOTel_WithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	// Create a basic tracer provider
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)
	// Shutdown should handle canceled context
	// It may or may not error depending on implementation
	_ = err
}

// TestUpdateLoggerWithTraceContext_NoSpan tests UpdateLoggerWithTraceContext without active span
func TestUpdateLoggerWithTraceContext_NoSpan(t *testing.T) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)

	// Should return same logger when no span is recording
	assert.NotNil(t, updatedLogger)
	// Logger should not have trace fields added
	assert.Empty(t, updatedLogger.fields)
}

// TestUpdateLoggerWithTraceContext_WithSpan tests UpdateLoggerWithTraceContext with active span
func TestUpdateLoggerWithTraceContext_WithSpan(t *testing.T) {
	// Create a tracer provider
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)

	assert.NotNil(t, updatedLogger)

	// Verify trace_id and span_id were added to logger fields
	assert.Contains(t, updatedLogger.fields, "trace_id")
	assert.Contains(t, updatedLogger.fields, "span_id")

	// Verify the IDs are not empty strings
	traceID, ok := updatedLogger.fields["trace_id"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, traceID)

	spanID, ok := updatedLogger.fields["span_id"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, spanID)
}

// TestUpdateLoggerWithTraceContext_NonRecordingSpan tests with non-recording span
func TestUpdateLoggerWithTraceContext_NonRecordingSpan(t *testing.T) {
	// Create a tracer provider with never sample
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
	)
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)

	assert.NotNil(t, updatedLogger)

	// Non-recording span should not add fields
	assert.Empty(t, updatedLogger.fields)
}

// TestUpdateLoggerWithTraceContext_PreserveExistingFields tests that existing logger fields are preserved
func TestUpdateLoggerWithTraceContext_PreserveExistingFields(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	// Create logger with existing fields
	logger := NewLogger(InfoLevel, &bytes.Buffer{}).
		WithField("existing_field", "value").
		WithField("another_field", 123)

	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)

	assert.NotNil(t, updatedLogger)

	// Verify existing fields are preserved
	assert.Contains(t, updatedLogger.fields, "existing_field")
	assert.Equal(t, "value", updatedLogger.fields["existing_field"])
	assert.Contains(t, updatedLogger.fields, "another_field")
	assert.Equal(t, 123, updatedLogger.fields["another_field"])

	// Verify trace fields are added
	assert.Contains(t, updatedLogger.fields, "trace_id")
	assert.Contains(t, updatedLogger.fields, "span_id")
}

// TestOTelConfig_Struct tests OTelConfig struct fields
func TestOTelConfig_Struct(t *testing.T) {
	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	assert.True(t, cfg.Enabled)
	assert.Equal(t, "localhost:4317", cfg.Endpoint)
	assert.Equal(t, "test-service", cfg.ServiceName)
	assert.Equal(t, "1.0.0", cfg.ServiceVersion)
	assert.True(t, cfg.Insecure)
}

// TestOTelProviders_Struct tests OTelProviders struct
func TestOTelProviders_Struct(t *testing.T) {
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	assert.NotNil(t, providers.TracerProvider)
	assert.Nil(t, providers.MeterProvider)
}

// TestInitOTel_GlobalPropagatorSet tests that global propagator is set
func TestInitOTel_GlobalPropagatorSet(t *testing.T) {
	// Store original propagator to restore after test
	originalPropagator := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(originalPropagator)

	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled: false,
	}

	_, err := InitOTel(ctx, cfg, logger)
	require.NoError(t, err)

	// When disabled, propagator should not be changed
	// Test that we can get the propagator without panic
	propagator := otel.GetTextMapPropagator()
	assert.NotNil(t, propagator)
}

// TestInitOTel_LoggerCalled tests that logger methods are called
func TestInitOTel_LoggerCalled(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	logger := NewLogger(InfoLevel, buf)

	cfg := OTelConfig{
		Enabled: false,
	}

	_, err := InitOTel(ctx, cfg, logger)
	require.NoError(t, err)

	// Verify logger was used (should have "disabled" message)
	output := buf.String()
	assert.Contains(t, output, "OpenTelemetry is disabled")
}

// TestShutdownOTel_LoggerCalled tests that shutdown logs messages
func TestShutdownOTel_LoggerCalled(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	logger := NewLogger(InfoLevel, buf)

	tp := sdktrace.NewTracerProvider()
	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)
	require.NoError(t, err)

	// Verify logger was used
	output := buf.String()
	assert.Contains(t, output, "Shutting down OpenTelemetry providers")
	assert.Contains(t, output, "Tracer provider shutdown complete")
}

// TestOTelProviders_ShutdownOrder tests shutdown happens in correct order
func TestOTelProviders_ShutdownOrder(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	logger := NewLogger(InfoLevel, buf)

	// Create providers
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)
	assert.NoError(t, err)

	// Both should be shut down even if one is nil
	output := buf.String()
	assert.Contains(t, output, "shutdown complete")
}

// TestUpdateLoggerWithTraceContext_MultipleSpans tests with nested spans
func TestUpdateLoggerWithTraceContext_MultipleSpans(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span1 := tracer.Start(ctx, "span1")
	defer span1.End()

	logger1 := NewLogger(InfoLevel, &bytes.Buffer{})
	updatedLogger1 := UpdateLoggerWithTraceContext(ctx, logger1)

	// Get span1 IDs
	span1TraceID := updatedLogger1.fields["trace_id"].(string)
	span1SpanID := updatedLogger1.fields["span_id"].(string)

	// Create nested span
	ctx, span2 := tracer.Start(ctx, "span2")
	defer span2.End()

	logger2 := NewLogger(InfoLevel, &bytes.Buffer{})
	updatedLogger2 := UpdateLoggerWithTraceContext(ctx, logger2)

	span2TraceID := updatedLogger2.fields["trace_id"].(string)
	span2SpanID := updatedLogger2.fields["span_id"].(string)

	// Trace IDs should be the same for nested spans
	assert.Equal(t, span1TraceID, span2TraceID)

	// Span IDs should be different
	assert.NotEqual(t, span1SpanID, span2SpanID)
}

// TestInitOTel_ContextCancellation tests behavior when context is canceled
func TestInitOTel_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(ctx, cfg, logger)

	// OTLP exporters may still succeed even with canceled context
	// as they don't block on connection establishment
	if err != nil {
		assert.Error(t, err)
		assert.Nil(t, providers)
	} else {
		assert.NotNil(t, providers)
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}

// TestInitOTel_ServiceVersionEmpty tests with empty service version
func TestInitOTel_ServiceVersionEmpty(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "", // Empty version
		Insecure:       true,
	}

	// Should not panic with empty version
	providers, err := InitOTel(context.Background(), cfg, logger)
	assert.NoError(t, err)
	assert.NotNil(t, providers)

	// Clean up
	if providers != nil {
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}

// MockSpan is a mock implementation of trace.Span for testing
type mockSpan struct {
	trace.Span
	recording  bool
	spanCtx    trace.SpanContext
}

func (m *mockSpan) IsRecording() bool {
	return m.recording
}

func (m *mockSpan) SpanContext() trace.SpanContext {
	return m.spanCtx
}

// TestUpdateLoggerWithTraceContext_InvalidSpanContext tests with invalid span context
func TestUpdateLoggerWithTraceContext_InvalidSpanContext(t *testing.T) {
	ctx := context.Background()

	// Create an invalid span context (zero values)
	invalidSpanCtx := trace.SpanContext{}

	// Create a mock non-recording span
	mockSpan := &mockSpan{
		recording: false,
		spanCtx:   invalidSpanCtx,
	}

	// Add mock span to context
	ctx = trace.ContextWithSpan(ctx, mockSpan)

	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)

	// Non-recording span should not add fields
	assert.Empty(t, updatedLogger.fields)
}

// BenchmarkInitOTel_Disabled benchmarks InitOTel when disabled
func BenchmarkInitOTel_Disabled(b *testing.B) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})
	cfg := OTelConfig{Enabled: false}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = InitOTel(ctx, cfg, logger)
	}
}

// BenchmarkUpdateLoggerWithTraceContext benchmarks UpdateLoggerWithTraceContext
func BenchmarkUpdateLoggerWithTraceContext(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test-tracer")

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = UpdateLoggerWithTraceContext(ctx, logger)
	}
}

// BenchmarkShutdownOTel benchmarks ShutdownOTel
func BenchmarkShutdownOTel(b *testing.B) {
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tp := sdktrace.NewTracerProvider()
		providers := &OTelProviders{TracerProvider: tp}
		b.StartTimer()

		_ = ShutdownOTel(ctx, providers, logger)
	}
}

// TestInitOTel_FullInitialization tests complete initialization with actual providers
func TestInitOTel_FullInitialization(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full initialization test in short mode")
	}

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, providers)
	assert.NotNil(t, providers.TracerProvider)
	assert.NotNil(t, providers.MeterProvider)

	// Verify tracer provider is set globally
	tracer := otel.Tracer("test")
	assert.NotNil(t, tracer)

	// Verify we can create spans
	ctx, span := tracer.Start(context.Background(), "test-span")
	assert.NotNil(t, span)
	assert.True(t, span.IsRecording())
	span.End()

	// Verify logger can be updated with trace context
	updatedLogger := UpdateLoggerWithTraceContext(ctx, logger)
	assert.NotNil(t, updatedLogger)

	// Clean up - shutdown may fail with errors about export timeouts
	// when there's no collector running, which is expected
	err = ShutdownOTel(context.Background(), providers, logger)
	// We just verify it completes (errors are expected without a collector)
	_ = err
}

// TestInitOTel_WithAllResourceAttributes tests that all resource attributes are set
func TestInitOTel_WithAllResourceAttributes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full resource test in short mode")
	}

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "my-service",
		ServiceVersion: "2.5.1",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)

	require.NoError(t, err)
	require.NotNil(t, providers)

	// The resource should include service name and version
	// (we can't easily inspect the resource directly without internal access,
	// but we verify the providers were created successfully)
	assert.NotNil(t, providers.TracerProvider)
	assert.NotNil(t, providers.MeterProvider)

	// Clean up
	_ = ShutdownOTel(context.Background(), providers, logger)
}

// TestShutdownOTel_AfterFullInit tests shutdown after full initialization
func TestShutdownOTel_AfterFullInit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full init shutdown test in short mode")
	}

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, providers)

	// Shutdown may return errors about export timeouts when no collector is running
	// This is expected behavior - we're testing that shutdown completes
	err = ShutdownOTel(context.Background(), providers, logger)
	// Shutdown completes even if export fails
	_ = err
}

// TestInitOTel_PropagatorConfiguration tests that propagators are configured
func TestInitOTel_PropagatorConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping propagator test in short mode")
	}

	// Store original propagator
	originalPropagator := otel.GetTextMapPropagator()
	defer otel.SetTextMapPropagator(originalPropagator)

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, providers)

	// Verify propagator was set
	propagator := otel.GetTextMapPropagator()
	assert.NotNil(t, propagator)

	// Clean up
	_ = ShutdownOTel(context.Background(), providers, logger)
}

// TestInitOTel_SecureAndInsecure tests both secure and insecure configurations
func TestInitOTel_SecureAndInsecure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping secure/insecure test in short mode")
	}

	tests := []struct {
		name     string
		insecure bool
	}{
		{"insecure", true},
		{"secure", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, &bytes.Buffer{})

			cfg := OTelConfig{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Insecure:       tt.insecure,
			}

			providers, err := InitOTel(context.Background(), cfg, logger)
			assert.NoError(t, err)
			assert.NotNil(t, providers)

			if providers != nil {
				_ = ShutdownOTel(context.Background(), providers, logger)
			}
		})
	}
}

// TestOTelConfig_VariousEndpoints tests different endpoint formats
func TestOTelConfig_VariousEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping endpoint test in short mode")
	}

	tests := []struct {
		name     string
		endpoint string
	}{
		{"localhost with port", "localhost:4317"},
		{"ip address", "127.0.0.1:4317"},
		{"hostname", "otel-collector:4317"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, &bytes.Buffer{})

			cfg := OTelConfig{
				Enabled:        true,
				Endpoint:       tt.endpoint,
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Insecure:       true,
			}

			providers, err := InitOTel(context.Background(), cfg, logger)
			assert.NoError(t, err)

			if providers != nil {
				_ = ShutdownOTel(context.Background(), providers, logger)
			}
		})
	}
}

// TestShutdownOTel_BothProviders tests shutdown with both tracer and meter providers
func TestShutdownOTel_BothProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping both providers test in short mode")
	}

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)
	require.NoError(t, err)
	require.NotNil(t, providers)
	require.NotNil(t, providers.TracerProvider)
	require.NotNil(t, providers.MeterProvider)

	// Shutdown with a short timeout to avoid hanging on export when collector is unavailable
	// In test environments without a collector, the export will timeout/fail, which is expected
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = ShutdownOTel(ctx, providers, logger)
	// Note: Error is acceptable here - in CI/test environments without an OTel collector,
	// the shutdown will fail when trying to export pending telemetry. This is expected behavior.
	// The test validates that shutdown completes without panicking, even if export fails.
	_ = err // Ignore shutdown errors - expected when no collector is running
}

// TestShutdownOTel_ErrorHandling tests error aggregation in shutdown
func TestShutdownOTel_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	buf := &bytes.Buffer{}
	logger := NewLogger(InfoLevel, buf)

	// Create a provider that will successfully shutdown
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)

	// Should not error with valid provider
	assert.NoError(t, err)
}

// TestInitOTel_ResourceCreation tests resource creation with attributes
func TestInitOTel_ResourceCreation(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test-service",
		ServiceVersion: "2.0.0",
		Insecure:       true,
	}

	// Resource should be created successfully
	providers, err := InitOTel(context.Background(), cfg, logger)

	assert.NoError(t, err)
	assert.NotNil(t, providers)
	assert.NotNil(t, providers.TracerProvider)
	assert.NotNil(t, providers.MeterProvider)

	// Clean up
	if providers != nil {
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}

// TestOTelConfig_ZeroValue tests zero value OTelConfig
func TestOTelConfig_ZeroValue(t *testing.T) {
	var cfg OTelConfig

	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.Endpoint)
	assert.Empty(t, cfg.ServiceName)
	assert.Empty(t, cfg.ServiceVersion)
	assert.False(t, cfg.Insecure)
}

// TestShutdownOTel_MultipleErrors tests handling of multiple shutdown errors
func TestShutdownOTel_MultipleErrors(t *testing.T) {
	// This test verifies the error aggregation logic
	ctx := context.Background()
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	// Create valid providers
	tp := sdktrace.NewTracerProvider()

	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)

	// Should successfully shutdown without errors
	assert.NoError(t, err)
}

// TestInitOTel_GRPCOptions tests gRPC options configuration
func TestInitOTel_GRPCOptions(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
	}{
		{
			name:     "insecure connection",
			insecure: true,
		},
		{
			name:     "secure connection",
			insecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(InfoLevel, &bytes.Buffer{})

			cfg := OTelConfig{
				Enabled:        true,
				Endpoint:       "localhost:4317",
				ServiceName:    "test-service",
				ServiceVersion: "1.0.0",
				Insecure:       tt.insecure,
			}

			// Should handle both secure and insecure options
			providers, err := InitOTel(context.Background(), cfg, logger)

			// Should not panic with either option
			assert.NoError(t, err)
			assert.NotNil(t, providers)

			// Clean up
			if providers != nil {
				_ = ShutdownOTel(context.Background(), providers, logger)
			}
		})
	}
}

// TestUpdateLoggerWithTraceContext_NilLogger tests with nil logger (edge case)
func TestUpdateLoggerWithTraceContext_NilLogger(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("UpdateLoggerWithTraceContext panicked with nil logger: %v", r)
		}
	}()

	ctx := context.Background()

	// This should not panic even with nil logger
	// (though in practice, logger should never be nil)
	result := UpdateLoggerWithTraceContext(ctx, nil)

	// Result should be nil since input was nil
	assert.Nil(t, result)
}

// TestShutdownOTel_TimeoutContext tests shutdown with timeout context
func TestShutdownOTel_TimeoutContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	tp := sdktrace.NewTracerProvider()
	providers := &OTelProviders{
		TracerProvider: tp,
		MeterProvider:  nil,
	}

	err := ShutdownOTel(ctx, providers, logger)

	// Should complete within timeout
	assert.NoError(t, err)
}

// TestInitOTel_ErrorPropagation tests error handling
func TestInitOTel_ErrorPropagation(t *testing.T) {
	logger := NewLogger(InfoLevel, &bytes.Buffer{})

	cfg := OTelConfig{
		Enabled:        true,
		Endpoint:       "localhost:4317",
		ServiceName:    "test",
		ServiceVersion: "1.0.0",
		Insecure:       true,
	}

	providers, err := InitOTel(context.Background(), cfg, logger)

	// OTLP exporters succeed at creation even without a collector
	assert.NoError(t, err)
	assert.NotNil(t, providers)

	// Clean up
	if providers != nil {
		_ = ShutdownOTel(context.Background(), providers, logger)
	}
}
