package audit

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileLogger_Basic(t *testing.T) {
	// Create temporary directory for test logs
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create file logger
	config := FileLoggerConfig{
		BasePath: tmpDir,
		Rotate:   false,
		MaxSize:  1024 * 1024,
		MaxFiles: 5,
	}

	logger, err := NewFileLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	// Log an event
	ctx := context.Background()
	userID := int64(123)
	event := &AuditEvent{
		Timestamp:    time.Now().UTC(),
		EventType:    EventTypeAuthLogin,
		Status:       EventStatusSuccess,
		UserID:       &userID,
		Username:     "testuser",
		ResourceType: ResourceTypeUser,
		IPAddress:    "192.168.1.1",
		Message:      "Test login",
		Metadata:     make(map[string]interface{}),
	}

	err = logger.Log(ctx, event)
	require.NoError(t, err)

	// Verify log file was created
	logFile := filepath.Join(tmpDir, "audit.log")
	assert.FileExists(t, logFile)

	// Read and verify content
	events, err := logger.ReadLogs(10)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, EventTypeAuthLogin, events[0].EventType)
	assert.Equal(t, "testuser", events[0].Username)
}

func TestFileLogger_MultipleEvents(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := FileLoggerConfig{
		BasePath: tmpDir,
		Rotate:   false,
	}

	logger, err := NewFileLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()

	// Log multiple events
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			Timestamp: time.Now().UTC(),
			EventType: EventTypeDataModuleCreate,
			Status:    EventStatusSuccess,
			Message:   "Test event",
			Metadata:  make(map[string]interface{}),
		}
		err = logger.Log(ctx, event)
		require.NoError(t, err)
	}

	// Read all events
	events, err := logger.ReadLogs(10)
	require.NoError(t, err)
	assert.Len(t, events, 5)
}

func TestFileLogger_LogAuthentication(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := FileLoggerConfig{
		BasePath: tmpDir,
		Rotate:   false,
	}

	logger, err := NewFileLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	userID := int64(456)

	err = logger.LogAuthentication(ctx, EventTypeAuthLogin, &userID, "testuser", EventStatusSuccess, "Login successful")
	require.NoError(t, err)

	events, err := logger.ReadLogs(1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, EventTypeAuthLogin, events[0].EventType)
	assert.Equal(t, "testuser", events[0].Username)
	assert.Equal(t, EventStatusSuccess, events[0].Status)
}

func TestFileLogger_LogDataMutation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := FileLoggerConfig{
		BasePath: tmpDir,
		Rotate:   false,
	}

	logger, err := NewFileLogger(config)
	require.NoError(t, err)
	defer logger.Close()

	ctx := context.Background()
	userID := int64(789)
	changes := &ChangeDetails{
		Before: map[string]interface{}{"name": "old"},
		After:  map[string]interface{}{"name": "new"},
	}

	err = logger.LogDataMutation(ctx, EventTypeDataModuleUpdate, &userID, ResourceTypeModule, "test-module", changes, "Module updated")
	require.NoError(t, err)

	events, err := logger.ReadLogs(1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, EventTypeDataModuleUpdate, events[0].EventType)
	assert.Equal(t, ResourceTypeModule, events[0].ResourceType)
	assert.NotNil(t, events[0].Changes)
}

func TestDefaultFileLoggerConfig(t *testing.T) {
	config := DefaultFileLoggerConfig()

	assert.Equal(t, "/var/log/spoke/audit", config.BasePath)
	assert.True(t, config.Rotate)
	assert.Equal(t, int64(100*1024*1024), config.MaxSize)
	assert.Equal(t, 10, config.MaxFiles)
}
