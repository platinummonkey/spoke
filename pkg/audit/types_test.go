package audit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditEvent_ToJSON(t *testing.T) {
	userID := int64(123)
	event := &AuditEvent{
		ID:           1,
		Timestamp:    time.Now().UTC(),
		EventType:    EventTypeAuthLogin,
		Status:       EventStatusSuccess,
		UserID:       &userID,
		Username:     "testuser",
		ResourceType: ResourceTypeUser,
		ResourceID:   "123",
		IPAddress:    "192.168.1.1",
		Message:      "User logged in successfully",
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
	}

	jsonData, err := event.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify we can parse it back
	parsed, err := FromJSON(jsonData)
	require.NoError(t, err)
	assert.Equal(t, event.ID, parsed.ID)
	assert.Equal(t, event.EventType, parsed.EventType)
	assert.Equal(t, event.Status, parsed.Status)
	assert.Equal(t, event.Username, parsed.Username)
}

func TestEventType_Constants(t *testing.T) {
	// Test that event type constants are properly defined
	assert.Equal(t, EventType("auth.login"), EventTypeAuthLogin)
	assert.Equal(t, EventType("auth.logout"), EventTypeAuthLogout)
	assert.Equal(t, EventType("data.module_create"), EventTypeDataModuleCreate)
	assert.Equal(t, EventType("authz.access_denied"), EventTypeAuthzAccessDenied)
}

func TestEventStatus_Constants(t *testing.T) {
	assert.Equal(t, EventStatus("success"), EventStatusSuccess)
	assert.Equal(t, EventStatus("failure"), EventStatusFailure)
	assert.Equal(t, EventStatus("denied"), EventStatusDenied)
}

func TestResourceType_Constants(t *testing.T) {
	assert.Equal(t, ResourceType("module"), ResourceTypeModule)
	assert.Equal(t, ResourceType("version"), ResourceTypeVersion)
	assert.Equal(t, ResourceType("user"), ResourceTypeUser)
}

func TestChangeDetails_JSON(t *testing.T) {
	changes := &ChangeDetails{
		Before: map[string]interface{}{
			"name":  "old-name",
			"value": 100,
		},
		After: map[string]interface{}{
			"name":  "new-name",
			"value": 200,
		},
	}

	jsonData, err := json.Marshal(changes)
	require.NoError(t, err)

	var parsed ChangeDetails
	err = json.Unmarshal(jsonData, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "old-name", parsed.Before["name"])
	assert.Equal(t, "new-name", parsed.After["name"])
}

func TestDefaultRetentionPolicy(t *testing.T) {
	policy := DefaultRetentionPolicy()

	assert.Equal(t, 90, policy.RetentionDays)
	assert.True(t, policy.ArchiveEnabled)
	assert.Equal(t, "/var/spoke/audit-archive", policy.ArchivePath)
	assert.True(t, policy.CompressArchive)
}

func TestSearchFilter_Defaults(t *testing.T) {
	filter := SearchFilter{}

	assert.Nil(t, filter.StartTime)
	assert.Nil(t, filter.EndTime)
	assert.Nil(t, filter.UserID)
	assert.Equal(t, "", filter.Username)
	assert.Equal(t, 0, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
}

func TestAuditStats_Initialization(t *testing.T) {
	stats := &AuditStats{
		EventsByType:         make(map[EventType]int64),
		EventsByStatus:       make(map[EventStatus]int64),
		EventsByUser:         make(map[int64]int64),
		EventsByOrganization: make(map[int64]int64),
		EventsByResource:     make(map[ResourceType]int64),
	}

	assert.NotNil(t, stats.EventsByType)
	assert.NotNil(t, stats.EventsByStatus)
	assert.Equal(t, 0, len(stats.EventsByType))
	assert.Equal(t, int64(0), stats.TotalEvents)
}

func TestExportFormat_Constants(t *testing.T) {
	assert.Equal(t, ExportFormat("json"), ExportFormatJSON)
	assert.Equal(t, ExportFormat("csv"), ExportFormatCSV)
	assert.Equal(t, ExportFormat("ndjson"), ExportFormatNDJSON)
}
