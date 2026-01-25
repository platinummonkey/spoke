package audit

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportJSON(t *testing.T) {
	userID := int64(123)
	events := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Username:  "testuser",
			Metadata:  make(map[string]interface{}),
		},
		{
			ID:        2,
			Timestamp: time.Now().UTC(),
			EventType: EventTypeDataModuleCreate,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Metadata:  make(map[string]interface{}),
		},
	}

	data, err := exportJSON(events)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var parsed []*AuditEvent
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 2)
}

func TestExportNDJSON(t *testing.T) {
	userID := int64(456)
	events := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Username:  "user1",
			Metadata:  make(map[string]interface{}),
		},
		{
			ID:        2,
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogout,
			Status:    EventStatusSuccess,
			UserID:    &userID,
			Username:  "user1",
			Metadata:  make(map[string]interface{}),
		},
	}

	data, err := exportNDJSON(events)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify each line is valid JSON
	lines := strings.Split(string(data), "\n")
	validLines := 0
	for _, line := range lines {
		if line == "" {
			continue
		}
		var event AuditEvent
		err := json.Unmarshal([]byte(line), &event)
		require.NoError(t, err)
		validLines++
	}
	assert.Equal(t, 2, validLines)
}

func TestExportCSV(t *testing.T) {
	userID := int64(789)
	orgID := int64(1)
	events := []*AuditEvent{
		{
			ID:             1,
			Timestamp:      time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			EventType:      EventTypeAuthLogin,
			Status:         EventStatusSuccess,
			UserID:         &userID,
			Username:       "testuser",
			OrganizationID: &orgID,
			ResourceType:   ResourceTypeUser,
			ResourceID:     "123",
			IPAddress:      "192.168.1.1",
			Message:        "Login successful",
			Metadata:       make(map[string]interface{}),
		},
	}

	data, err := exportCSV(events)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify CSV format
	lines := strings.Split(string(data), "\n")
	assert.GreaterOrEqual(t, len(lines), 2) // At least header + 1 row

	// Check header
	header := lines[0]
	assert.Contains(t, header, "ID")
	assert.Contains(t, header, "Timestamp")
	assert.Contains(t, header, "EventType")
	assert.Contains(t, header, "Status")

	// Check data row
	dataRow := lines[1]
	assert.Contains(t, dataRow, "1") // ID
	assert.Contains(t, dataRow, "testuser")
	assert.Contains(t, dataRow, "auth.login")
}

func TestExportCSV_EmptyEvents(t *testing.T) {
	events := []*AuditEvent{}

	data, err := exportCSV(events)
	require.NoError(t, err)
	assert.NotEmpty(t, data) // Should still have header

	lines := strings.Split(string(data), "\n")
	assert.GreaterOrEqual(t, len(lines), 1) // At least header
}

func TestExportCSV_NilValues(t *testing.T) {
	events := []*AuditEvent{
		{
			ID:        1,
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			// All pointer fields are nil
			Metadata: make(map[string]interface{}),
		},
	}

	data, err := exportCSV(events)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Should not panic and should handle nil values gracefully
	lines := strings.Split(string(data), "\n")
	assert.GreaterOrEqual(t, len(lines), 2)
}

func TestFormatInt64Ptr(t *testing.T) {
	// Test with nil
	assert.Equal(t, "", formatInt64Ptr(nil))

	// Test with value
	val := int64(123)
	assert.Equal(t, "123", formatInt64Ptr(&val))

	// Test with zero
	zero := int64(0)
	assert.Equal(t, "0", formatInt64Ptr(&zero))

	// Test with negative
	neg := int64(-456)
	assert.Equal(t, "-456", formatInt64Ptr(&neg))
}
