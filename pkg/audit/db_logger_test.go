package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	return db, mock
}

func TestNewDBLogger(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		// Expect the table creation query
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

		logger, err := NewDBLogger(db)
		require.NoError(t, err)
		assert.NotNil(t, logger)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nil database", func(t *testing.T) {
		logger, err := NewDBLogger(nil)
		assert.Error(t, err)
		assert.Nil(t, logger)
		assert.Contains(t, err.Error(), "database connection is required")
	})

	t.Run("table creation error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		// Expect the table creation to fail
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnError(errors.New("table creation failed"))

		logger, err := NewDBLogger(db)
		assert.Error(t, err)
		assert.Nil(t, logger)
		assert.Contains(t, err.Error(), "failed to ensure audit_logs table")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_ensureTable(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}

	// Expect the table creation query with indexes
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	err := logger.ensureTable()
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_Log(t *testing.T) {
	t.Run("success - basic event", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		userID := int64(123)
		orgID := int64(456)
		tokenID := int64(789)

		event := &AuditEvent{
			Timestamp:      time.Now().UTC(),
			EventType:      EventTypeAuthLogin,
			Status:         EventStatusSuccess,
			UserID:         &userID,
			Username:       "testuser",
			OrganizationID: &orgID,
			TokenID:        &tokenID,
			ResourceType:   ResourceTypeUser,
			ResourceID:     "user-123",
			ResourceName:   "Test User",
			IPAddress:      "192.168.1.1",
			UserAgent:      "Mozilla/5.0",
			RequestID:      "req-123",
			Method:         "POST",
			Path:           "/api/auth/login",
			StatusCode:     200,
			Message:        "User logged in successfully",
			ErrorMessage:   "",
			Metadata:       map[string]interface{}{"key": "value"},
		}

		// Expect the insert query - use sqlmock.AnyArg() for JSON fields
		mock.ExpectQuery("INSERT INTO audit_logs").
			WithArgs(
				sqlmock.AnyArg(), event.EventType, event.Status,
				event.UserID, event.Username, event.OrganizationID, event.TokenID,
				event.ResourceType, event.ResourceID, event.ResourceName,
				event.IPAddress, event.UserAgent, event.RequestID,
				event.Method, event.Path, event.StatusCode,
				event.Message, event.ErrorMessage, sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.Log(ctx, event)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), event.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - with changes", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		userID := int64(123)

		changes := &ChangeDetails{
			Before: map[string]interface{}{"status": "inactive"},
			After:  map[string]interface{}{"status": "active"},
		}

		event := &AuditEvent{
			Timestamp:    time.Now().UTC(),
			EventType:    EventTypeDataModuleUpdate,
			Status:       EventStatusSuccess,
			UserID:       &userID,
			Username:     "testuser",
			ResourceType: ResourceTypeModule,
			ResourceID:   "module-123",
			Message:      "Module updated",
			Changes:      changes,
			Metadata:     map[string]interface{}{},
		}

		metadataJSON, _ := json.Marshal(event.Metadata)
		changesJSON, _ := json.Marshal(event.Changes)

		mock.ExpectQuery("INSERT INTO audit_logs").
			WithArgs(
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
				sqlmock.AnyArg(), sqlmock.AnyArg(), metadataJSON, changesJSON,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err := logger.Log(ctx, event)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), event.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("metadata marshal error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		event := &AuditEvent{
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			Metadata: map[string]interface{}{
				"invalid": make(chan int), // channels can't be marshaled to JSON
			},
		}

		err := logger.Log(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal metadata")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("changes marshal error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		event := &AuditEvent{
			Timestamp: time.Now().UTC(),
			EventType: EventTypeDataModuleUpdate,
			Status:    EventStatusSuccess,
			Metadata:  map[string]interface{}{},
			Changes: &ChangeDetails{
				Before: map[string]interface{}{
					"invalid": make(chan int), // channels can't be marshaled to JSON
				},
			},
		}

		err := logger.Log(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to marshal changes")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database insert error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		event := &AuditEvent{
			Timestamp: time.Now().UTC(),
			EventType: EventTypeAuthLogin,
			Status:    EventStatusSuccess,
			Metadata:  map[string]interface{}{},
		}

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnError(errors.New("database error"))

		err := logger.Log(ctx, event)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert audit log")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_LogAuthentication(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	ctx := context.Background()
	userID := int64(123)

	mock.ExpectQuery("INSERT INTO audit_logs").
		WithArgs(
			sqlmock.AnyArg(), EventTypeAuthLogin, EventStatusSuccess,
			&userID, "testuser", sqlmock.AnyArg(), sqlmock.AnyArg(),
			ResourceTypeUser, sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"User logged in", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := logger.LogAuthentication(ctx, EventTypeAuthLogin, &userID, "testuser", EventStatusSuccess, "User logged in")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_LogAuthorization(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	ctx := context.Background()
	userID := int64(123)

	mock.ExpectQuery("INSERT INTO audit_logs").
		WithArgs(
			sqlmock.AnyArg(), EventTypeAuthzPermissionCheck, EventStatusSuccess,
			&userID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			ResourceTypeModule, "module-123", sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
			"Permission granted", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := logger.LogAuthorization(ctx, EventTypeAuthzPermissionCheck, &userID, ResourceTypeModule, "module-123", EventStatusSuccess, "Permission granted")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_LogDataMutation(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	ctx := context.Background()
	userID := int64(123)

	changes := &ChangeDetails{
		Before: map[string]interface{}{"name": "old"},
		After:  map[string]interface{}{"name": "new"},
	}

	mock.ExpectQuery("INSERT INTO audit_logs").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := logger.LogDataMutation(ctx, EventTypeDataModuleUpdate, &userID, ResourceTypeModule, "module-123", changes, "Module updated")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_LogConfiguration(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	ctx := context.Background()
	userID := int64(123)

	changes := &ChangeDetails{
		Before: map[string]interface{}{"enabled": false},
		After:  map[string]interface{}{"enabled": true},
	}

	mock.ExpectQuery("INSERT INTO audit_logs").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := logger.LogConfiguration(ctx, EventTypeConfigChange, &userID, "config-123", changes, "Configuration updated")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_LogAdminAction(t *testing.T) {
	t.Run("with target user", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		adminUserID := int64(123)
		targetUserID := int64(456)

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.LogAdminAction(ctx, EventTypeAdminUserUpdate, &adminUserID, &targetUserID, "User updated by admin")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("without target user", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		adminUserID := int64(123)

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.LogAdminAction(ctx, EventTypeAdminOrgCreate, &adminUserID, nil, "Organization created by admin")
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_LogAccess(t *testing.T) {
	db, mock := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	ctx := context.Background()
	userID := int64(123)

	mock.ExpectQuery("INSERT INTO audit_logs").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err := logger.LogAccess(ctx, EventTypeAccessModuleRead, &userID, ResourceTypeModule, "module-123", "Module accessed")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDBLogger_LogHTTPRequest(t *testing.T) {
	t.Run("success request", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		req := httptest.NewRequest("GET", "/api/modules", nil)
		duration := 150 * time.Millisecond

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.LogHTTPRequest(ctx, req, 200, duration, nil)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failure request", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		req := httptest.NewRequest("POST", "/api/modules", nil)
		duration := 50 * time.Millisecond
		requestError := errors.New("internal server error")

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.LogHTTPRequest(ctx, req, 500, duration, requestError)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("denied request", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()
		req := httptest.NewRequest("DELETE", "/api/modules/123", nil)
		duration := 10 * time.Millisecond

		mock.ExpectQuery("INSERT INTO audit_logs").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := logger.LogHTTPRequest(ctx, req, 403, duration, nil)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_Search(t *testing.T) {
	t.Run("no filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{
			"id", "timestamp", "event_type", "status",
			"user_id", "username", "organization_id", "token_id",
			"resource_type", "resource_id", "resource_name",
			"ip_address", "user_agent", "request_id",
			"method", "path", "status_code",
			"message", "error_message", "metadata", "changes",
		}).AddRow(
			1, time.Now(), EventTypeAuthLogin, EventStatusSuccess,
			int64(123), "testuser", int64(456), int64(789),
			ResourceTypeUser, "user-123", "Test User",
			"192.168.1.1", "Mozilla/5.0", "req-123",
			"POST", "/api/auth/login", 200,
			"Login successful", "", []byte("{}"), nil,
		)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 ORDER BY timestamp DESC").
			WillReturnRows(rows)

		events, err := logger.Search(ctx, SearchFilter{})
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.Equal(t, int64(1), events[0].ID)
		assert.Equal(t, EventTypeAuthLogin, events[0].EventType)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with time filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		startTime := time.Now().Add(-24 * time.Hour)
		endTime := time.Now()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with user filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		userID := int64(123)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND user_id = \\$1").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			UserID: &userID,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with username filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND username = \\$1").
			WithArgs("testuser").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			Username: "testuser",
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with organization filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		orgID := int64(456)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND organization_id = \\$1").
			WithArgs(orgID).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			OrganizationID: &orgID,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with event types filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		eventTypes := []EventType{EventTypeAuthLogin, EventTypeAuthLogout}

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND event_type = ANY\\(\\$1\\)").
			WithArgs(pq.Array([]string{string(EventTypeAuthLogin), string(EventTypeAuthLogout)})).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			EventTypes: eventTypes,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with status filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		status := EventStatusFailure

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND status = \\$1").
			WithArgs(string(status)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			Status: &status,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with resource filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND resource_type = \\$1 AND resource_id = \\$2").
			WithArgs(string(ResourceTypeModule), "module-123").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			ResourceType: ResourceTypeModule,
			ResourceID:   "module-123",
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with IP and method filters", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND ip_address = \\$1 AND method = \\$2").
			WithArgs("192.168.1.1", "POST").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			IPAddress: "192.168.1.1",
			Method:    "POST",
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with path filter", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 AND path LIKE \\$1").
			WithArgs("%/api/modules%").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			Path: "/api/modules",
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with custom sorting", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 ORDER BY event_type ASC").
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			SortBy:    "event_type",
			SortOrder: "asc",
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with pagination", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 ORDER BY timestamp DESC LIMIT \\$1 OFFSET \\$2").
			WithArgs(10, 20).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "timestamp", "event_type", "status",
				"user_id", "username", "organization_id", "token_id",
				"resource_type", "resource_id", "resource_name",
				"ip_address", "user_agent", "request_id",
				"method", "path", "status_code",
				"message", "error_message", "metadata", "changes",
			}))

		filter := SearchFilter{
			Limit:  10,
			Offset: 20,
		}

		events, err := logger.Search(ctx, filter)
		assert.NoError(t, err)
		assert.NotNil(t, events)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("with changes data", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		changesData := ChangeDetails{
			Before: map[string]interface{}{"status": "inactive"},
			After:  map[string]interface{}{"status": "active"},
		}
		changesJSON, _ := json.Marshal(changesData)

		rows := sqlmock.NewRows([]string{
			"id", "timestamp", "event_type", "status",
			"user_id", "username", "organization_id", "token_id",
			"resource_type", "resource_id", "resource_name",
			"ip_address", "user_agent", "request_id",
			"method", "path", "status_code",
			"message", "error_message", "metadata", "changes",
		}).AddRow(
			1, time.Now(), EventTypeDataModuleUpdate, EventStatusSuccess,
			int64(123), "testuser", int64(456), int64(789),
			ResourceTypeModule, "module-123", "Test Module",
			"192.168.1.1", "Mozilla/5.0", "req-123",
			"PUT", "/api/modules/123", 200,
			"Module updated", "", []byte("{}"), changesJSON,
		)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1 ORDER BY timestamp DESC").
			WillReturnRows(rows)

		events, err := logger.Search(ctx, SearchFilter{})
		assert.NoError(t, err)
		assert.Len(t, events, 1)
		assert.NotNil(t, events[0].Changes)
		assert.Equal(t, "inactive", events[0].Changes.Before["status"])
		assert.Equal(t, "active", events[0].Changes.After["status"])
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1").
			WillReturnError(errors.New("database error"))

		events, err := logger.Search(ctx, SearchFilter{})
		assert.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "failed to search audit logs")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{"id"}).AddRow(1)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1").
			WillReturnRows(rows)

		events, err := logger.Search(ctx, SearchFilter{})
		assert.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "failed to scan audit log")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("metadata unmarshal error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{
			"id", "timestamp", "event_type", "status",
			"user_id", "username", "organization_id", "token_id",
			"resource_type", "resource_id", "resource_name",
			"ip_address", "user_agent", "request_id",
			"method", "path", "status_code",
			"message", "error_message", "metadata", "changes",
		}).AddRow(
			1, time.Now(), EventTypeAuthLogin, EventStatusSuccess,
			int64(123), "testuser", int64(456), int64(789),
			ResourceTypeUser, "user-123", "Test User",
			"192.168.1.1", "Mozilla/5.0", "req-123",
			"POST", "/api/auth/login", 200,
			"Login successful", "", []byte("invalid json"), nil,
		)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1").
			WillReturnRows(rows)

		events, err := logger.Search(ctx, SearchFilter{})
		assert.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "failed to unmarshal metadata")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("changes unmarshal error", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		rows := sqlmock.NewRows([]string{
			"id", "timestamp", "event_type", "status",
			"user_id", "username", "organization_id", "token_id",
			"resource_type", "resource_id", "resource_name",
			"ip_address", "user_agent", "request_id",
			"method", "path", "status_code",
			"message", "error_message", "metadata", "changes",
		}).AddRow(
			1, time.Now(), EventTypeDataModuleUpdate, EventStatusSuccess,
			int64(123), "testuser", int64(456), int64(789),
			ResourceTypeModule, "module-123", "Test Module",
			"192.168.1.1", "Mozilla/5.0", "req-123",
			"PUT", "/api/modules/123", 200,
			"Module updated", "", []byte("{}"), []byte("invalid json"),
		)

		mock.ExpectQuery("SELECT (.+) FROM audit_logs WHERE 1=1").
			WillReturnRows(rows)

		events, err := logger.Search(ctx, SearchFilter{})
		assert.Error(t, err)
		assert.Nil(t, events)
		assert.Contains(t, err.Error(), "failed to unmarshal changes")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_GetStats(t *testing.T) {
	t.Run("success - no time range", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		// Total events
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		// Events by type
		mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY event_type").
			WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}).
				AddRow(EventTypeAuthLogin, 50).
				AddRow(EventTypeAuthLogout, 30))

		// Events by status
		mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY status").
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
				AddRow(EventStatusSuccess, 80).
				AddRow(EventStatusFailure, 20))

		// Unique users
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\) FROM audit_logs WHERE 1=1 AND user_id IS NOT NULL").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(25))

		// Unique IPs
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT ip_address\\) FROM audit_logs WHERE 1=1 AND ip_address IS NOT NULL").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(40))

		// Failed auth attempts
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND event_type LIKE 'auth.%' AND status = 'failure'").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

		// Access denials
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND status = 'denied'").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		stats, err := logger.GetStats(ctx, nil, nil)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(100), stats.TotalEvents)
		assert.Equal(t, int64(50), stats.EventsByType[EventTypeAuthLogin])
		assert.Equal(t, int64(30), stats.EventsByType[EventTypeAuthLogout])
		assert.Equal(t, int64(80), stats.EventsByStatus[EventStatusSuccess])
		assert.Equal(t, int64(20), stats.EventsByStatus[EventStatusFailure])
		assert.Equal(t, int64(25), stats.UniqueUsers)
		assert.Equal(t, int64(40), stats.UniqueIPs)
		assert.Equal(t, int64(10), stats.FailedAuthAttempts)
		assert.Equal(t, int64(5), stats.AccessDenials)
		assert.Nil(t, stats.TimeRange)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success - with time range", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		startTime := time.Now().Add(-24 * time.Hour)
		endTime := time.Now()

		// Total events
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(50))

		// Events by type
		mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 GROUP BY event_type").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}).
				AddRow(EventTypeAuthLogin, 30))

		// Events by status
		mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 GROUP BY status").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
				AddRow(EventStatusSuccess, 45))

		// Unique users
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 AND user_id IS NOT NULL").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(15))

		// Unique IPs
		mock.ExpectQuery("SELECT COUNT\\(DISTINCT ip_address\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 AND ip_address IS NOT NULL").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(20))

		// Failed auth attempts
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 AND event_type LIKE 'auth.%' AND status = 'failure'").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

		// Access denials
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 AND timestamp >= \\$1 AND timestamp <= \\$2 AND status = 'denied'").
			WithArgs(startTime, endTime).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

		stats, err := logger.GetStats(ctx, &startTime, &endTime)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(50), stats.TotalEvents)
		assert.NotNil(t, stats.TimeRange)
		assert.Equal(t, startTime, stats.TimeRange.Start)
		assert.Equal(t, endTime, stats.TimeRange.End)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - total events query fails", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1").
			WillReturnError(errors.New("database error"))

		stats, err := logger.GetStats(ctx, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to get total events")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - events by type query fails", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY event_type").
			WillReturnError(errors.New("database error"))

		stats, err := logger.GetStats(ctx, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to get events by type")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - events by status query fails", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY event_type").
			WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}))

		mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY status").
			WillReturnError(errors.New("database error"))

		stats, err := logger.GetStats(ctx, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to get events by status")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("error - unique users query fails", func(t *testing.T) {
		db, mock := setupMockDB(t)
		defer db.Close()

		logger := &DBLogger{db: db}
		ctx := context.Background()

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs WHERE 1=1").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

		mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY event_type").
			WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}))

		mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) FROM audit_logs WHERE 1=1 GROUP BY status").
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}))

		mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\) FROM audit_logs WHERE 1=1 AND user_id IS NOT NULL").
			WillReturnError(errors.New("database error"))

		stats, err := logger.GetStats(ctx, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, stats)
		assert.Contains(t, err.Error(), "failed to get unique users")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestDBLogger_Close(t *testing.T) {
	db, _ := setupMockDB(t)
	defer db.Close()

	logger := &DBLogger{db: db}
	err := logger.Close()
	assert.NoError(t, err)
}
