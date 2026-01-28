package config

import (
	"os"
	"testing"
	"time"

	"github.com/platinummonkey/spoke/pkg/observability"
)

// TestGetEnv tests the getEnv helper function
func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "returns env value when set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "returns default when env not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			}

			got := getEnv(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvBool tests the getEnvBool helper function
func TestGetEnvBool(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		want         bool
	}{
		{
			name:         "returns true for 'true'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			want:         true,
		},
		{
			name:         "returns true for '1'",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			want:         true,
		},
		{
			name:         "returns false for 'false'",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			want:         false,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_BOOL_NOT_SET",
			defaultValue: true,
			envValue:     "",
			want:         true,
		},
		{
			name:         "returns true for 'TRUE' (case insensitive)",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "TRUE",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvBool(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvInt tests the getEnvInt helper function
func TestGetEnvInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		want         int
	}{
		{
			name:         "returns parsed int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "42",
			want:         42,
		},
		{
			name:         "returns default for invalid int",
			key:          "TEST_INT",
			defaultValue: 10,
			envValue:     "invalid",
			want:         10,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT_NOT_SET",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvInt(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvInt() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvInt64 tests the getEnvInt64 helper function
func TestGetEnvInt64(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int64
		envValue     string
		want         int64
	}{
		{
			name:         "returns parsed int64",
			key:          "TEST_INT64",
			defaultValue: 10,
			envValue:     "9223372036854775807",
			want:         9223372036854775807,
		},
		{
			name:         "returns default for invalid int64",
			key:          "TEST_INT64",
			defaultValue: 10,
			envValue:     "invalid",
			want:         10,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_INT64_NOT_SET",
			defaultValue: 10,
			envValue:     "",
			want:         10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvInt64(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnvDuration tests the getEnvDuration helper function
func TestGetEnvDuration(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue time.Duration
		envValue     string
		want         time.Duration
	}{
		{
			name:         "returns parsed duration",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "30s",
			want:         30 * time.Second,
		},
		{
			name:         "returns default for invalid duration",
			key:          "TEST_DURATION",
			defaultValue: 10 * time.Second,
			envValue:     "invalid",
			want:         10 * time.Second,
		},
		{
			name:         "returns default when not set",
			key:          "TEST_DURATION_NOT_SET",
			defaultValue: 10 * time.Second,
			envValue:     "",
			want:         10 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
				defer os.Unsetenv(tt.key)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvDuration(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestParseLogLevel tests the parseLogLevel function
func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  observability.LogLevel
	}{
		{
			name:  "debug",
			level: "debug",
			want:  observability.DebugLevel,
		},
		{
			name:  "DEBUG uppercase",
			level: "DEBUG",
			want:  observability.DebugLevel,
		},
		{
			name:  "info",
			level: "info",
			want:  observability.InfoLevel,
		},
		{
			name:  "warn",
			level: "warn",
			want:  observability.WarnLevel,
		},
		{
			name:  "warning",
			level: "warning",
			want:  observability.WarnLevel,
		},
		{
			name:  "error",
			level: "error",
			want:  observability.ErrorLevel,
		},
		{
			name:  "invalid defaults to info",
			level: "invalid",
			want:  observability.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLogLevel(tt.level)
			if got != tt.want {
				t.Errorf("parseLogLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestLoadServerConfig tests the loadServerConfig function
func TestLoadServerConfig(t *testing.T) {
	// Save current env and restore after test
	originalEnv := map[string]string{
		"SPOKE_HOST":             os.Getenv("SPOKE_HOST"),
		"SPOKE_PORT":             os.Getenv("SPOKE_PORT"),
		"SPOKE_READ_TIMEOUT":     os.Getenv("SPOKE_READ_TIMEOUT"),
		"SPOKE_WRITE_TIMEOUT":    os.Getenv("SPOKE_WRITE_TIMEOUT"),
		"SPOKE_IDLE_TIMEOUT":     os.Getenv("SPOKE_IDLE_TIMEOUT"),
		"SPOKE_SHUTDOWN_TIMEOUT": os.Getenv("SPOKE_SHUTDOWN_TIMEOUT"),
		"SPOKE_HEALTH_PORT":      os.Getenv("SPOKE_HEALTH_PORT"),
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name string
		env  map[string]string
		want ServerConfig
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: ServerConfig{
				Host:            "0.0.0.0",
				Port:            "8080",
				ReadTimeout:     15 * time.Second,
				WriteTimeout:    15 * time.Second,
				IdleTimeout:     60 * time.Second,
				ShutdownTimeout: 30 * time.Second,
				HealthPort:      "9090",
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"SPOKE_HOST":             "localhost",
				"SPOKE_PORT":             "3000",
				"SPOKE_READ_TIMEOUT":     "30s",
				"SPOKE_WRITE_TIMEOUT":    "30s",
				"SPOKE_IDLE_TIMEOUT":     "120s",
				"SPOKE_SHUTDOWN_TIMEOUT": "60s",
				"SPOKE_HEALTH_PORT":      "9091",
			},
			want: ServerConfig{
				Host:            "localhost",
				Port:            "3000",
				ReadTimeout:     30 * time.Second,
				WriteTimeout:    30 * time.Second,
				IdleTimeout:     120 * time.Second,
				ShutdownTimeout: 60 * time.Second,
				HealthPort:      "9091",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for k := range originalEnv {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got := loadServerConfig()
			if got.Host != tt.want.Host {
				t.Errorf("Host = %v, want %v", got.Host, tt.want.Host)
			}
			if got.Port != tt.want.Port {
				t.Errorf("Port = %v, want %v", got.Port, tt.want.Port)
			}
			if got.ReadTimeout != tt.want.ReadTimeout {
				t.Errorf("ReadTimeout = %v, want %v", got.ReadTimeout, tt.want.ReadTimeout)
			}
			if got.WriteTimeout != tt.want.WriteTimeout {
				t.Errorf("WriteTimeout = %v, want %v", got.WriteTimeout, tt.want.WriteTimeout)
			}
			if got.IdleTimeout != tt.want.IdleTimeout {
				t.Errorf("IdleTimeout = %v, want %v", got.IdleTimeout, tt.want.IdleTimeout)
			}
			if got.ShutdownTimeout != tt.want.ShutdownTimeout {
				t.Errorf("ShutdownTimeout = %v, want %v", got.ShutdownTimeout, tt.want.ShutdownTimeout)
			}
			if got.HealthPort != tt.want.HealthPort {
				t.Errorf("HealthPort = %v, want %v", got.HealthPort, tt.want.HealthPort)
			}
		})
	}
}

// TestLoadStorageConfig tests the loadStorageConfig function
func TestLoadStorageConfig(t *testing.T) {
	// Save current env and restore after test
	envVars := []string{
		"SPOKE_STORAGE_TYPE",
		"SPOKE_FILESYSTEM_ROOT",
		"SPOKE_POSTGRES_URL",
		"SPOKE_POSTGRES_REPLICA_URLS",
		"SPOKE_POSTGRES_MAX_CONNS",
		"SPOKE_POSTGRES_MIN_CONNS",
		"SPOKE_POSTGRES_TIMEOUT",
		"SPOKE_S3_ENDPOINT",
		"SPOKE_S3_REGION",
		"SPOKE_S3_BUCKET",
		"SPOKE_S3_ACCESS_KEY",
		"SPOKE_S3_SECRET_KEY",
		"SPOKE_S3_USE_PATH_STYLE",
		"SPOKE_S3_FORCE_PATH_STYLE",
		"SPOKE_REDIS_URL",
		"SPOKE_REDIS_PASSWORD",
		"SPOKE_REDIS_DB",
		"SPOKE_REDIS_MAX_RETRIES",
		"SPOKE_REDIS_POOL_SIZE",
		"SPOKE_CACHE_ENABLED",
		"SPOKE_L1_CACHE_SIZE",
	}
	originalEnv := make(map[string]string)
	for _, k := range envVars {
		originalEnv[k] = os.Getenv(k)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	t.Run("loads default config", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		cfg := loadStorageConfig()
		if cfg.Type != "filesystem" {
			t.Errorf("Type = %v, want filesystem", cfg.Type)
		}
	})

	t.Run("loads postgres config from env", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_POSTGRES_URL", "postgres://localhost/db")
		os.Setenv("SPOKE_POSTGRES_REPLICA_URLS", "postgres://replica1,postgres://replica2")
		os.Setenv("SPOKE_POSTGRES_MAX_CONNS", "50")
		os.Setenv("SPOKE_POSTGRES_MIN_CONNS", "5")
		os.Setenv("SPOKE_POSTGRES_TIMEOUT", "20s")

		cfg := loadStorageConfig()
		if cfg.PostgresURL != "postgres://localhost/db" {
			t.Errorf("PostgresURL = %v, want postgres://localhost/db", cfg.PostgresURL)
		}
		if cfg.PostgresReplicaURLs != "postgres://replica1,postgres://replica2" {
			t.Errorf("PostgresReplicaURLs = %v, want postgres://replica1,postgres://replica2", cfg.PostgresReplicaURLs)
		}
		if cfg.PostgresMaxConns != 50 {
			t.Errorf("PostgresMaxConns = %v, want 50", cfg.PostgresMaxConns)
		}
		if cfg.PostgresMinConns != 5 {
			t.Errorf("PostgresMinConns = %v, want 5", cfg.PostgresMinConns)
		}
		if cfg.PostgresTimeout != 20*time.Second {
			t.Errorf("PostgresTimeout = %v, want 20s", cfg.PostgresTimeout)
		}
	})

	t.Run("loads s3 config from env", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_S3_ENDPOINT", "s3.amazonaws.com")
		os.Setenv("SPOKE_S3_REGION", "us-east-1")
		os.Setenv("SPOKE_S3_BUCKET", "my-bucket")
		os.Setenv("SPOKE_S3_ACCESS_KEY", "access")
		os.Setenv("SPOKE_S3_SECRET_KEY", "secret")
		os.Setenv("SPOKE_S3_USE_PATH_STYLE", "true")
		os.Setenv("SPOKE_S3_FORCE_PATH_STYLE", "true")

		cfg := loadStorageConfig()
		if cfg.S3Endpoint != "s3.amazonaws.com" {
			t.Errorf("S3Endpoint = %v, want s3.amazonaws.com", cfg.S3Endpoint)
		}
		if cfg.S3Region != "us-east-1" {
			t.Errorf("S3Region = %v, want us-east-1", cfg.S3Region)
		}
		if cfg.S3Bucket != "my-bucket" {
			t.Errorf("S3Bucket = %v, want my-bucket", cfg.S3Bucket)
		}
		if cfg.S3AccessKey != "access" {
			t.Errorf("S3AccessKey = %v, want access", cfg.S3AccessKey)
		}
		if cfg.S3SecretKey != "secret" {
			t.Errorf("S3SecretKey = %v, want secret", cfg.S3SecretKey)
		}
		if !cfg.S3UsePathStyle {
			t.Errorf("S3UsePathStyle = %v, want true", cfg.S3UsePathStyle)
		}
		if !cfg.S3ForcePathStyle {
			t.Errorf("S3ForcePathStyle = %v, want true", cfg.S3ForcePathStyle)
		}
	})

	t.Run("loads redis config from env", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_REDIS_URL", "redis://localhost:6379")
		os.Setenv("SPOKE_REDIS_PASSWORD", "password")
		os.Setenv("SPOKE_REDIS_DB", "1")
		os.Setenv("SPOKE_REDIS_MAX_RETRIES", "5")
		os.Setenv("SPOKE_REDIS_POOL_SIZE", "20")

		cfg := loadStorageConfig()
		if cfg.RedisURL != "redis://localhost:6379" {
			t.Errorf("RedisURL = %v, want redis://localhost:6379", cfg.RedisURL)
		}
		if cfg.RedisPassword != "password" {
			t.Errorf("RedisPassword = %v, want password", cfg.RedisPassword)
		}
		if cfg.RedisDB != 1 {
			t.Errorf("RedisDB = %v, want 1", cfg.RedisDB)
		}
		if cfg.RedisMaxRetries != 5 {
			t.Errorf("RedisMaxRetries = %v, want 5", cfg.RedisMaxRetries)
		}
		if cfg.RedisPoolSize != 20 {
			t.Errorf("RedisPoolSize = %v, want 20", cfg.RedisPoolSize)
		}
	})

	t.Run("loads cache config from env", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_CACHE_ENABLED", "true")
		os.Setenv("SPOKE_L1_CACHE_SIZE", "20971520")

		cfg := loadStorageConfig()
		if !cfg.CacheEnabled {
			t.Errorf("CacheEnabled = %v, want true", cfg.CacheEnabled)
		}
		if cfg.L1CacheSize != 20971520 {
			t.Errorf("L1CacheSize = %v, want 20971520", cfg.L1CacheSize)
		}
	})

	t.Run("ignores invalid postgres max conns", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_POSTGRES_MAX_CONNS", "0")

		cfg := loadStorageConfig()
		// Should keep default value
		if cfg.PostgresMaxConns != 20 {
			t.Errorf("PostgresMaxConns = %v, want 20 (default)", cfg.PostgresMaxConns)
		}
	})

	t.Run("ignores invalid redis db", func(t *testing.T) {
		// Clear all env vars
		for _, k := range envVars {
			os.Unsetenv(k)
		}

		os.Setenv("SPOKE_REDIS_DB", "-1")

		cfg := loadStorageConfig()
		// Should keep default value
		if cfg.RedisDB != 0 {
			t.Errorf("RedisDB = %v, want 0 (default)", cfg.RedisDB)
		}
	})
}

// TestLoadObservabilityConfig tests the loadObservabilityConfig function
func TestLoadObservabilityConfig(t *testing.T) {
	// Save current env and restore after test
	envVars := []string{
		"SPOKE_LOG_LEVEL",
		"SPOKE_METRICS_ENABLED",
		"SPOKE_OTEL_ENABLED",
		"SPOKE_OTEL_ENDPOINT",
		"SPOKE_OTEL_SERVICE_NAME",
		"SPOKE_OTEL_SERVICE_VERSION",
		"SPOKE_OTEL_INSECURE",
	}
	originalEnv := make(map[string]string)
	for _, k := range envVars {
		originalEnv[k] = os.Getenv(k)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name string
		env  map[string]string
		want ObservabilityConfig
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: ObservabilityConfig{
				LogLevel:           observability.InfoLevel,
				MetricsEnabled:     true,
				OTelEnabled:        false,
				OTelEndpoint:       "localhost:4317",
				OTelServiceName:    "spoke-registry",
				OTelServiceVersion: "1.0.0",
				OTelInsecure:       true,
			},
		},
		{
			name: "custom values",
			env: map[string]string{
				"SPOKE_LOG_LEVEL":            "debug",
				"SPOKE_METRICS_ENABLED":      "false",
				"SPOKE_OTEL_ENABLED":         "true",
				"SPOKE_OTEL_ENDPOINT":        "otel-collector:4317",
				"SPOKE_OTEL_SERVICE_NAME":    "my-service",
				"SPOKE_OTEL_SERVICE_VERSION": "2.0.0",
				"SPOKE_OTEL_INSECURE":        "false",
			},
			want: ObservabilityConfig{
				LogLevel:           observability.DebugLevel,
				MetricsEnabled:     false,
				OTelEnabled:        true,
				OTelEndpoint:       "otel-collector:4317",
				OTelServiceName:    "my-service",
				OTelServiceVersion: "2.0.0",
				OTelInsecure:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for _, k := range envVars {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			got := loadObservabilityConfig()
			if got.LogLevel != tt.want.LogLevel {
				t.Errorf("LogLevel = %v, want %v", got.LogLevel, tt.want.LogLevel)
			}
			if got.MetricsEnabled != tt.want.MetricsEnabled {
				t.Errorf("MetricsEnabled = %v, want %v", got.MetricsEnabled, tt.want.MetricsEnabled)
			}
			if got.OTelEnabled != tt.want.OTelEnabled {
				t.Errorf("OTelEnabled = %v, want %v", got.OTelEnabled, tt.want.OTelEnabled)
			}
			if got.OTelEndpoint != tt.want.OTelEndpoint {
				t.Errorf("OTelEndpoint = %v, want %v", got.OTelEndpoint, tt.want.OTelEndpoint)
			}
			if got.OTelServiceName != tt.want.OTelServiceName {
				t.Errorf("OTelServiceName = %v, want %v", got.OTelServiceName, tt.want.OTelServiceName)
			}
			if got.OTelServiceVersion != tt.want.OTelServiceVersion {
				t.Errorf("OTelServiceVersion = %v, want %v", got.OTelServiceVersion, tt.want.OTelServiceVersion)
			}
			if got.OTelInsecure != tt.want.OTelInsecure {
				t.Errorf("OTelInsecure = %v, want %v", got.OTelInsecure, tt.want.OTelInsecure)
			}
		})
	}
}

// TestConfigValidate tests the Config.Validate method
func TestConfigValidate(t *testing.T) {
	// Import storage to use Config type
	t.Run("missing server port", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "",
				HealthPort: "9090",
			},
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "server port is required" {
			t.Errorf("Validate() error = %v, want 'server port is required'", err.Error())
		}
	})

	t.Run("missing health port", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "",
			},
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "health port is required" {
			t.Errorf("Validate() error = %v, want 'health port is required'", err.Error())
		}
	})

	t.Run("same server and health port", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "8080",
			},
		}
		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "server port and health port must be different" {
			t.Errorf("Validate() error = %v, want 'server port and health port must be different'", err.Error())
		}
	})

	t.Run("otel enabled without endpoint", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
			Observability: ObservabilityConfig{
				OTelEnabled:     true,
				OTelEndpoint:    "",
				OTelServiceName: "test",
			},
		}
		// We need a valid storage config to pass server validation
		cfg.Storage.Type = "filesystem"
		cfg.Storage.FilesystemRoot = "/tmp/spoke"

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "OpenTelemetry endpoint is required when OTel is enabled" {
			t.Errorf("Validate() error = %v, want 'OpenTelemetry endpoint is required when OTel is enabled'", err.Error())
		}
	})

	t.Run("otel enabled without service name", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
			Observability: ObservabilityConfig{
				OTelEnabled:     true,
				OTelEndpoint:    "localhost:4317",
				OTelServiceName: "",
			},
		}
		// We need a valid storage config to pass server validation
		cfg.Storage.Type = "filesystem"
		cfg.Storage.FilesystemRoot = "/tmp/spoke"

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "OpenTelemetry service name is required when OTel is enabled" {
			t.Errorf("Validate() error = %v, want 'OpenTelemetry service name is required when OTel is enabled'", err.Error())
		}
	})

	t.Run("filesystem storage without root", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "filesystem"
		cfg.Storage.FilesystemRoot = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "filesystem root is required for filesystem storage" {
			t.Errorf("Validate() error = %v, want 'filesystem root is required for filesystem storage'", err.Error())
		}
	})

	t.Run("postgres storage without postgres url", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "postgres"
		cfg.Storage.PostgresURL = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "postgres URL is required for postgres storage" {
			t.Errorf("Validate() error = %v, want 'postgres URL is required for postgres storage'", err.Error())
		}
	})

	t.Run("postgres storage without s3 config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "postgres"
		cfg.Storage.PostgresURL = "postgres://localhost/db"
		cfg.Storage.S3Endpoint = ""
		cfg.Storage.S3Bucket = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "S3 configuration is required for postgres storage" {
			t.Errorf("Validate() error = %v, want 'S3 configuration is required for postgres storage'", err.Error())
		}
	})

	t.Run("hybrid storage without postgres url", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "hybrid"
		cfg.Storage.PostgresURL = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "postgres URL is required for hybrid storage" {
			t.Errorf("Validate() error = %v, want 'postgres URL is required for hybrid storage'", err.Error())
		}
	})

	t.Run("hybrid storage without s3 config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "hybrid"
		cfg.Storage.PostgresURL = "postgres://localhost/db"
		cfg.Storage.S3Endpoint = ""
		cfg.Storage.S3Bucket = ""

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		if err != nil && err.Error() != "S3 configuration is required for hybrid storage" {
			t.Errorf("Validate() error = %v, want 'S3 configuration is required for hybrid storage'", err.Error())
		}
	})

	t.Run("invalid storage type", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "invalid"

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() expected error, got nil")
		}
		expectedErr := "invalid storage type: invalid (must be filesystem, postgres, or hybrid)"
		if err != nil && err.Error() != expectedErr {
			t.Errorf("Validate() error = %v, want %v", err.Error(), expectedErr)
		}
	})

	t.Run("valid filesystem config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "filesystem"
		cfg.Storage.FilesystemRoot = "/tmp/spoke"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error = %v", err)
		}
	})

	t.Run("valid postgres config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "postgres"
		cfg.Storage.PostgresURL = "postgres://localhost/db"
		cfg.Storage.S3Endpoint = "s3.amazonaws.com"
		cfg.Storage.S3Bucket = "my-bucket"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error = %v", err)
		}
	})

	t.Run("valid hybrid config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
		}
		cfg.Storage.Type = "hybrid"
		cfg.Storage.PostgresURL = "postgres://localhost/db"
		cfg.Storage.S3Endpoint = "s3.amazonaws.com"
		cfg.Storage.S3Bucket = "my-bucket"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error = %v", err)
		}
	})

	t.Run("valid otel config", func(t *testing.T) {
		cfg := Config{
			Server: ServerConfig{
				Port:       "8080",
				HealthPort: "9090",
			},
			Observability: ObservabilityConfig{
				OTelEnabled:     true,
				OTelEndpoint:    "localhost:4317",
				OTelServiceName: "test-service",
			},
		}
		cfg.Storage.Type = "filesystem"
		cfg.Storage.FilesystemRoot = "/tmp/spoke"

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error = %v", err)
		}
	})
}

