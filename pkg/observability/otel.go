package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// OTelConfig holds OpenTelemetry configuration
type OTelConfig struct {
	Enabled        bool
	Endpoint       string
	ServiceName    string
	ServiceVersion string
	Insecure       bool
}

// OTelProviders holds OpenTelemetry providers for shutdown
type OTelProviders struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
}

// InitOTel initializes OpenTelemetry providers
func InitOTel(ctx context.Context, cfg OTelConfig, logger *Logger) (*OTelProviders, error) {
	if !cfg.Enabled {
		logger.Info("OpenTelemetry is disabled")
		return nil, nil
	}

	logger.Infof("Initializing OpenTelemetry with endpoint: %s", cfg.Endpoint)

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup gRPC connection options
	grpcOpts := []grpc.DialOption{
		//nolint:staticcheck // SA1019: WithBlock deprecated but needed for OTEL collector connection
		grpc.WithBlock(),
	}
	if cfg.Insecure {
		grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Initialize tracer provider
	tracerProvider, err := initTracerProvider(ctx, cfg.Endpoint, res, grpcOpts, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Initialize meter provider
	meterProvider, err := initMeterProvider(ctx, cfg.Endpoint, res, grpcOpts, logger)
	if err != nil {
		// Shutdown tracer provider if meter provider fails
		if shutdownErr := tracerProvider.Shutdown(ctx); shutdownErr != nil {
			logger.WithError(shutdownErr).Error("Failed to shutdown tracer provider after meter provider error")
		}
		return nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	// Set global providers
	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	// Set global propagator for context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	logger.Info("OpenTelemetry initialized successfully")

	return &OTelProviders{
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	}, nil
}

// initTracerProvider initializes the OpenTelemetry tracer provider
func initTracerProvider(ctx context.Context, endpoint string, res *resource.Resource, grpcOpts []grpc.DialOption, logger *Logger) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithDialOption(grpcOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider with batch span processor
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	logger.Info("Tracer provider initialized")
	return tp, nil
}

// initMeterProvider initializes the OpenTelemetry meter provider
func initMeterProvider(ctx context.Context, endpoint string, res *resource.Resource, grpcOpts []grpc.DialOption, logger *Logger) (*metric.MeterProvider, error) {
	// Create OTLP metric exporter
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithDialOption(grpcOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create meter provider with periodic reader
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(10*time.Second),
		)),
	)

	logger.Info("Meter provider initialized")
	return mp, nil
}

// ShutdownOTel gracefully shuts down OpenTelemetry providers
func ShutdownOTel(ctx context.Context, providers *OTelProviders, logger *Logger) error {
	if providers == nil {
		return nil
	}

	logger.Info("Shutting down OpenTelemetry providers")

	var errs []error

	// Shutdown tracer provider
	if providers.TracerProvider != nil {
		if err := providers.TracerProvider.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Failed to shutdown tracer provider")
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		} else {
			logger.Info("Tracer provider shutdown complete")
		}
	}

	// Shutdown meter provider
	if providers.MeterProvider != nil {
		if err := providers.MeterProvider.Shutdown(ctx); err != nil {
			logger.WithError(err).Error("Failed to shutdown meter provider")
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		} else {
			logger.Info("Meter provider shutdown complete")
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("OpenTelemetry shutdown errors: %v", errs)
	}

	logger.Info("OpenTelemetry shutdown complete")
	return nil
}

// UpdateLoggerWithTraceContext adds trace context to logger
func UpdateLoggerWithTraceContext(ctx context.Context, logger *Logger) *Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	spanCtx := span.SpanContext()
	return logger.WithFields(map[string]interface{}{
		"trace_id": spanCtx.TraceID().String(),
		"span_id":  spanCtx.SpanID().String(),
	})
}
