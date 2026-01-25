package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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
