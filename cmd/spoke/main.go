package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/platinummonkey/spoke/pkg/api"
	"github.com/platinummonkey/spoke/pkg/config"
	"github.com/platinummonkey/spoke/pkg/dependencies"
	"github.com/platinummonkey/spoke/pkg/docs"
	"github.com/platinummonkey/spoke/pkg/observability"
	"github.com/platinummonkey/spoke/pkg/search"
	"github.com/platinummonkey/spoke/pkg/storage"
	"github.com/platinummonkey/spoke/pkg/storage/postgres"
)

func main() {
	// Load configuration from environment
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := observability.NewLogger(cfg.Observability.LogLevel, os.Stdout)
	logger.Info("Starting Spoke Schema Registry")
	logger.Infof("Storage type: %s", cfg.Storage.Type)

	// Initialize OpenTelemetry (if enabled)
	ctx := context.Background()
	otelProviders, err := observability.InitOTel(ctx, observability.OTelConfig{
		Enabled:        cfg.Observability.OTelEnabled,
		Endpoint:       cfg.Observability.OTelEndpoint,
		ServiceName:    cfg.Observability.OTelServiceName,
		ServiceVersion: cfg.Observability.OTelServiceVersion,
		Insecure:       cfg.Observability.OTelInsecure,
	}, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to initialize OpenTelemetry")
		// Don't fail - continue without OTel
	}

	// Initialize storage based on configuration
	var store api.Storage
	switch cfg.Storage.Type {
	case "filesystem":
		store, err = storage.NewFileSystemStorage(cfg.Storage.FilesystemRoot)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize filesystem storage")
			log.Fatalf("Failed to initialize filesystem storage: %v", err)
		}
		logger.Infof("Filesystem storage initialized: %s", cfg.Storage.FilesystemRoot)

	case "postgres", "hybrid":
		store, err = postgres.NewPostgresStorage(cfg.Storage)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize PostgreSQL storage")
			log.Fatalf("Failed to initialize PostgreSQL storage: %v", err)
		}
		logger.Infof("PostgreSQL storage initialized")

	default:
		log.Fatalf("Unknown storage type: %s", cfg.Storage.Type)
	}

	// Initialize health checker
	var healthChecker *observability.HealthChecker
	if pgStore, ok := store.(*postgres.PostgresStorage); ok {
		// Get database and Redis connections for health checks
		db := pgStore.GetDB()
		var redisClient *redis.Client
		if redisWrapper := pgStore.GetRedis(); redisWrapper != nil {
			redisClient = redisWrapper.GetClient()
		}
		healthChecker = observability.NewHealthChecker(db, redisClient)
		if redisClient != nil {
			logger.Info("Health checker initialized with database and Redis")
		} else {
			logger.Info("Health checker initialized with database (Redis not configured)")
		}
	} else {
		// Filesystem storage - health checker without dependencies
		healthChecker = observability.NewHealthChecker(nil, nil)
		logger.Info("Health checker initialized (no external dependencies)")
	}

	// Create API server
	// TODO: Initialize database connection for auth/compat/validation APIs
	server := api.NewServer(store, nil)

	// Register additional handlers
	docsHandlers := docs.NewDocsHandlers(store)
	server.RegisterRoutes(docsHandlers)
	logger.Info("Documentation routes registered")

	searchHandlers := search.NewSearchHandlers(store)
	server.RegisterRoutes(searchHandlers)
	logger.Info("Search routes registered")

	depHandlers := dependencies.NewDependencyHandlers(store)
	server.RegisterRoutes(depHandlers)
	logger.Info("Dependency routes registered")

	// Wrap with OpenTelemetry HTTP instrumentation
	var handler http.Handler = server
	if cfg.Observability.OTelEnabled {
		handler = otelhttp.NewHandler(handler, "spoke-api",
			otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
		)
		logger.Info("OpenTelemetry HTTP instrumentation enabled")
	}

	// Create main HTTP server with timeouts
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Create separate health/metrics server
	healthMux := http.NewServeMux()
	observability.RegisterHealthRoutes(healthMux, healthChecker)

	// Optionally expose Prometheus metrics
	if cfg.Observability.MetricsEnabled {
		// Note: This exposes existing Prometheus metrics
		// OTel metrics are exported via OTLP to collector
		healthMux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("# Prometheus metrics endpoint\n"))
			w.Write([]byte("# For OTel metrics, use the OpenTelemetry Collector\n"))
		}))
		logger.Info("Metrics endpoint enabled at /metrics")
	}

	healthServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.HealthPort),
		Handler:      healthMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Start health/metrics server in background
	go func() {
		logger.Infof("Starting health/metrics server on port %s", cfg.Server.HealthPort)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Health server failed")
		}
	}()

	// Setup graceful shutdown
	shutdownManager := observability.NewShutdownManager(logger, httpServer, cfg.Server.ShutdownTimeout)

	// Register cleanup functions
	shutdownManager.RegisterShutdownFunc(func(ctx context.Context) error {
		logger.Info("Shutting down health server")
		return healthServer.Shutdown(ctx)
	})

	if otelProviders != nil {
		shutdownManager.RegisterShutdownFunc(func(ctx context.Context) error {
			logger.Info("Shutting down OpenTelemetry")
			return observability.ShutdownOTel(ctx, otelProviders, logger)
		})
	}

	// Start main server in background
	go func() {
		logger.Infof("Starting Spoke API server on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("HTTP server failed")
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	logger.Info("Server started successfully, waiting for shutdown signal")
	if err := shutdownManager.WaitForShutdown(); err != nil {
		logger.WithError(err).Error("Graceful shutdown failed")
		os.Exit(1)
	}

	logger.Info("Server shutdown complete")
}
