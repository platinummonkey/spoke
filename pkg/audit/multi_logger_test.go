package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiLogger_Log_Sync(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1, logger2)
	multiLogger.SetAsync(false) // Sync mode

	ctx := context.Background()
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAuthLogin,
		Status:    EventStatusSuccess,
		Metadata:  make(map[string]interface{}),
	}

	err := multiLogger.Log(ctx, event)
	require.NoError(t, err)

	// Both loggers should have received the event
	assert.Len(t, logger1.events, 1)
	assert.Len(t, logger2.events, 1)
}

func TestMultiLogger_Log_Async(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1, logger2)
	multiLogger.SetAsync(true) // Async mode

	ctx := context.Background()
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAuthLogin,
		Status:    EventStatusSuccess,
		Metadata:  make(map[string]interface{}),
	}

	err := multiLogger.Log(ctx, event)
	require.NoError(t, err)

	// Wait for async operations
	multiLogger.Wait()

	// Both loggers should have received the event
	assert.Len(t, logger1.events, 1)
	assert.Len(t, logger2.events, 1)
}

func TestMultiLogger_LogAuthentication(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1, logger2)
	multiLogger.SetAsync(false)

	ctx := context.Background()
	userID := int64(123)

	err := multiLogger.LogAuthentication(ctx, EventTypeAuthLogin, &userID, "testuser", EventStatusSuccess, "Login successful")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
	assert.Len(t, logger2.events, 1)
}

func TestMultiLogger_LogAuthorization(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	userID := int64(456)

	err := multiLogger.LogAuthorization(ctx, EventTypeAuthzAccessDenied, &userID, ResourceTypeModule, "test-module", EventStatusDenied, "Access denied")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
}

func TestMultiLogger_LogDataMutation(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	userID := int64(789)
	changes := &ChangeDetails{
		Before: map[string]interface{}{"name": "old"},
		After:  map[string]interface{}{"name": "new"},
	}

	err := multiLogger.LogDataMutation(ctx, EventTypeDataModuleUpdate, &userID, ResourceTypeModule, "test-module", changes, "Module updated")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
}

func TestMultiLogger_LogConfiguration(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	userID := int64(111)
	changes := &ChangeDetails{
		Before: map[string]interface{}{"enabled": false},
		After:  map[string]interface{}{"enabled": true},
	}

	err := multiLogger.LogConfiguration(ctx, EventTypeConfigChange, &userID, "sso-config", changes, "SSO enabled")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
}

func TestMultiLogger_LogAdminAction(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	adminID := int64(1)
	targetID := int64(999)

	err := multiLogger.LogAdminAction(ctx, EventTypeAdminUserDelete, &adminID, &targetID, "User deleted by admin")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
}

func TestMultiLogger_LogAccess(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	userID := int64(222)

	err := multiLogger.LogAccess(ctx, EventTypeAccessModuleRead, &userID, ResourceTypeModule, "test-module", "Module accessed")
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
}

func TestMultiLogger_LogHTTPRequest(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)

	ctx := context.Background()
	req := httptest.NewRequest("GET", "/test", nil)

	err := multiLogger.LogHTTPRequest(ctx, req, http.StatusOK, 100*time.Millisecond, nil)
	require.NoError(t, err)

	multiLogger.Wait()

	assert.Len(t, logger1.events, 1)
	assert.Equal(t, http.StatusOK, logger1.events[0].StatusCode)
}

func TestMultiLogger_Close(t *testing.T) {
	logger1 := &mockLogger{}
	logger2 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1, logger2)

	err := multiLogger.Close()
	require.NoError(t, err)
}

func TestMultiLogger_Empty(t *testing.T) {
	multiLogger := NewMultiLogger()

	ctx := context.Background()
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAuthLogin,
		Status:    EventStatusSuccess,
		Metadata:  make(map[string]interface{}),
	}

	// Should not error even with no loggers
	err := multiLogger.Log(ctx, event)
	require.NoError(t, err)
}

func TestMultiLogger_GetErrors(t *testing.T) {
	multiLogger := NewMultiLogger()

	errors := multiLogger.GetErrors()
	assert.Empty(t, errors)
}

func TestMultiLogger_Wait(t *testing.T) {
	logger1 := &mockLogger{}

	multiLogger := NewMultiLogger(logger1)
	multiLogger.SetAsync(true)

	ctx := context.Background()

	// Log multiple events
	for i := 0; i < 5; i++ {
		event := &AuditEvent{
			Timestamp: time.Now(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			Metadata:  make(map[string]interface{}),
		}
		multiLogger.Log(ctx, event)
	}

	// Wait for all async operations
	multiLogger.Wait()

	// All events should be logged
	assert.Len(t, logger1.events, 5)
}
