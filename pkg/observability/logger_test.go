package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	t.Run("debug not logged at info level", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message")
		if buf.Len() > 0 {
			t.Error("Debug message should not be logged at Info level")
		}
	})

	t.Run("info logged at info level", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message")
		if buf.Len() == 0 {
			t.Error("Info message should be logged at Info level")
		}

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Level != "INFO" {
			t.Errorf("Expected level INFO, got %s", entry.Level)
		}
		if entry.Message != "info message" {
			t.Errorf("Expected message 'info message', got %s", entry.Message)
		}
	})

	t.Run("warn logged at info level", func(t *testing.T) {
		buf.Reset()
		logger.Warn("warn message")
		if buf.Len() == 0 {
			t.Error("Warn message should be logged at Info level")
		}
	})

	t.Run("error logged at info level", func(t *testing.T) {
		buf.Reset()
		logger.Error("error message")
		if buf.Len() == 0 {
			t.Error("Error message should be logged at Info level")
		}
	})
}

func TestLogger_WithField(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.WithField("key", "value").Info("message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["key"] != "value" {
		t.Errorf("Expected field 'key' to be 'value', got %v", entry.Fields["key"])
	}
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	}
	logger.WithFields(fields).Info("message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["key1"] != "value1" {
		t.Errorf("Expected field 'key1' to be 'value1', got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != float64(42) {
		t.Errorf("Expected field 'key2' to be 42, got %v", entry.Fields["key2"])
	}
}

func TestLogger_WithError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	err := strings.NewReader("test error").UnreadByte()
	logger.WithError(err).Error("something went wrong")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if _, exists := entry.Fields["error"]; !exists {
		t.Error("Expected error field to exist")
	}
}

func TestLogger_Formatters(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	t.Run("Debugf", func(t *testing.T) {
		buf.Reset()
		debugLogger := NewLogger(DebugLevel, &buf)
		debugLogger.Debugf("test %s %d", "string", 42)

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Message != "test string 42" {
			t.Errorf("Expected formatted message, got %s", entry.Message)
		}
	})

	t.Run("Infof", func(t *testing.T) {
		buf.Reset()
		logger.Infof("test %d", 123)

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Message != "test 123" {
			t.Errorf("Expected formatted message, got %s", entry.Message)
		}
	})

	t.Run("Warnf", func(t *testing.T) {
		buf.Reset()
		logger.Warnf("warning %s", "test")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Message != "warning test" {
			t.Errorf("Expected formatted message, got %s", entry.Message)
		}
	})

	t.Run("Errorf", func(t *testing.T) {
		buf.Reset()
		logger.Errorf("error %v", "test")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Message != "error test" {
			t.Errorf("Expected formatted message, got %s", entry.Message)
		}
	})
}

func TestContextHelpers(t *testing.T) {
	t.Run("RequestID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithRequestID(ctx, "req-123")

		requestID := GetRequestID(ctx)
		if requestID != "req-123" {
			t.Errorf("Expected request ID 'req-123', got %s", requestID)
		}
	})

	t.Run("UserID", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithUserID(ctx, "user-456")

		userID := GetUserID(ctx)
		if userID != "user-456" {
			t.Errorf("Expected user ID 'user-456', got %s", userID)
		}
	})

	t.Run("Logger", func(t *testing.T) {
		ctx := context.Background()
		logger := NewLogger(InfoLevel, nil)
		ctx = WithLogger(ctx, logger)

		retrievedLogger := GetLogger(ctx)
		if retrievedLogger == nil {
			t.Error("Expected to retrieve logger from context")
		}
	})

	t.Run("FromContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(InfoLevel, &buf)

		ctx := context.Background()
		ctx = WithLogger(ctx, logger)
		ctx = WithRequestID(ctx, "req-123")
		ctx = WithUserID(ctx, "user-456")

		contextLogger := FromContext(ctx)
		contextLogger.Info("test message")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Fields["request_id"] != "req-123" {
			t.Errorf("Expected request_id 'req-123', got %v", entry.Fields["request_id"])
		}
		if entry.Fields["user_id"] != "user-456" {
			t.Errorf("Expected user_id 'user-456', got %v", entry.Fields["user_id"])
		}
	})
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level LogLevel
		want  string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("LogLevel.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLogger_NilOutput(t *testing.T) {
	logger := NewLogger(InfoLevel, nil)
	if logger == nil {
		t.Error("Expected logger to be created with nil output")
	}
	if logger.output == nil {
		t.Error("Expected output to default to os.Stdout")
	}
}