// TestLoadConfig tests the LoadConfig function
func TestLoadConfig(t *testing.T) {
	// Save current env and restore after test
	envVars := []string{
		"SPOKE_PORT",
		"SPOKE_HEALTH_PORT",
		"SPOKE_STORAGE_TYPE",
		"SPOKE_FILESYSTEM_ROOT",
	}
	originalEnv := make(map[string]string)
	for _, k := range envVars {
		originalEnv[k] = os.Getenv(k)
	}
	defer func() {
		for k, v := range originalEnv {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	}()

	tests := []struct {
		name    string
		env     map[string]string
		wantErr bool
	}{
		{
			name: "valid config",
			env: map[string]string{
				"SPOKE_PORT":          "8080",
				"SPOKE_HEALTH_PORT":   "9090",
				"SPOKE_STORAGE_TYPE":  "filesystem",
				"SPOKE_FILESYSTEM_ROOT": "/tmp/spoke",
			},
			wantErr: false,
		},
		{
			name: "invalid config - same ports",
			env: map[string]string{
				"SPOKE_PORT":        "8080",
				"SPOKE_HEALTH_PORT": "8080",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars
			for _, k := range envVars {
				os.Unsetenv(k)
			}

			// Set test env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cfg, err := LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg == nil {
				t.Error("LoadConfig() returned nil config without error")
			}
		})
	}
}
