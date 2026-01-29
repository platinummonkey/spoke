package observability

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
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

// toSlogLevel converts LogLevel to slog.Level
func (l LogLevel) toSlogLevel() slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Logger provides structured JSON logging using stdlib slog
type Logger struct {
	logger *slog.Logger
	level  LogLevel
}

// NewLogger creates a new structured logger using slog
func NewLogger(level LogLevel, output io.Writer) *Logger {
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{
		Level: level.toSlogLevel(),
	}
	handler := slog.NewJSONHandler(output, opts)

	return &Logger{
		logger: slog.New(handler),
		level:  level,
	}
}

// WithField adds a field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger: l.logger.With(key, value),
		level:  l.level,
	}
}

// WithFields adds multiple fields to the logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return &Logger{
		logger: l.logger.With(args...),
		level:  l.level,
	}
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
	l.logger.Debug(message)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.logger.Info(message)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.logger.Warn(message)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *Logger) Error(message string) {
	l.logger.Error(message)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
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
