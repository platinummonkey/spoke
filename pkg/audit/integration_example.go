package audit

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// IntegrationConfig configures audit logging for the application
type IntegrationConfig struct {
	// Database connection for DB logger
	DB *sql.DB

	// File logging configuration
	FileLoggingEnabled bool
	FileLogPath        string
	FileLogRotate      bool
	FileLogMaxSize     int64
	FileLogMaxFiles    int

	// DB logging configuration
	DBLoggingEnabled bool

	// Middleware configuration
	LogAllRequests bool // If false, only log mutations and sensitive operations

	// Retention policy
	RetentionPolicy RetentionPolicy
}

// DefaultIntegrationConfig returns a default integration configuration
func DefaultIntegrationConfig(db *sql.DB) IntegrationConfig {
	return IntegrationConfig{
		DB:                 db,
		FileLoggingEnabled: true,
		FileLogPath:        "/var/log/spoke/audit",
		FileLogRotate:      true,
		FileLogMaxSize:     100 * 1024 * 1024, // 100MB
		FileLogMaxFiles:    10,
		DBLoggingEnabled:   true,
		LogAllRequests:     false,
		RetentionPolicy:    DefaultRetentionPolicy(),
	}
}

// SetupAuditLogging initializes audit logging for the application
func SetupAuditLogging(config IntegrationConfig) (*Middleware, *Handlers, error) {
	loggers := make([]Logger, 0)

	// Setup file logger if enabled
	if config.FileLoggingEnabled {
		fileConfig := FileLoggerConfig{
			BasePath: config.FileLogPath,
			Rotate:   config.FileLogRotate,
			MaxSize:  config.FileLogMaxSize,
			MaxFiles: config.FileLogMaxFiles,
		}

		fileLogger, err := NewFileLogger(fileConfig)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create file logger: %w", err)
		}

		loggers = append(loggers, fileLogger)
	}

	// Setup database logger if enabled
	var dbLogger *DBLogger
	if config.DBLoggingEnabled && config.DB != nil {
		var err error
		dbLogger, err = NewDBLogger(config.DB)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create database logger: %w", err)
		}

		loggers = append(loggers, dbLogger)
	}

	// Create multi-logger
	multiLogger := NewMultiLogger(loggers...)

	// Create middleware
	middleware := NewMiddleware(multiLogger, config.LogAllRequests)

	// Create store and handlers (only if DB logging is enabled)
	var handlers *Handlers
	if dbLogger != nil {
		store := NewDBStore(dbLogger)
		handlers = NewHandlers(store)
	}

	return middleware, handlers, nil
}

// Example: Integrating audit logging into the Spoke server
/*
func main() {
	// ... existing initialization code ...

	// Initialize database connection
	db, err := sql.Open("postgres", "postgres://...")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Setup audit logging
	config := audit.DefaultIntegrationConfig(db)
	auditMiddleware, auditHandlers, err := audit.SetupAuditLogging(config)
	if err != nil {
		log.Fatalf("Failed to setup audit logging: %v", err)
	}

	// Create router
	router := mux.NewRouter()

	// Register audit API handlers
	if auditHandlers != nil {
		auditHandlers.RegisterRoutes(router)
	}

	// Register other handlers...

	// Wrap router with audit middleware
	handler := auditMiddleware.Handler(router)

	// Start server
	http.ListenAndServe(":8080", handler)
}
*/

// Example: Logging audit events from application code
/*
func (h *ModuleHandlers) createModule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ... create module logic ...

	// Log successful module creation
	audit.LogSuccess(ctx, audit.EventTypeDataModuleCreate,
		"Module created successfully",
		map[string]interface{}{
			"module_name": module.Name,
			"version": module.Version,
		},
	)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(module)
}
*/

// Example: Logging authentication events
/*
func (h *AuthHandlers) login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := audit.FromContext(ctx)

	// ... authentication logic ...

	if err != nil {
		// Log failed authentication
		logger.LogAuthentication(ctx,
			audit.EventTypeAuthLoginFailed,
			nil,
			username,
			audit.EventStatusFailure,
			"Invalid credentials",
		)
		http.Error(w, "authentication failed", http.StatusUnauthorized)
		return
	}

	// Log successful authentication
	logger.LogAuthentication(ctx,
		audit.EventTypeAuthLogin,
		&user.ID,
		user.Username,
		audit.EventStatusSuccess,
		"Login successful",
	)

	// ... return token ...
}
*/

// Example: Logging authorization denials
/*
func (h *ModuleHandlers) deleteModule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// ... check permissions ...

	if !hasPermission {
		audit.LogDenied(ctx,
			audit.EventTypeAuthzAccessDenied,
			audit.ResourceTypeModule,
			moduleName,
			"User does not have delete permission",
		)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// ... delete module ...
}
*/

// Example: Logging data mutations with change tracking
/*
func (h *ModuleHandlers) updateModule(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := audit.FromContext(ctx)

	// Get original module
	original, err := h.store.GetModule(moduleName)
	if err != nil {
		return
	}

	// ... update module ...

	// Log the update with before/after values
	changes := &audit.ChangeDetails{
		Before: map[string]interface{}{
			"description": original.Description,
			"owner": original.Owner,
		},
		After: map[string]interface{}{
			"description": updated.Description,
			"owner": updated.Owner,
		},
	}

	logger.LogDataMutation(ctx,
		audit.EventTypeDataModuleUpdate,
		&userID,
		audit.ResourceTypeModule,
		moduleName,
		changes,
		"Module updated",
	)

	// ... return response ...
}
*/

// WrapRouterWithAudit is a convenience function to wrap a router with audit middleware
func WrapRouterWithAudit(router *mux.Router, middleware *Middleware) http.Handler {
	return middleware.Handler(router)
}

// AddAuditRoutes adds audit API routes to a router
func AddAuditRoutes(router *mux.Router, handlers *Handlers) {
	if handlers != nil {
		handlers.RegisterRoutes(router)
	}
}
