package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLogger for testing (thread-safe for async operations)
type mockLogger struct {
	mu     sync.Mutex
	events []*AuditEvent
}

func (m *mockLogger) Log(ctx context.Context, event *AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockLogger) LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error {
	return nil
}

func (m *mockLogger) LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error {
	return nil
}

func (m *mockLogger) LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error {
	return nil
}

func (m *mockLogger) LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error {
	return nil
}

func (m *mockLogger) LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error {
	return nil
}

func (m *mockLogger) LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error {
	return nil
}

func (m *mockLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	event := &AuditEvent{
		Timestamp:  time.Now().UTC(),
		EventType:  EventTypeAccessModuleRead,
		Method:     r.Method,
		Path:       r.URL.Path,
		StatusCode: statusCode,
		Metadata:   map[string]interface{}{"duration_ms": duration.Milliseconds()},
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *mockLogger) Close() error {
	return nil
}

// GetEvents returns a copy of events (thread-safe)
func (m *mockLogger) GetEvents() []*AuditEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*AuditEvent, len(m.events))
	copy(result, m.events)
	return result
}

func TestMiddleware_Handler(t *testing.T) {
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, true)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := middleware.Handler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Len(t, logger.events, 1)
	assert.Equal(t, "GET", logger.events[0].Method)
	assert.Equal(t, "/test", logger.events[0].Path)
	assert.Equal(t, http.StatusOK, logger.events[0].StatusCode)
}

func TestMiddleware_Handler_LogMutationsOnly(t *testing.T) {
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, false) // Only log mutations

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Handler(handler)

	// GET request (should not be logged)
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	assert.Len(t, logger.events, 0)

	// POST request (should be logged)
	req = httptest.NewRequest("POST", "/test", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	assert.Len(t, logger.events, 1)
}

func TestMiddleware_Handler_LogErrors(t *testing.T) {
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, false) // Only log mutations and errors

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrapped := middleware.Handler(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Should log because of error status
	assert.Len(t, logger.events, 1)
	assert.Equal(t, http.StatusInternalServerError, logger.events[0].StatusCode)
}

func TestMiddleware_Handler_LogSensitiveEndpoints(t *testing.T) {
	logger := &mockLogger{}
	middleware := NewMiddleware(logger, false)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware.Handler(handler)

	// Test auth endpoint
	req := httptest.NewRequest("GET", "/auth/login", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	assert.Len(t, logger.events, 1)

	// Test admin endpoint
	logger.events = nil
	req = httptest.NewRequest("GET", "/admin/users", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	assert.Len(t, logger.events, 1)

	// Test audit endpoint
	logger.events = nil
	req = httptest.NewRequest("GET", "/audit/events", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)
	assert.Len(t, logger.events, 1)
}

func TestResponseWriter_StatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
	assert.True(t, rw.written)

	// Second WriteHeader should not change status
	rw.WriteHeader(http.StatusBadRequest)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
}

func TestResponseWriter_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Write should call WriteHeader if not already written
	n, err := rw.Write([]byte("test"))
	require.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.True(t, rw.written)
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestWithAuditContext(t *testing.T) {
	ctx := context.Background()
	userID := int64(123)
	username := "testuser"
	orgID := int64(456)
	tokenID := int64(789)

	ctx = WithAuditContext(ctx, &userID, username, &orgID, &tokenID)

	// Retrieve values
	retrievedUserID, retrievedUsername, retrievedOrgID, retrievedTokenID := GetAuditContext(ctx)

	require.NotNil(t, retrievedUserID)
	assert.Equal(t, userID, *retrievedUserID)
	assert.Equal(t, username, retrievedUsername)
	require.NotNil(t, retrievedOrgID)
	assert.Equal(t, orgID, *retrievedOrgID)
	require.NotNil(t, retrievedTokenID)
	assert.Equal(t, tokenID, *retrievedTokenID)
}

func TestGetAuditContext_Empty(t *testing.T) {
	ctx := context.Background()

	userID, username, orgID, tokenID := GetAuditContext(ctx)

	assert.Nil(t, userID)
	assert.Equal(t, "", username)
	assert.Nil(t, orgID)
	assert.Nil(t, tokenID)
}

func TestQuickLog(t *testing.T) {
	logger := &mockLogger{}
	ctx := WithLogger(context.Background(), logger)

	err := QuickLog(ctx, EventTypeAuthLogin, EventStatusSuccess, "Test message")
	require.NoError(t, err)

	assert.Len(t, logger.events, 1)
	assert.Equal(t, EventTypeAuthLogin, logger.events[0].EventType)
	assert.Equal(t, EventStatusSuccess, logger.events[0].Status)
	assert.Equal(t, "Test message", logger.events[0].Message)
}

func TestLogSuccess(t *testing.T) {
	logger := &mockLogger{}
	ctx := WithLogger(context.Background(), logger)

	metadata := map[string]interface{}{
		"key": "value",
	}

	err := LogSuccess(ctx, EventTypeDataModuleCreate, "Module created", metadata)
	require.NoError(t, err)

	assert.Len(t, logger.events, 1)
	assert.Equal(t, EventStatusSuccess, logger.events[0].Status)
	assert.Equal(t, "Module created", logger.events[0].Message)
	assert.Equal(t, "value", logger.events[0].Metadata["key"])
}

func TestLogFailure(t *testing.T) {
	logger := &mockLogger{}
	ctx := WithLogger(context.Background(), logger)

	testErr := assert.AnError

	err := LogFailure(ctx, EventTypeDataModuleCreate, "Failed to create module", testErr)
	require.NoError(t, err)

	assert.Len(t, logger.events, 1)
	assert.Equal(t, EventStatusFailure, logger.events[0].Status)
	assert.Equal(t, "Failed to create module", logger.events[0].Message)
	assert.NotEmpty(t, logger.events[0].ErrorMessage)
}

func TestLogDenied(t *testing.T) {
	logger := &mockLogger{}
	ctx := WithLogger(context.Background(), logger)

	err := LogDenied(ctx, EventTypeAuthzAccessDenied, ResourceTypeModule, "test-module", "Insufficient permissions")
	require.NoError(t, err)

	assert.Len(t, logger.events, 1)
	assert.Equal(t, EventStatusDenied, logger.events[0].Status)
	assert.Equal(t, ResourceTypeModule, logger.events[0].ResourceType)
	assert.Equal(t, "test-module", logger.events[0].ResourceID)
	assert.Contains(t, logger.events[0].Message, "Access denied")
}
