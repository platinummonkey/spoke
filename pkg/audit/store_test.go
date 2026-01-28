package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreNewDBStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect the table creation queries
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	assert.NotNil(t, store)
	assert.Equal(t, logger, store.logger)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Search(t *testing.T) {
	ctx := context.Background()
	userID := int64(123)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	// Mock the search query
	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		int64(1), time.Now().UTC(), EventTypeAuthLogin, EventStatusSuccess,
		userID, "testuser", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	filter := SearchFilter{
		UserID: &userID,
		Limit:  10,
	}

	events, err := store.Search(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, int64(1), events[0].ID)
	assert.Equal(t, "testuser", events[0].Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Search_Error(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	expectedError := errors.New("database error")
	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnError(expectedError)

	filter := SearchFilter{Limit: 10}

	events, err := store.Search(ctx, filter)
	assert.Error(t, err)
	assert.Nil(t, events)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Get(t *testing.T) {
	ctx := context.Background()
	targetID := int64(42)
	userID := int64(123)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	// Mock the search query for Get
	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		targetID, time.Now().UTC(), EventTypeAuthLogin, EventStatusSuccess,
		userID, "testuser", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	event, err := store.Get(ctx, targetID)

	require.NoError(t, err)
	require.NotNil(t, event)
	assert.Equal(t, targetID, event.ID)
	assert.Equal(t, "testuser", event.Username)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Get_NotFound(t *testing.T) {
	ctx := context.Background()
	targetID := int64(42)
	differentID := int64(99)
	userID := int64(123)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	// Return a different ID
	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		differentID, time.Now().UTC(), EventTypeAuthLogin, EventStatusSuccess,
		userID, "testuser", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	event, err := store.Get(ctx, targetID)

	require.NoError(t, err)
	assert.Nil(t, event)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Get_EmptyResults(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	// Return empty results
	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	})

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	event, err := store.Get(ctx, int64(1))

	require.NoError(t, err)
	assert.Nil(t, event)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Get_Error(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	expectedError := errors.New("search error")
	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnError(expectedError)

	event, err := store.Get(ctx, int64(1))

	assert.Error(t, err)
	assert.Nil(t, event)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_GetStats(t *testing.T) {
	ctx := context.Background()
	startTime := time.Now().Add(-24 * time.Hour)
	endTime := time.Now()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	// Mock total events query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock events by type query
	mock.ExpectQuery("SELECT event_type, COUNT\\(\\*\\) FROM audit_logs").
		WillReturnRows(sqlmock.NewRows([]string{"event_type", "count"}).
			AddRow(EventTypeAuthLogin, 50).
			AddRow(EventTypeAuthLogout, 50))

	// Mock events by status query
	mock.ExpectQuery("SELECT status, COUNT\\(\\*\\) FROM audit_logs").
		WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).
			AddRow(EventStatusSuccess, 90).
			AddRow(EventStatusFailure, 10))

	// Mock unique users query
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\) FROM audit_logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))

	// Mock unique IPs query
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT ip_address\\) FROM audit_logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Mock failed auth attempts query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	// Mock access denials query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	stats, err := store.GetStats(ctx, &startTime, &endTime)

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, int64(100), stats.TotalEvents)
	assert.Equal(t, int64(10), stats.UniqueUsers)
	assert.Equal(t, int64(5), stats.UniqueIPs)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_GetStats_Error(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	expectedError := errors.New("stats error")
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM audit_logs").WillReturnError(expectedError)

	stats, err := store.GetStats(ctx, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, stats)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Export_JSON(t *testing.T) {
	ctx := context.Background()
	userID := int64(123)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		int64(1), time.Now().UTC(), EventTypeAuthLogin, EventStatusSuccess,
		userID, "testuser", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	data, err := store.Export(ctx, SearchFilter{}, ExportFormatJSON)

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "testuser")
	assert.Contains(t, string(data), "auth.login")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Export_CSV(t *testing.T) {
	ctx := context.Background()
	userID := int64(456)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		int64(2), time.Now().UTC(), EventTypeAuthLogout, EventStatusSuccess,
		userID, "user2", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	data, err := store.Export(ctx, SearchFilter{}, ExportFormatCSV)

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Export_NDJSON(t *testing.T) {
	ctx := context.Background()
	userID := int64(789)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		int64(3), time.Now().UTC(), EventTypeDataModuleCreate, EventStatusSuccess,
		userID, "user3", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	data, err := store.Export(ctx, SearchFilter{}, ExportFormatNDJSON)

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Export_DefaultFormat(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	rows := sqlmock.NewRows([]string{
		"id", "timestamp", "event_type", "status",
		"user_id", "username", "organization_id", "token_id",
		"resource_type", "resource_id", "resource_name",
		"ip_address", "user_agent", "request_id",
		"method", "path", "status_code",
		"message", "error_message", "metadata", "changes",
	}).AddRow(
		int64(4), time.Now().UTC(), EventTypeAuthLogin, EventStatusSuccess,
		nil, "", nil, nil,
		"", "", "",
		"", "", "",
		"", "", 0,
		"", "", []byte("{}"), nil,
	)

	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnRows(rows)

	data, err := store.Export(ctx, SearchFilter{}, ExportFormat("unknown"))

	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Export_Error(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	store := NewDBStore(logger)

	expectedError := errors.New("export error")
	mock.ExpectQuery("SELECT (.+) FROM audit_logs").WillReturnError(expectedError)

	data, err := store.Export(ctx, SearchFilter{}, ExportFormatJSON)

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Cleanup(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	policy := RetentionPolicy{
		RetentionDays:  30,
		ArchiveEnabled: false,
	}

	mock.ExpectExec("DELETE FROM audit_logs WHERE timestamp < \\$1").
		WillReturnResult(sqlmock.NewResult(0, 10))

	store := NewDBStore(logger)

	count, err := store.Cleanup(ctx, policy)

	require.NoError(t, err)
	assert.Equal(t, int64(10), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Cleanup_WithArchiving(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	policy := RetentionPolicy{
		RetentionDays:   90,
		ArchiveEnabled:  true,
		ArchivePath:     "/tmp/archive",
		CompressArchive: true,
	}

	mock.ExpectExec("DELETE FROM audit_logs WHERE timestamp < \\$1").
		WillReturnResult(sqlmock.NewResult(0, 25))

	store := NewDBStore(logger)

	count, err := store.Cleanup(ctx, policy)

	require.NoError(t, err)
	assert.Equal(t, int64(25), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Cleanup_Error(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	policy := RetentionPolicy{
		RetentionDays: 30,
	}

	expectedError := errors.New("cleanup error")
	mock.ExpectExec("DELETE FROM audit_logs WHERE timestamp < \\$1").
		WillReturnError(expectedError)

	store := NewDBStore(logger)

	count, err := store.Cleanup(ctx, policy)

	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStore_Cleanup_RowsAffectedError(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Expect table creation
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS audit_logs").WillReturnResult(sqlmock.NewResult(0, 0))

	logger, err := NewDBLogger(db)
	require.NoError(t, err)

	policy := RetentionPolicy{
		RetentionDays: 30,
	}

	mock.ExpectExec("DELETE FROM audit_logs WHERE timestamp < \\$1").
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	store := NewDBStore(logger)

	count, err := store.Cleanup(ctx, policy)

	assert.Error(t, err)
	assert.Equal(t, int64(0), count)
	assert.NoError(t, mock.ExpectationsWereMet())
}
