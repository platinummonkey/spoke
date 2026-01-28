// Package config provides application configuration management from environment variables.
//
// # Overview
//
// This package loads and validates configuration from environment variables with
// sensible defaults for all settings.
//
// # Configuration Structure
//
// Server settings:
//
//	SPOKE_HOST="0.0.0.0"
//	SPOKE_PORT="8080"
//	SPOKE_HEALTH_PORT="8081"
//	SPOKE_READ_TIMEOUT="30s"
//	SPOKE_WRITE_TIMEOUT="30s"
//
// Storage settings:
//
//	SPOKE_STORAGE_TYPE="postgres"  # filesystem, postgres, hybrid, s3
//	SPOKE_FILESYSTEM_ROOT="/var/spoke/data"
//	SPOKE_POSTGRES_URL="postgres://localhost/spoke"
//	SPOKE_POSTGRES_MAX_CONNS="20"
//	SPOKE_S3_BUCKET="spoke-artifacts"
//	SPOKE_S3_REGION="us-east-1"
//
// Cache settings:
//
//	SPOKE_CACHE_ENABLED="true"
//	SPOKE_REDIS_URL="redis://localhost:6379"
//	SPOKE_REDIS_POOL_SIZE="10"
//
// Observability settings:
//
//	SPOKE_LOG_LEVEL="info"  # debug, info, warn, error
//	SPOKE_METRICS_ENABLED="true"
//	SPOKE_OTEL_ENABLED="true"
//	SPOKE_OTEL_ENDPOINT="otel-collector:4317"
//
// # Usage Example
//
// Load configuration:
//
//	cfg, err := config.Load()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
//	fmt.Printf("Storage: %s\n", cfg.Storage.Type)
//	fmt.Printf("Log level: %s\n", cfg.Observability.LogLevel)
//
// # Related Packages
//
//   - pkg/storage: Uses storage configuration
//   - pkg/observability: Uses observability configuration
package config
