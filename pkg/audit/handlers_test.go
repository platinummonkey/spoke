package audit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore for testing handlers
type mockStore struct {
	events []*AuditEvent
	stats  *AuditStats
}

func (m *mockStore) Search(ctx context.Context, filter SearchFilter) ([]*AuditEvent, error) {
	return m.events, nil
}

func (m *mockStore) Get(ctx context.Context, id int64) (*AuditEvent, error) {
	for _, event := range m.events {
		if event.ID == id {
			return event, nil
		}
	}
	return nil, nil
}

func (m *mockStore) GetStats(ctx context.Context, startTime, endTime *time.Time) (*AuditStats, error) {
	return m.stats, nil
}

func (m *mockStore) Export(ctx context.Context, filter SearchFilter, format ExportFormat) ([]byte, error) {
	switch format {
	case ExportFormatCSV:
		return exportCSV(m.events)
	case ExportFormatNDJSON:
		return exportNDJSON(m.events)
	default:
		return exportJSON(m.events)
	}
}

func (m *mockStore) Cleanup(ctx context.Context, policy RetentionPolicy) (int64, error) {
	return 0, nil
}

func TestHandlers_ListEvents(t *testing.T) {
	userID := int64(123)
	mockEvents := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Username:  "testuser",
			Metadata:  make(map[string]interface{}),
		},
	}

	store := &mockStore{events: mockEvents}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/events?limit=10", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	events := response["events"].([]interface{})
	assert.Len(t, events, 1)
}

func TestHandlers_GetEvent(t *testing.T) {
	userID := int64(456)
	mockEvents := []*AuditEvent{
		{
			ID:        42,
			Timestamp: time.Now(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Metadata:  make(map[string]interface{}),
		},
	}

	store := &mockStore{events: mockEvents}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/events/42", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var event AuditEvent
	err := json.NewDecoder(rec.Body).Decode(&event)
	require.NoError(t, err)
	assert.Equal(t, int64(42), event.ID)
}

func TestHandlers_GetEvent_NotFound(t *testing.T) {
	store := &mockStore{events: []*AuditEvent{}}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/events/999", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandlers_ExportEvents_JSON(t *testing.T) {
	mockEvents := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			Metadata:  make(map[string]interface{}),
		},
	}

	store := &mockStore{events: mockEvents}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/export?format=json", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "audit-logs.json")
}

func TestHandlers_ExportEvents_CSV(t *testing.T) {
	mockEvents := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			Metadata:  make(map[string]interface{}),
		},
	}

	store := &mockStore{events: mockEvents}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/export?format=csv", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/csv")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "audit-logs.csv")
}

func TestHandlers_GetStats(t *testing.T) {
	mockStats := &AuditStats{
		TotalEvents:        100,
		UniqueUsers:        10,
		FailedAuthAttempts: 5,
		EventsByType: map[EventType]int64{
			EventTypeAuthLogin: 50,
		},
		EventsByStatus: map[EventStatus]int64{
			EventStatusSuccess: 95,
			EventStatusFailure: 5,
		},
	}

	store := &mockStore{stats: mockStats}
	handlers := NewHandlers(store)

	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	req := httptest.NewRequest("GET", "/audit/stats", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var stats AuditStats
	err := json.NewDecoder(rec.Body).Decode(&stats)
	require.NoError(t, err)
	assert.Equal(t, int64(100), stats.TotalEvents)
	assert.Equal(t, int64(10), stats.UniqueUsers)
}

func TestParseFilter(t *testing.T) {
	handlers := &Handlers{}

	req := httptest.NewRequest("GET", "/audit/events?user_id=123&limit=50&offset=10&status=success", nil)

	filter := handlers.parseFilter(req)

	require.NotNil(t, filter.UserID)
	assert.Equal(t, int64(123), *filter.UserID)
	assert.Equal(t, 50, filter.Limit)
	assert.Equal(t, 10, filter.Offset)
	require.NotNil(t, filter.Status)
	assert.Equal(t, EventStatusSuccess, *filter.Status)
}

func TestParseFilter_TimeRange(t *testing.T) {
	handlers := &Handlers{}

	startTime := "2024-01-01T00:00:00Z"
	endTime := "2024-01-31T23:59:59Z"

	req := httptest.NewRequest("GET", "/audit/events?start_time="+startTime+"&end_time="+endTime, nil)

	filter := handlers.parseFilter(req)

	require.NotNil(t, filter.StartTime)
	require.NotNil(t, filter.EndTime)
}

func TestParseFilter_EventTypes(t *testing.T) {
	handlers := &Handlers{}

	req := httptest.NewRequest("GET", "/audit/events?event_types=auth.login,auth.logout", nil)

	filter := handlers.parseFilter(req)

	assert.Len(t, filter.EventTypes, 2)
	assert.Equal(t, EventTypeAuthLogin, filter.EventTypes[0])
	assert.Equal(t, EventTypeAuthLogout, filter.EventTypes[1])
}

func TestParseCommaSeparated(t *testing.T) {
	// Test normal case
	result := parseCommaSeparated("auth.login,auth.logout,data.create")
	assert.Len(t, result, 3)
	assert.Equal(t, "auth.login", result[0])
	assert.Equal(t, "auth.logout", result[1])
	assert.Equal(t, "data.create", result[2])

	// Test with spaces
	result = parseCommaSeparated("auth.login , auth.logout , data.create")
	assert.Len(t, result, 3)
	assert.Equal(t, "auth.login", result[0])

	// Test empty string
	result = parseCommaSeparated("")
	assert.Nil(t, result)

	// Test single value
	result = parseCommaSeparated("auth.login")
	assert.Len(t, result, 1)
	assert.Equal(t, "auth.login", result[0])
}