func TestLogger_WithError_Nil(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Test with nil error - should return the same logger
	newLogger := logger.WithError(nil)
	if newLogger != logger {
		t.Error("Expected WithError(nil) to return the same logger")
	}
}

func TestLogger_WithFields_Chaining(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Test chaining multiple WithField calls
	logger.WithField("key1", "value1").
		WithField("key2", "value2").
		Info("chained message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["key1"] != "value1" {
		t.Errorf("Expected field 'key1' to be 'value1', got %v", entry.Fields["key1"])
	}
	if entry.Fields["key2"] != "value2" {
		t.Errorf("Expected field 'key2' to be 'value2', got %v", entry.Fields["key2"])
	}
}

func TestLogger_WithFields_OverwriteExisting(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Add initial field
	logger = logger.WithField("key", "value1")
	// Overwrite with WithFields
	logger.WithFields(map[string]interface{}{
		"key": "value2",
	}).Info("message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["key"] != "value2" {
		t.Errorf("Expected field 'key' to be overwritten to 'value2', got %v", entry.Fields["key"])
	}
}

func TestLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel LogLevel
		logFunc     func(*Logger)
		shouldLog   bool
		expectedLvl string
	}{
		{
			name:        "Debug at Debug level",
			loggerLevel: DebugLevel,
			logFunc:     func(l *Logger) { l.Debug("test") },
			shouldLog:   true,
			expectedLvl: "DEBUG",
		},
		{
			name:        "Debug at Info level",
			loggerLevel: InfoLevel,
			logFunc:     func(l *Logger) { l.Debug("test") },
			shouldLog:   false,
		},
		{
			name:        "Info at Debug level",
			loggerLevel: DebugLevel,
			logFunc:     func(l *Logger) { l.Info("test") },
			shouldLog:   true,
			expectedLvl: "INFO",
		},
		{
			name:        "Warn at Debug level",
			loggerLevel: DebugLevel,
			logFunc:     func(l *Logger) { l.Warn("test") },
			shouldLog:   true,
			expectedLvl: "WARN",
		},
		{
			name:        "Error at Debug level",
			loggerLevel: DebugLevel,
			logFunc:     func(l *Logger) { l.Error("test") },
			shouldLog:   true,
			expectedLvl: "ERROR",
		},
		{
			name:        "Info at Warn level",
			loggerLevel: WarnLevel,
			logFunc:     func(l *Logger) { l.Info("test") },
			shouldLog:   false,
		},
		{
			name:        "Warn at Error level",
			loggerLevel: ErrorLevel,
			logFunc:     func(l *Logger) { l.Warn("test") },
			shouldLog:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.loggerLevel, &buf)
			tt.logFunc(logger)

			if tt.shouldLog {
				if buf.Len() == 0 {
					t.Error("Expected message to be logged")
				}
				var entry LogEntry
				if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
					t.Fatalf("Failed to unmarshal log entry: %v", err)
				}
				if entry.Level != tt.expectedLvl {
					t.Errorf("Expected level %s, got %s", tt.expectedLvl, entry.Level)
				}
			} else {
				if buf.Len() > 0 {
					t.Error("Expected no log output")
				}
			}
		})
	}
}

