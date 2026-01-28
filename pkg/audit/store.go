package audit

import (
	"context"
	"time"
)

// Store provides methods for querying and managing audit logs
type Store interface {
	// Search searches audit logs based on filters
	Search(ctx context.Context, filter SearchFilter) ([]*AuditEvent, error)

	// Get retrieves a specific audit event by ID
	Get(ctx context.Context, id int64) (*AuditEvent, error)

	// GetStats retrieves audit log statistics
	GetStats(ctx context.Context, startTime, endTime *time.Time) (*AuditStats, error)

	// Export exports audit logs in the specified format
	Export(ctx context.Context, filter SearchFilter, format ExportFormat) ([]byte, error)

	// Cleanup removes audit logs older than the retention period
	Cleanup(ctx context.Context, policy RetentionPolicy) (int64, error)
}

// DBStore implements Store interface using PostgreSQL
type DBStore struct {
	logger *DBLogger
}

// NewDBStore creates a new database-backed audit store
func NewDBStore(logger *DBLogger) *DBStore {
	return &DBStore{
		logger: logger,
	}
}

// Search searches audit logs based on filters
func (s *DBStore) Search(ctx context.Context, filter SearchFilter) ([]*AuditEvent, error) {
	return s.logger.Search(ctx, filter)
}

// Get retrieves a specific audit event by ID
func (s *DBStore) Get(ctx context.Context, id int64) (*AuditEvent, error) {
	events, err := s.logger.Search(ctx, SearchFilter{
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}

	if len(events) == 0 {
		return nil, nil
	}

	// Filter by ID in memory (or we could add ID to SearchFilter)
	for _, event := range events {
		if event.ID == id {
			return event, nil
		}
	}

	return nil, nil
}

// GetStats retrieves audit log statistics
func (s *DBStore) GetStats(ctx context.Context, startTime, endTime *time.Time) (*AuditStats, error) {
	return s.logger.GetStats(ctx, startTime, endTime)
}

// Export exports audit logs in the specified format
func (s *DBStore) Export(ctx context.Context, filter SearchFilter, format ExportFormat) ([]byte, error) {
	// Get all events matching the filter
	events, err := s.logger.Search(ctx, filter)
	if err != nil {
		return nil, err
	}

	switch format {
	case ExportFormatJSON:
		return exportJSON(events)
	case ExportFormatCSV:
		return exportCSV(events)
	case ExportFormatNDJSON:
		return exportNDJSON(events)
	default:
		return exportJSON(events)
	}
}

// Cleanup removes audit logs older than the retention period
func (s *DBStore) Cleanup(ctx context.Context, policy RetentionPolicy) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -policy.RetentionDays)

	// If archiving is enabled, we'd export and save the logs first
	//nolint:staticcheck // SA9003: Empty branch is placeholder for future archiving feature
	if policy.ArchiveEnabled {
		// TODO: Implement archiving
	}

	// Delete old logs
	result, err := s.logger.db.ExecContext(ctx, "DELETE FROM audit_logs WHERE timestamp < $1", cutoffDate)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
