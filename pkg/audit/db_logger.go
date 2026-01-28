package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/lib/pq"
)

// DBLogger implements audit logging to PostgreSQL database
type DBLogger struct {
	db *sql.DB
}

// NewDBLogger creates a new database-based audit logger
func NewDBLogger(db *sql.DB) (*DBLogger, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	logger := &DBLogger{
		db: db,
	}

	// Ensure the audit_logs table exists
	if err := logger.ensureTable(); err != nil {
		return nil, fmt.Errorf("failed to ensure audit_logs table: %w", err)
	}

	return logger, nil
}

// ensureTable creates the audit_logs table if it doesn't exist
func (l *DBLogger) ensureTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
		event_type VARCHAR(100) NOT NULL,
		status VARCHAR(20) NOT NULL,
		user_id BIGINT,
		username VARCHAR(255),
		organization_id BIGINT,
		token_id BIGINT,
		resource_type VARCHAR(50),
		resource_id VARCHAR(255),
		resource_name VARCHAR(255),
		ip_address VARCHAR(45),
		user_agent TEXT,
		request_id VARCHAR(100),
		method VARCHAR(10),
		path TEXT,
		status_code INTEGER,
		message TEXT,
		error_message TEXT,
		metadata JSONB,
		changes JSONB,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	);

	-- Create indexes for common query patterns
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_event_type ON audit_logs(event_type);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_organization_id ON audit_logs(organization_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_status ON audit_logs(status);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_ip_address ON audit_logs(ip_address);
	`

	_, err := l.db.Exec(query)
	return err
}

// Log logs an audit event to the database
func (l *DBLogger) Log(ctx context.Context, event *AuditEvent) error {
	// Serialize metadata and changes to JSON
	var metadataJSON, changesJSON []byte
	var err error

	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	if event.Changes != nil {
		changesJSON, err = json.Marshal(event.Changes)
		if err != nil {
			return fmt.Errorf("failed to marshal changes: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (
			timestamp, event_type, status,
			user_id, username, organization_id, token_id,
			resource_type, resource_id, resource_name,
			ip_address, user_agent, request_id,
			method, path, status_code,
			message, error_message, metadata, changes
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7,
			$8, $9, $10,
			$11, $12, $13,
			$14, $15, $16,
			$17, $18, $19, $20
		) RETURNING id
	`

	err = l.db.QueryRowContext(ctx, query,
		event.Timestamp, event.EventType, event.Status,
		event.UserID, event.Username, event.OrganizationID, event.TokenID,
		event.ResourceType, event.ResourceID, event.ResourceName,
		event.IPAddress, event.UserAgent, event.RequestID,
		event.Method, event.Path, event.StatusCode,
		event.Message, event.ErrorMessage, metadataJSON, changesJSON,
	).Scan(&event.ID)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

// LogAuthentication logs an authentication event
func (l *DBLogger) LogAuthentication(ctx context.Context, eventType EventType, userID *int64, username string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.Username = username
	event.Message = message
	event.ResourceType = ResourceTypeUser

	return l.Log(ctx, event)
}

// LogAuthorization logs an authorization event
func (l *DBLogger) LogAuthorization(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, status EventStatus, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, status)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return l.Log(ctx, event)
}

// LogDataMutation logs a data mutation event
func (l *DBLogger) LogDataMutation(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return l.Log(ctx, event)
}

// LogConfiguration logs a configuration change event
func (l *DBLogger) LogConfiguration(ctx context.Context, eventType EventType, userID *int64, resourceID string, changes *ChangeDetails, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = ResourceTypeConfig
	event.ResourceID = resourceID
	event.Changes = changes
	event.Message = message

	return l.Log(ctx, event)
}

// LogAdminAction logs an admin action event
func (l *DBLogger) LogAdminAction(ctx context.Context, eventType EventType, adminUserID *int64, targetUserID *int64, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = adminUserID
	event.Message = message
	if targetUserID != nil {
		event.Metadata["target_user_id"] = *targetUserID
	}

	return l.Log(ctx, event)
}

// LogAccess logs a resource access event
func (l *DBLogger) LogAccess(ctx context.Context, eventType EventType, userID *int64, resourceType ResourceType, resourceID string, message string) error {
	event := buildBaseEvent(ctx, nil, eventType, EventStatusSuccess)
	event.UserID = userID
	event.ResourceType = resourceType
	event.ResourceID = resourceID
	event.Message = message

	return l.Log(ctx, event)
}

// LogHTTPRequest logs an HTTP request
func (l *DBLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	eventType := EventTypeAccessModuleRead
	status := EventStatusSuccess

	if statusCode >= 400 {
		status = EventStatusFailure
	}
	if statusCode == 403 {
		status = EventStatusDenied
	}

	event := buildBaseEvent(ctx, r, eventType, status)
	event.StatusCode = statusCode
	event.Metadata["duration_ms"] = duration.Milliseconds()

	if err != nil {
		event.ErrorMessage = err.Error()
	}

	return l.Log(ctx, event)
}