func TestLogger_JSONStructure(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.Info("test message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	// Verify required fields
	if entry.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
	if entry.Level == "" {
		t.Error("Expected level to be set")
	}
	if entry.Message == "" {
		t.Error("Expected message to be set")
	}
}

func TestLogger_WithMultipleFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.WithFields(map[string]interface{}{
		"string":  "value",
		"int":     42,
		"float":   3.14,
		"bool":    true,
		"nil":     nil,
		"array":   []int{1, 2, 3},
		"map":     map[string]string{"nested": "value"},
	}).Info("complex fields")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["string"] != "value" {
		t.Errorf("Expected string field, got %v", entry.Fields["string"])
	}
	if entry.Fields["int"] != float64(42) {
		t.Errorf("Expected int field, got %v", entry.Fields["int"])
	}
	if entry.Fields["bool"] != true {
		t.Errorf("Expected bool field, got %v", entry.Fields["bool"])
	}
}

func TestContextHelpers_EmptyCases(t *testing.T) {
	t.Run("GetRequestID empty context", func(t *testing.T) {
		ctx := context.Background()
		requestID := GetRequestID(ctx)
		if requestID != "" {
			t.Errorf("Expected empty request ID, got %s", requestID)
		}
	})

	t.Run("GetUserID empty context", func(t *testing.T) {
		ctx := context.Background()
		userID := GetUserID(ctx)
		if userID != "" {
			t.Errorf("Expected empty user ID, got %s", userID)
		}
	})

	t.Run("GetLogger empty context", func(t *testing.T) {
		ctx := context.Background()
		logger := GetLogger(ctx)
		if logger == nil {
			t.Error("Expected default logger to be returned")
		}
	})

	t.Run("GetRequestID with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), RequestIDKey, 123)
		requestID := GetRequestID(ctx)
		if requestID != "" {
			t.Errorf("Expected empty request ID for wrong type, got %s", requestID)
		}
	})

	t.Run("GetUserID with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserIDKey, 456)
		userID := GetUserID(ctx)
		if userID != "" {
			t.Errorf("Expected empty user ID for wrong type, got %s", userID)
		}
	})

	t.Run("GetLogger with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), LoggerKey, "not a logger")
		logger := GetLogger(ctx)
		if logger == nil {
			t.Error("Expected default logger to be returned for wrong type")
		}
	})
}

func TestFromContext_PartialContext(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := NewLogger(InfoLevel, &buf)

	t.Run("only request ID", func(t *testing.T) {
		buf.Reset()
		ctx := context.Background()
		ctx = WithLogger(ctx, baseLogger)
		ctx = WithRequestID(ctx, "req-only")

		logger := FromContext(ctx)
		logger.Info("test")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Fields["request_id"] != "req-only" {
			t.Errorf("Expected request_id, got %v", entry.Fields["request_id"])
		}
		if _, exists := entry.Fields["user_id"]; exists {
			t.Error("Did not expect user_id field")
		}
	})

	t.Run("only user ID", func(t *testing.T) {
		buf.Reset()
		ctx := context.Background()
		ctx = WithLogger(ctx, baseLogger)
		ctx = WithUserID(ctx, "user-only")

		logger := FromContext(ctx)
		logger.Info("test")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Fields["user_id"] != "user-only" {
			t.Errorf("Expected user_id, got %v", entry.Fields["user_id"])
		}
		if _, exists := entry.Fields["request_id"]; exists {
			t.Error("Did not expect request_id field")
		}
	})

	t.Run("no IDs", func(t *testing.T) {
		buf.Reset()
		ctx := context.Background()
		ctx = WithLogger(ctx, baseLogger)

		logger := FromContext(ctx)
		logger.Info("test")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if _, exists := entry.Fields["request_id"]; exists {
			t.Error("Did not expect request_id field")
		}
		if _, exists := entry.Fields["user_id"]; exists {
			t.Error("Did not expect user_id field")
		}
	})
}

func TestLogger_FieldIsolation(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := NewLogger(InfoLevel, &buf)

	// Create child logger with field
	childLogger := baseLogger.WithField("child", "value")

	// Log with base logger - should not have child field
	buf.Reset()
	baseLogger.Info("base message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if _, exists := entry.Fields["child"]; exists {
		t.Error("Base logger should not have child's field")
	}

	// Log with child logger - should have child field
	buf.Reset()
	childLogger.Info("child message")

	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["child"] != "value" {
		t.Errorf("Child logger should have child field, got %v", entry.Fields["child"])
	}
}

