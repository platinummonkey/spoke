package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l LogLevel) String() string {
	return []string{"DEBUG", "INFO", "WARN", "ERROR"}[l]
}

// Logger provides structured JSON logging
type Logger struct {
	level  LogLevel
	output io.Writer
	fields map[string]interface{}
}

// NewLogger creates a new structured logger
func NewLogger(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}
	return &Logger{
		level:  level,
		output: output,
		fields: make(map[string]interface{}),
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	Error       string                 `json:"error,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value
	return newLogger
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := &Logger{
		level:  l.level,
		output: l.output,
		fields: make(map[string]interface{}),
	}
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithError adds an error to the logger context
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return l.WithField("error", err.Error())
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(DebugLevel, message, nil)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.log(DebugLevel, fmt.Sprintf(format, args...), nil)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(InfoLevel, message, nil)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.log(InfoLevel, fmt.Sprintf(format, args...), nil)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(WarnLevel, message, nil)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.log(WarnLevel, fmt.Sprintf(format, args...), nil)
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.log(ErrorLevel, message, nil)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.log(ErrorLevel, fmt.Sprintf(format, args...), nil)
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   message,
		Fields:    make(map[string]interface{}),
	}

	// Add logger context fields
	for k, v := range l.fields {
		entry.Fields[k] = v
	}

	// Add additional fields
	for k, v := range fields {
		entry.Fields[k] = v
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// Fallback to simple output
		fmt.Fprintf(l.output, "[%s] %s: %s\n", entry.Timestamp.Format(time.RFC3339), level.String(), message)
		return
	}

	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

// contextKey is the type for context keys
type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// LoggerKey is the context key for the logger
	LoggerKey contextKey = "logger"
)

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves the user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// GetLogger retrieves the logger from context
func GetLogger(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerKey).(*Logger); ok {
		return logger
	}
	return NewLogger(InfoLevel, os.Stdout)
}

// FromContext creates a logger with request ID and user ID from context
func FromContext(ctx context.Context) *Logger {
	logger := GetLogger(ctx)

	if requestID := GetRequestID(ctx); requestID != "" {
		logger = logger.WithField("request_id", requestID)
	}

	if userID := GetUserID(ctx); userID != "" {
		logger = logger.WithField("user_id", userID)
	}

	return logger
}
