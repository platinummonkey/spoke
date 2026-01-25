package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileLogger implements audit logging to files
type FileLogger struct {
	basePath string
	file     *os.File
	mu       sync.Mutex
	encoder  *json.Encoder
	rotate   bool
	maxSize  int64 // Max file size in bytes before rotation
	maxFiles int   // Max number of files to keep
}

// FileLoggerConfig configures the file logger
type FileLoggerConfig struct {
	BasePath string // Base directory for audit logs
	Rotate   bool   // Enable log rotation
	MaxSize  int64  // Max file size in bytes (default: 100MB)
	MaxFiles int    // Max number of files to keep (default: 10)
}

// DefaultFileLoggerConfig returns default configuration
func DefaultFileLoggerConfig() FileLoggerConfig {
	return FileLoggerConfig{
		BasePath: "/var/log/spoke/audit",
		Rotate:   true,
		MaxSize:  100 * 1024 * 1024, // 100MB
		MaxFiles: 10,
	}
}

// NewFileLogger creates a new file-based audit logger
func NewFileLogger(config FileLoggerConfig) (*FileLogger, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(config.BasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}

	logger := &FileLogger{
		basePath: config.BasePath,
		rotate:   config.Rotate,
		maxSize:  config.MaxSize,
		maxFiles: config.MaxFiles,
	}

	if logger.maxSize == 0 {
		logger.maxSize = 100 * 1024 * 1024 // 100MB default
	}
	if logger.maxFiles == 0 {
		logger.maxFiles = 10
	}

	// Open the current log file
	if err := logger.openLogFile(); err != nil {
		return nil, err
	}

	return logger, nil
}

// openLogFile opens or creates the current log file
func (l *FileLogger) openLogFile() error {
	filename := filepath.Join(l.basePath, "audit.log")

	// Check if we need to rotate
	if l.rotate {
		if info, err := os.Stat(filename); err == nil && info.Size() >= l.maxSize {
			if err := l.rotateFile(); err != nil {
				return fmt.Errorf("failed to rotate log file: %w", err)
			}
		}
	}

	// Open file in append mode
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file: %w", err)
	}

	l.file = file
	l.encoder = json.NewEncoder(file)

	return nil
}

// rotateFile rotates the log file
func (l *FileLogger) rotateFile() error {
	currentFile := filepath.Join(l.basePath, "audit.log")

	// Close current file if open
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	// Generate timestamp for rotated file
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	rotatedFile := filepath.Join(l.basePath, fmt.Sprintf("audit-%s.log", timestamp))

	// Rename current file to rotated name
	if err := os.Rename(currentFile, rotatedFile); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Clean up old files if needed
	if err := l.cleanupOldFiles(); err != nil {
		// Log but don't fail on cleanup errors
		fmt.Fprintf(os.Stderr, "failed to cleanup old audit logs: %v\n", err)
	}

	return nil
}

// cleanupOldFiles removes old log files beyond the retention limit
func (l *FileLogger) cleanupOldFiles() error {
	pattern := filepath.Join(l.basePath, "audit-*.log")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	// If we have more files than maxFiles, delete the oldest ones
	if len(files) > l.maxFiles {
		// Sort files by modification time (oldest first)
		// Note: This is a simple implementation; in production you'd want to
		// parse the timestamp from the filename or use file info
		filesToDelete := files[:len(files)-l.maxFiles]
		for _, file := range filesToDelete {
			if err := os.Remove(file); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove old audit log %s: %v\n", file, err)
			}
		}
	}

	return nil
}

// Log logs an audit event to the file
func (l *FileLogger) Log(ctx context.Context, event *AuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if we need to rotate
	if l.rotate && l.file != nil {
		if info, err := l.file.Stat(); err == nil && info.Size() >= l.maxSize {
			if err := l.openLogFile(); err != nil {
				return fmt.Errorf("failed to rotate log file: %w", err)
			}
		}
	}

	// Write event as JSON
	if err := l.encoder.Encode(event); err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// LogAuthentication logs an authentication event
func (l *FileLogger) LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.Username = username
	event.Message = message
	event.ResourceType = ResourceTypeUser

	return l.Log(ctx, event)
}

// LogAuthorization logs an authorization event
func (l *FileLogger) LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return l.Log(ctx, event)
}

// LogDataMutation logs a data mutation event
func (l *FileLogger) LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return l.Log(ctx, event)
}

// LogConfiguration logs a configuration change event
func (l *FileLogger) LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = ResourceTypeConfig
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return l.Log(ctx, event)
}

// LogAdminAction logs an admin action event
func (l *FileLogger) LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = adminUserID
	event.Message = message
	if targetUserID != nil {
		event.Metadata["target_user_id"] = *targetUserID
	}

	return l.Log(ctx, event)
}

// LogAccess logs a resource access event
func (l *FileLogger) LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return l.Log(ctx, event)
}

// LogHTTPRequest logs an HTTP request
func (l *FileLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	// Determine event type based on method and status
	eventType := EventTypeAccessModuleRead
	status := EventStatusSuccess

	if statusCode >= 400 {
		status = EventStatusFailure
	}
	if statusCode == 403 {
		status = EventStatusDenied
	}

	event := buildBaseEvent(ctx, r, eventType, status)
	event.StatusCode = statusCode
	event.Metadata["duration_ms"] = duration.Milliseconds()

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(ctx, event)
}

// Close closes the file logger
func (l *FileLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		err := l.file.Close()
		l.file = nil
		return err
	}

	return nil
}

// ReadLogs reads audit logs from the file
func (l *FileLogger) ReadLogs(count int) ([]*AuditEvent, error) {
	filename := filepath.Join(l.basePath, "audit.log")

	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log: %w", err)
	}
	defer file.Close()

	var events []*AuditEvent
	decoder := json.NewDecoder(file)

	for {
		var event AuditEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to decode audit log entry: %w", err)
		}
		events = append(events, &event)

		if count > 0 && len(events) >= count {
			break
		}
	}

	return events, nil
}