func TestLogger_ConcurrentLogging(t *testing.T) {
	// Use a thread-safe writer for concurrent test
	// We're testing that logging doesn't panic, not validating output format
	logger := NewLogger(InfoLevel, io.Discard)

	// Test that concurrent logging doesn't panic
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			logger.WithField("goroutine", id).Info("concurrent log")
		}(i)
	}

	wg.Wait()
	// If we reach here without panicking, the test passes
}

func TestLogger_EmptyFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Log with empty fields map
	logger.WithFields(map[string]interface{}{}).Info("empty fields")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Message != "empty fields" {
		t.Errorf("Expected message 'empty fields', got %s", entry.Message)
	}
}

func TestLogger_FormatterEdgeCases(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	t.Run("Infof with no args", func(t *testing.T) {
		buf.Reset()
		logger.Infof("no args")

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if entry.Message != "no args" {
			t.Errorf("Expected message 'no args', got %s", entry.Message)
		}
	})

	t.Run("Errorf with multiple args", func(t *testing.T) {
		buf.Reset()
		logger.Errorf("error: %s, code: %d, details: %v", "test error", 500, map[string]string{"key": "value"})

		var entry LogEntry
		if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
			t.Fatalf("Failed to unmarshal log entry: %v", err)
		}

		if !strings.Contains(entry.Message, "test error") {
			t.Errorf("Expected message to contain 'test error', got %s", entry.Message)
		}
		if !strings.Contains(entry.Message, "500") {
			t.Errorf("Expected message to contain '500', got %s", entry.Message)
		}
	})
}

func TestLogger_TimestampIsUTC(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	logger.Info("test")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Timestamp.Location() != nil && entry.Timestamp.Location().String() != "UTC" {
		t.Errorf("Expected timestamp to be in UTC, got %s", entry.Timestamp.Location())
	}
}

// mockFailingWriter is a writer that always returns an error
type mockFailingWriter struct{}

func (m *mockFailingWriter) Write(p []byte) (n int, err error) {
	return 0, strings.NewReader("mock error").UnreadByte()
}

// unmarshalableValue is a type that cannot be marshaled to JSON
type unmarshalableValue struct {
	Ch chan int
}

func TestLogger_JSONMarshalError(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Add a field that cannot be marshaled to JSON (channels can't be marshaled)
	logger.WithField("bad_field", unmarshalableValue{Ch: make(chan int)}).Info("test message")

	// The logger should fall back to simple text output when JSON marshaling fails
	output := buf.String()
	if output == "" {
		t.Error("Expected fallback output when JSON marshaling fails")
	}

	// The output should not be valid JSON since it fell back to simple format
	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err == nil {
		t.Error("Expected JSON unmarshaling to fail for fallback format")
	}

	// Verify the fallback output contains the message
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected fallback output to contain message, got: %s", output)
	}

	// Verify the fallback output contains the log level
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected fallback output to contain log level, got: %s", output)
	}
}

