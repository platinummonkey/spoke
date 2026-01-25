package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/platinummonkey/spoke/pkg/observability"
	"github.com/platinummonkey/spoke/pkg/storage"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Storage configuration
	Storage storage.Config

	// Observability configuration
	Observability ObservabilityConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration

	// Health/metrics server (separate port for k8s probes)
	HealthPort string
}

// ObservabilityConfig holds observability settings
type ObservabilityConfig struct {
	// Logging
	LogLevel observability.LogLevel

	// Metrics
	MetricsEnabled bool

	// OpenTelemetry
	OTelEnabled        bool
	OTelEndpoint       string
	OTelServiceName    string
	OTelServiceVersion string
	OTelInsecure       bool // Use insecure gRPC connection
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Server:        loadServerConfig(),
		Storage:       loadStorageConfig(),
		Observability: loadObservabilityConfig(),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// loadServerConfig loads server configuration from environment
func loadServerConfig() ServerConfig {
	return ServerConfig{
		Host:            getEnv("SPOKE_HOST", "0.0.0.0"),
		Port:            getEnv("SPOKE_PORT", "8080"),
		ReadTimeout:     getEnvDuration("SPOKE_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getEnvDuration("SPOKE_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:     getEnvDuration("SPOKE_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getEnvDuration("SPOKE_SHUTDOWN_TIMEOUT", 30*time.Second),
		HealthPort:      getEnv("SPOKE_HEALTH_PORT", "9090"),
	}
}

// loadStorageConfig loads storage configuration from environment
func loadStorageConfig() storage.Config {
	cfg := storage.DefaultConfig()

	// Storage type
	if storageType := getEnv("SPOKE_STORAGE_TYPE", ""); storageType != "" {
		cfg.Type = storageType
	}

	// Filesystem config
	if fsRoot := getEnv("SPOKE_FILESYSTEM_ROOT", ""); fsRoot != "" {
		cfg.FilesystemRoot = fsRoot
	}

	// PostgreSQL config
	if pgURL := getEnv("SPOKE_POSTGRES_URL", ""); pgURL != "" {
		cfg.PostgresURL = pgURL
	}
	if replicaURLs := getEnv("SPOKE_POSTGRES_REPLICA_URLS", ""); replicaURLs != "" {
		cfg.PostgresReplicaURLs = replicaURLs
	}
	if maxConns := getEnvInt("SPOKE_POSTGRES_MAX_CONNS", 0); maxConns > 0 {
		cfg.PostgresMaxConns = maxConns
	}
	if minConns := getEnvInt("SPOKE_POSTGRES_MIN_CONNS", 0); minConns > 0 {
		cfg.PostgresMinConns = minConns
	}
	if timeout := getEnvDuration("SPOKE_POSTGRES_TIMEOUT", 0); timeout > 0 {
		cfg.PostgresTimeout = timeout
	}

	// S3 config
	if s3Endpoint := getEnv("SPOKE_S3_ENDPOINT", ""); s3Endpoint != "" {
		cfg.S3Endpoint = s3Endpoint
	}
	if s3Region := getEnv("SPOKE_S3_REGION", ""); s3Region != "" {
		cfg.S3Region = s3Region
	}
	if s3Bucket := getEnv("SPOKE_S3_BUCKET", ""); s3Bucket != "" {
		cfg.S3Bucket = s3Bucket
	}
	if s3AccessKey := getEnv("SPOKE_S3_ACCESS_KEY", ""); s3AccessKey != "" {
		cfg.S3AccessKey = s3AccessKey
	}
	if s3SecretKey := getEnv("SPOKE_S3_SECRET_KEY", ""); s3SecretKey != "" {
		cfg.S3SecretKey = s3SecretKey
	}
	if s3UsePathStyle := getEnv("SPOKE_S3_USE_PATH_STYLE", ""); s3UsePathStyle != "" {
		cfg.S3UsePathStyle = strings.ToLower(s3UsePathStyle) == "true"
	}
	if s3ForcePathStyle := getEnv("SPOKE_S3_FORCE_PATH_STYLE", ""); s3ForcePathStyle != "" {
		cfg.S3ForcePathStyle = strings.ToLower(s3ForcePathStyle) == "true"
	}

	// Redis config
	if redisURL := getEnv("SPOKE_REDIS_URL", ""); redisURL != "" {
		cfg.RedisURL = redisURL
	}
	if redisPassword := getEnv("SPOKE_REDIS_PASSWORD", ""); redisPassword != "" {
		cfg.RedisPassword = redisPassword
	}
	if redisDB := getEnvInt("SPOKE_REDIS_DB", -1); redisDB >= 0 {
		cfg.RedisDB = redisDB
	}
	if redisMaxRetries := getEnvInt("SPOKE_REDIS_MAX_RETRIES", 0); redisMaxRetries > 0 {
		cfg.RedisMaxRetries = redisMaxRetries
	}
	if redisPoolSize := getEnvInt("SPOKE_REDIS_POOL_SIZE", 0); redisPoolSize > 0 {
		cfg.RedisPoolSize = redisPoolSize
	}

	// Cache config
	if cacheEnabled := getEnv("SPOKE_CACHE_ENABLED", ""); cacheEnabled != "" {
		cfg.CacheEnabled = strings.ToLower(cacheEnabled) == "true"
	}
	if l1CacheSize := getEnvInt64("SPOKE_L1_CACHE_SIZE", 0); l1CacheSize > 0 {
		cfg.L1CacheSize = l1CacheSize
	}

	return cfg
}

// loadObservabilityConfig loads observability configuration from environment
func loadObservabilityConfig() ObservabilityConfig {
	cfg := ObservabilityConfig{
		LogLevel:           parseLogLevel(getEnv("SPOKE_LOG_LEVEL", "info")),
		MetricsEnabled:     getEnvBool("SPOKE_METRICS_ENABLED", true),
		OTelEnabled:        getEnvBool("SPOKE_OTEL_ENABLED", false),
		OTelEndpoint:       getEnv("SPOKE_OTEL_ENDPOINT", "localhost:4317"),
		OTelServiceName:    getEnv("SPOKE_OTEL_SERVICE_NAME", "spoke-registry"),
		OTelServiceVersion: getEnv("SPOKE_OTEL_SERVICE_VERSION", "1.0.0"),
		OTelInsecure:       getEnvBool("SPOKE_OTEL_INSECURE", true),
	}

	return cfg
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Validate server config
	if c.Server.Port == "" {
		return fmt.Errorf("server port is required")
	}
	if c.Server.HealthPort == "" {
		return fmt.Errorf("health port is required")
	}
	if c.Server.Port == c.Server.HealthPort {
		return fmt.Errorf("server port and health port must be different")
	}

	// Validate storage config based on type
	switch c.Storage.Type {
	case "filesystem":
		if c.Storage.FilesystemRoot == "" {
			return fmt.Errorf("filesystem root is required for filesystem storage")
		}
	case "postgres":
		if c.Storage.PostgresURL == "" {
			return fmt.Errorf("postgres URL is required for postgres storage")
		}
		if c.Storage.S3Endpoint == "" || c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3 configuration is required for postgres storage")
		}
	case "hybrid":
		if c.Storage.PostgresURL == "" {
			return fmt.Errorf("postgres URL is required for hybrid storage")
		}
		if c.Storage.S3Endpoint == "" || c.Storage.S3Bucket == "" {
			return fmt.Errorf("S3 configuration is required for hybrid storage")
		}
	default:
		return fmt.Errorf("invalid storage type: %s (must be filesystem, postgres, or hybrid)", c.Storage.Type)
	}

	// Validate OpenTelemetry config
	if c.Observability.OTelEnabled {
		if c.Observability.OTelEndpoint == "" {
			return fmt.Errorf("OpenTelemetry endpoint is required when OTel is enabled")
		}
		if c.Observability.OTelServiceName == "" {
			return fmt.Errorf("OpenTelemetry service name is required when OTel is enabled")
		}
	}

	return nil
}

// parseLogLevel parses a log level string
func parseLogLevel(level string) observability.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return observability.DebugLevel
	case "info":
		return observability.InfoLevel
	case "warn", "warning":
		return observability.WarnLevel
	case "error":
		return observability.ErrorLevel
	default:
		return observability.InfoLevel
	}
}

// getEnv returns an environment variable value or a default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns a boolean environment variable or a default
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}

// getEnvInt returns an integer environment variable or a default
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvInt64 returns an int64 environment variable or a default
func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getEnvDuration returns a duration environment variable or a default
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