// Search searches audit logs based on filters
func (l *DBLogger) Search(ctx context.Context, filter SearchFilter) ([]*AuditEvent, error) {
	query := `
		SELECT
			id, timestamp, event_type, status,
			user_id, username, organization_id, token_id,
			resource_type, resource_id, resource_name,
			ip_address, user_agent, request_id,
			method, path, status_code,
			message, error_message, metadata, changes
		FROM audit_logs
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Build WHERE clause based on filters
	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *filter.StartTime)
		argCount++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *filter.EndTime)
		argCount++
	}

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *filter.UserID)
		argCount++
	}

	if filter.Username != "" {
		query += fmt.Sprintf(" AND username = $%d", argCount)
		args = append(args, filter.Username)
		argCount++
	}

	if filter.OrganizationID != nil {
		query += fmt.Sprintf(" AND organization_id = $%d", argCount)
		args = append(args, *filter.OrganizationID)
		argCount++
	}

	if len(filter.EventTypes) > 0 {
		query += fmt.Sprintf(" AND event_type = ANY($%d)", argCount)
		eventTypeStrs := make([]string, len(filter.EventTypes))
		for i, et := range filter.EventTypes {
			eventTypeStrs[i] = string(et)
		}
		args = append(args, pq.Array(eventTypeStrs))
		argCount++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, string(*filter.Status))
		argCount++
	}

	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argCount)
		args = append(args, string(filter.ResourceType))
		argCount++
	}

	if filter.ResourceID != "" {
		query += fmt.Sprintf(" AND resource_id = $%d", argCount)
		args = append(args, filter.ResourceID)
		argCount++
	}

	if filter.IPAddress != "" {
		query += fmt.Sprintf(" AND ip_address = $%d", argCount)
		args = append(args, filter.IPAddress)
		argCount++
	}

	if filter.Method != "" {
		query += fmt.Sprintf(" AND method = $%d", argCount)
		args = append(args, filter.Method)
		argCount++
	}

	if filter.Path != "" {
		query += fmt.Sprintf(" AND path LIKE $%d", argCount)
		args = append(args, "%"+filter.Path+"%")
		argCount++
	}

	// Add sorting
	if filter.SortBy != "" {
		order := "DESC"
		if filter.SortOrder == "asc" {
			order = "ASC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", filter.SortBy, order)
	} else {
		query += " ORDER BY timestamp DESC"
	}

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search audit logs: %w", err)
	}
	defer rows.Close()

	events := make([]*AuditEvent, 0)
	for rows.Next() {
		event := &AuditEvent{
			Metadata: make(map[string]interface{}),
		}

		var metadataJSON, changesJSON []byte

		err := rows.Scan(
			&event.ID, &event.Timestamp, &event.EventType, &event.Status,
			&event.UserID, &event.Username, &event.OrganizationID, &event.TokenID,
			&event.ResourceType, &event.ResourceID, &event.ResourceName,
			&event.IPAddress, &event.UserAgent, &event.RequestID,
			&event.Method, &event.Path, &event.StatusCode,
			&event.Message, &event.ErrorMessage, &metadataJSON, &changesJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Unmarshal JSON fields
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		if len(changesJSON) > 0 {
			event.Changes = &ChangeDetails{}
			if err := json.Unmarshal(changesJSON, event.Changes); err != nil {
				return nil, fmt.Errorf("failed to unmarshal changes: %w", err)
			}
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return events, nil
}

// GetStats retrieves audit log statistics
func (l *DBLogger) GetStats(ctx context.Context, startTime, endTime *time.Time) (*AuditStats, error) {
	stats := &AuditStats{
		EventsByType:         make(map[EventType]int64),
		EventsByStatus:       make(map[EventStatus]int64),
		EventsByUser:         make(map[int64]int64),
		EventsByOrganization: make(map[int64]int64),
		EventsByResource:     make(map[ResourceType]int64),
	}

	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if startTime != nil {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *startTime)
		argCount++
		if stats.TimeRange == nil {
			stats.TimeRange = &TimeRange{}
		}
		stats.TimeRange.Start = *startTime
	}

	if endTime != nil {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *endTime)
		if stats.TimeRange == nil {
			stats.TimeRange = &TimeRange{}
		}
		stats.TimeRange.End = *endTime
	}

	// Total events
	err := l.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause), args...).Scan(&stats.TotalEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}

	// Events by type
	rows, err := l.db.QueryContext(ctx, fmt.Sprintf("SELECT event_type, COUNT(*) FROM audit_logs %s GROUP BY event_type", whereClause), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventType EventType
		var count int64
		if err := rows.Scan(&eventType, &count); err != nil {
			return nil, err
		}
		stats.EventsByType[eventType] = count
	}

	// Events by status
	rows, err = l.db.QueryContext(ctx, fmt.Sprintf("SELECT status, COUNT(*) FROM audit_logs %s GROUP BY status", whereClause), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status EventStatus
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.EventsByStatus[status] = count
	}

	// Unique users
	err = l.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(DISTINCT user_id) FROM audit_logs %s AND user_id IS NOT NULL", whereClause), args...).Scan(&stats.UniqueUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique users: %w", err)
	}

	// Unique IPs
	err = l.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(DISTINCT ip_address) FROM audit_logs %s AND ip_address IS NOT NULL", whereClause), args...).Scan(&stats.UniqueIPs)
	if err != nil {
		return nil, fmt.Errorf("failed to get unique IPs: %w", err)
	}

	// Failed auth attempts
	failedAuthClause := whereClause + " AND event_type LIKE 'auth.%' AND status = 'failure'"
	err = l.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", failedAuthClause), args...).Scan(&stats.FailedAuthAttempts)
	if err != nil {
		return nil, fmt.Errorf("failed to get failed auth attempts: %w", err)
	}

	// Access denials
	deniedClause := whereClause + " AND status = 'denied'"
	err = l.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", deniedClause), args...).Scan(&stats.AccessDenials)
	if err != nil {
		return nil, fmt.Errorf("failed to get access denials: %w", err)
	}

	return stats, nil
}

// Close closes the database logger
func (l *DBLogger) Close() error {
	// We don't close the database connection as it may be shared
	return nil
}