func TestLogger_DifferentFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(DebugLevel, &buf)

	// Test with various types of values
	tests := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "str", "test"},
		{"int", "int", 42},
		{"int64", "int64", int64(9223372036854775807)},
		{"float32", "float32", float32(3.14)},
		{"float64", "float64", 3.14159265359},
		{"bool_true", "bool_t", true},
		{"bool_false", "bool_f", false},
		{"nil", "nil_val", nil},
		{"slice", "slice", []string{"a", "b", "c"}},
		{"map", "map", map[string]int{"one": 1, "two": 2}},
		{"struct", "struct", struct{ Name string }{"test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger.WithField(tt.key, tt.value).Debug("field test")

			if buf.Len() == 0 {
				t.Error("Expected log output")
			}

			var entry LogEntry
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("Failed to unmarshal log entry: %v", err)
			}

			if _, exists := entry.Fields[tt.key]; !exists {
				t.Errorf("Expected field %s to exist", tt.key)
			}
		})
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	tests := []struct {
		name        string
		loggerLevel LogLevel
		logMethod   string
		shouldLog   bool
	}{
		// Debug level logger
		{"debug->debug", DebugLevel, "debug", true},
		{"debug->info", DebugLevel, "info", true},
		{"debug->warn", DebugLevel, "warn", true},
		{"debug->error", DebugLevel, "error", true},

		// Info level logger
		{"info->debug", InfoLevel, "debug", false},
		{"info->info", InfoLevel, "info", true},
		{"info->warn", InfoLevel, "warn", true},
		{"info->error", InfoLevel, "error", true},

		// Warn level logger
		{"warn->debug", WarnLevel, "debug", false},
		{"warn->info", WarnLevel, "info", false},
		{"warn->warn", WarnLevel, "warn", true},
		{"warn->error", WarnLevel, "error", true},

		// Error level logger
		{"error->debug", ErrorLevel, "debug", false},
		{"error->info", ErrorLevel, "info", false},
		{"error->warn", ErrorLevel, "warn", false},
		{"error->error", ErrorLevel, "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(tt.loggerLevel, &buf)

			switch tt.logMethod {
			case "debug":
				logger.Debug("test")
			case "info":
				logger.Info("test")
			case "warn":
				logger.Warn("test")
			case "error":
				logger.Error("test")
			}

			hasOutput := buf.Len() > 0
			if hasOutput != tt.shouldLog {
				t.Errorf("Expected shouldLog=%v, got output=%v", tt.shouldLog, hasOutput)
			}
		})
	}
}

func TestLogger_ContextPropagation(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := NewLogger(InfoLevel, &buf)

	ctx := context.Background()
	ctx = WithLogger(ctx, baseLogger)
	ctx = WithRequestID(ctx, "test-req-id")
	ctx = WithUserID(ctx, "test-user-id")

	// Verify all values are properly stored and retrieved
	if reqID := GetRequestID(ctx); reqID != "test-req-id" {
		t.Errorf("Expected request ID 'test-req-id', got %s", reqID)
	}

	if userID := GetUserID(ctx); userID != "test-user-id" {
		t.Errorf("Expected user ID 'test-user-id', got %s", userID)
	}

	logger := GetLogger(ctx)
	if logger != baseLogger {
		t.Error("Expected to get the same logger from context")
	}

	// Test FromContext creates logger with both fields
	contextLogger := FromContext(ctx)
	contextLogger.Info("context test")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry.Fields["request_id"] != "test-req-id" {
		t.Errorf("Expected request_id in fields, got %v", entry.Fields["request_id"])
	}

	if entry.Fields["user_id"] != "test-user-id" {
		t.Errorf("Expected user_id in fields, got %v", entry.Fields["user_id"])
	}
}

func TestLogger_OutputToStdout(t *testing.T) {
	// Test that logger with nil output defaults to stdout
	logger := NewLogger(InfoLevel, nil)

	// Just verify it doesn't panic when logging
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Logger panicked with nil output: %v", r)
		}
	}()

	logger.Info("test message to stdout")
}

func TestLogger_ImmutableFields(t *testing.T) {
	var buf bytes.Buffer
	logger := NewLogger(InfoLevel, &buf)

	// Create base logger with a field
	logger1 := logger.WithField("logger", "1")

	// Create child logger from logger1 with additional field
	logger2 := logger1.WithField("logger", "2")

	// Log with both loggers and verify they have different field values
	buf.Reset()
	logger1.Info("from logger1")
	var entry1 LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry1); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	buf.Reset()
	logger2.Info("from logger2")
	var entry2 LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry2); err != nil {
		t.Fatalf("Failed to unmarshal log entry: %v", err)
	}

	if entry1.Fields["logger"] != "1" {
		t.Errorf("Expected logger1 field to be '1', got %v", entry1.Fields["logger"])
	}

	if entry2.Fields["logger"] != "2" {
		t.Errorf("Expected logger2 field to be '2', got %v", entry2.Fields["logger"])
	}
}
