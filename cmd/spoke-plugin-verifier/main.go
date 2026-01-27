package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/platinummonkey/spoke/pkg/plugins"
	"github.com/sirupsen/logrus"
)

// Config holds the verifier service configuration
type Config struct {
	DBConnectionString string
	PollInterval       time.Duration
	MaxConcurrent      int
	LogLevel           string
}

// Verifier Service continuously polls for pending plugin verifications and processes them
func main() {
	// Parse command-line flags
	config := parseFlags()

	// Setup logger
	logger := setupLogger(config.LogLevel)
	logger.Info("Starting Spoke Plugin Verifier Service")

	// Connect to database
	db, err := connectDatabase(config.DBConnectionString)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create verifier
	verifier := plugins.NewVerifier(db, logger)

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping verifier...")
		cancel()
	}()

	// Start verification worker pool
	logger.Infof("Starting %d verification workers with poll interval %v", config.MaxConcurrent, config.PollInterval)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	// Semaphore for limiting concurrent verifications
	sem := make(chan struct{}, config.MaxConcurrent)

	// Main verification loop
	ticker := time.NewTicker(config.PollInterval)
	defer ticker.Stop()

	// Process pending verifications on startup
	processPendingVerifications(workerCtx, verifier, logger, sem)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Shutting down verifier service")
			return

		case <-ticker.C:
			processPendingVerifications(workerCtx, verifier, logger, sem)
		}
	}
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.DBConnectionString, "db", getEnv("DATABASE_URL", "postgres://spoke:spoke@localhost:5432/spoke?sslmode=disable"), "Database connection string")
	flag.DurationVar(&config.PollInterval, "poll-interval", 30*time.Second, "Interval to poll for pending verifications")
	flag.IntVar(&config.MaxConcurrent, "max-concurrent", 3, "Maximum concurrent verifications")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	flag.Parse()

	return config
}

func setupLogger(logLevel string) *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	return logger
}

func connectDatabase(connectionString string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func processPendingVerifications(ctx context.Context, verifier *plugins.Verifier, logger *logrus.Logger, sem chan struct{}) {
	// List pending verifications
	pending, err := verifier.ListPendingVerifications(ctx, 20)
	if err != nil {
		logger.Errorf("Failed to list pending verifications: %v", err)
		return
	}

	if len(pending) == 0 {
		logger.Debug("No pending verifications found")
		return
	}

	logger.Infof("Found %d pending verifications", len(pending))

	for _, verification := range pending {
		// Skip if already processing
		if verification.Status == "in_progress" {
			continue
		}

		// Acquire semaphore slot
		select {
		case sem <- struct{}{}:
			// Got a slot, process verification
			go func(v *plugins.VerificationResult) {
				defer func() { <-sem }() // Release slot

				processVerification(ctx, verifier, v, logger)
			}(verification)

		case <-ctx.Done():
			return

		default:
			// No slots available, wait for next poll
			logger.Debug("All verification workers busy, waiting for next poll")
			return
		}
	}
}

func processVerification(ctx context.Context, verifier *plugins.Verifier, verification *plugins.VerificationResult, logger *logrus.Logger) {
	logger.Infof("Processing verification #%d for plugin %s v%s", verification.VerificationID, verification.PluginID, verification.Version)

	startTime := time.Now()

	// Get plugin download URL from database
	downloadURL, err := getPluginDownloadURL(ctx, verifier, verification.PluginID, verification.Version)
	if err != nil {
		logger.Errorf("Failed to get download URL for %s v%s: %v", verification.PluginID, verification.Version, err)
		rejectVerification(ctx, verifier, verification.VerificationID, "system", fmt.Sprintf("Failed to get download URL: %v", err))
		return
	}

	// Run verification
	result, err := verifier.RunVerification(ctx, verification.VerificationID, downloadURL)
	if err != nil {
		logger.Errorf("Verification #%d failed: %v", verification.VerificationID, err)
		return
	}

	duration := time.Since(startTime)
	logger.Infof("Verification #%d completed with status %s in %v", verification.VerificationID, result.Status, duration)
	logger.Infof("  - Manifest errors: %d", len(result.ManifestErrors))
	logger.Infof("  - Security issues: %d", len(result.SecurityIssues))

	// Log critical issues
	for _, issue := range result.SecurityIssues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			logger.Warnf("  [%s] %s: %s (in %s:%d)", issue.Severity, issue.Category, issue.Description, issue.File, issue.Line)
		}
	}
}

func getPluginDownloadURL(ctx context.Context, verifier *plugins.Verifier, pluginID, version string) (string, error) {
	// This is a simplified version - in production, this would query the marketplace API
	// or database to get the actual download URL
	return fmt.Sprintf("http://localhost:8080/api/v1/plugins/%s/versions/%s/download", pluginID, version), nil
}

func rejectVerification(ctx context.Context, verifier *plugins.Verifier, verificationID int64, rejectedBy, reason string) {
	if err := verifier.RejectVerification(ctx, verificationID, rejectedBy, reason); err != nil {
		logrus.Errorf("Failed to reject verification #%d: %v", verificationID, err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
