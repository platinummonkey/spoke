package audit

import (
	"encoding/json"
	"time"
)

// EventType represents the category of audit event
type EventType string

const (
	// Authentication events
	EventTypeAuthLogin              EventType = "auth.login"
	EventTypeAuthLogout             EventType = "auth.logout"
	EventTypeAuthLoginFailed        EventType = "auth.login_failed"
	EventTypeAuthPasswordChange     EventType = "auth.password_change"
	EventTypeAuthTokenCreate        EventType = "auth.token_create"
	EventTypeAuthTokenRevoke        EventType = "auth.token_revoke"
	EventTypeAuthTokenValidate      EventType = "auth.token_validate"
	EventTypeAuthTokenValidateFail  EventType = "auth.token_validate_fail"

	// Authorization events
	EventTypeAuthzPermissionCheck   EventType = "authz.permission_check"
	EventTypeAuthzPermissionGrant   EventType = "authz.permission_grant"
	EventTypeAuthzPermissionRevoke  EventType = "authz.permission_revoke"
	EventTypeAuthzRoleChange        EventType = "authz.role_change"
	EventTypeAuthzAccessDenied      EventType = "authz.access_denied"

	// Data mutation events
	EventTypeDataModuleCreate       EventType = "data.module_create"
	EventTypeDataModuleUpdate       EventType = "data.module_update"
	EventTypeDataModuleDelete       EventType = "data.module_delete"
	EventTypeDataVersionCreate      EventType = "data.version_create"
	EventTypeDataVersionUpdate      EventType = "data.version_update"
	EventTypeDataVersionDelete      EventType = "data.version_delete"
	EventTypeDataFileUpload         EventType = "data.file_upload"
	EventTypeDataFileDelete         EventType = "data.file_delete"

	// Configuration events
	EventTypeConfigChange           EventType = "config.change"
	EventTypeConfigSSOUpdate        EventType = "config.sso_update"
	EventTypeConfigWebhookCreate    EventType = "config.webhook_create"
	EventTypeConfigWebhookUpdate    EventType = "config.webhook_update"
	EventTypeConfigWebhookDelete    EventType = "config.webhook_delete"

	// Admin events
	EventTypeAdminUserCreate        EventType = "admin.user_create"
	EventTypeAdminUserUpdate        EventType = "admin.user_update"
	EventTypeAdminUserDelete        EventType = "admin.user_delete"
	EventTypeAdminUserActivate      EventType = "admin.user_activate"
	EventTypeAdminUserDeactivate    EventType = "admin.user_deactivate"
	EventTypeAdminOrgCreate         EventType = "admin.org_create"
	EventTypeAdminOrgUpdate         EventType = "admin.org_update"
	EventTypeAdminOrgDelete         EventType = "admin.org_delete"
	EventTypeAdminOrgMemberAdd      EventType = "admin.org_member_add"
	EventTypeAdminOrgMemberRemove   EventType = "admin.org_member_remove"
	EventTypeAdminOrgMemberRoleChange EventType = "admin.org_member_role_change"

	// Read/access events (for sensitive operations)
	EventTypeAccessModuleRead       EventType = "access.module_read"
	EventTypeAccessVersionRead      EventType = "access.version_read"
	EventTypeAccessFileRead         EventType = "access.file_read"
	EventTypeAccessCompileDownload  EventType = "access.compile_download"
)

// EventStatus represents the outcome of an event
type EventStatus string

const (
	EventStatusSuccess EventStatus = "success"
	EventStatusFailure EventStatus = "failure"
	EventStatusDenied  EventStatus = "denied"
)

// ResourceType represents the type of resource being accessed
type ResourceType string

const (
	ResourceTypeModule       ResourceType = "module"
	ResourceTypeVersion      ResourceType = "version"
	ResourceTypeFile         ResourceType = "file"
	ResourceTypeUser         ResourceType = "user"
	ResourceTypeOrganization ResourceType = "organization"
	ResourceTypeToken        ResourceType = "token"
	ResourceTypePermission   ResourceType = "permission"
	ResourceTypeWebhook      ResourceType = "webhook"
	ResourceTypeConfig       ResourceType = "config"
)

// AuditEvent represents a single audit log entry
type AuditEvent struct {
	// Core fields
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	EventType EventType `json:"event_type"`
	Status    EventStatus `json:"status"`

	// Actor information
	UserID         *int64  `json:"user_id,omitempty"`
	Username       string  `json:"username,omitempty"`
	OrganizationID *int64  `json:"organization_id,omitempty"`
	TokenID        *int64  `json:"token_id,omitempty"`

	// Resource information
	ResourceType ResourceType `json:"resource_type,omitempty"`
	ResourceID   string       `json:"resource_id,omitempty"`
	ResourceName string       `json:"resource_name,omitempty"`

	// Request context
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`

	// Additional details
	Message      string                 `json:"message,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// Changes tracking (before/after for updates)
	Changes *ChangeDetails `json:"changes,omitempty"`
}

// ChangeDetails tracks before/after values for updates
type ChangeDetails struct {
	Before map[string]interface{} `json:"before,omitempty"`
	After  map[string]interface{} `json:"after,omitempty"`
}

// ToJSON converts the audit event to JSON
func (e *AuditEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON parses an audit event from JSON
func FromJSON(data []byte) (*AuditEvent, error) {
	var event AuditEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}

// SearchFilter represents filters for searching audit logs
type SearchFilter struct {
	// Time range
	StartTime *time.Time
	EndTime   *time.Time

	// Actor filters
	UserID         *int64
	Username       string
	OrganizationID *int64

	// Event filters
	EventTypes []EventType
	Status     *EventStatus

	// Resource filters
	ResourceType ResourceType
	ResourceID   string
	ResourceName string

	// Request context filters
	IPAddress string
	Method    string
	Path      string

	// Pagination
	Limit  int
	Offset int

	// Sorting
	SortBy    string // field name to sort by
	SortOrder string // "asc" or "desc"
}

// ExportFormat represents the format for exporting audit logs
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatNDJSON ExportFormat = "ndjson" // Newline-delimited JSON
)

// AuditStats represents statistics about audit logs
type AuditStats struct {
	TotalEvents           int64            `json:"total_events"`
	EventsByType          map[EventType]int64 `json:"events_by_type"`
	EventsByStatus        map[EventStatus]int64 `json:"events_by_status"`
	EventsByUser          map[int64]int64  `json:"events_by_user"`
	EventsByOrganization  map[int64]int64  `json:"events_by_organization"`
	EventsByResource      map[ResourceType]int64 `json:"events_by_resource"`
	UniqueUsers           int64            `json:"unique_users"`
	UniqueIPs             int64            `json:"unique_ips"`
	FailedAuthAttempts    int64            `json:"failed_auth_attempts"`
	AccessDenials         int64            `json:"access_denials"`
	TimeRange             *TimeRange       `json:"time_range,omitempty"`
}

// TimeRange represents a time range for statistics
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// RetentionPolicy defines how long audit logs should be kept
type RetentionPolicy struct {
	// RetentionDays is the number of days to keep audit logs
	RetentionDays int

	// ArchiveEnabled determines if old logs should be archived instead of deleted
	ArchiveEnabled bool

	// ArchivePath is where archived logs should be stored
	ArchivePath string

	// CompressArchive determines if archived logs should be compressed
	CompressArchive bool
}

// DefaultRetentionPolicy returns a default retention policy (90 days)
func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		RetentionDays:   90,
		ArchiveEnabled:  true,
		ArchivePath:     "/var/spoke/audit-archive",
		CompressArchive: true,
	}
}
