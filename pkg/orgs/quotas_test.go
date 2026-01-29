package orgs

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckModuleQuota_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		5, 50, int64(1024*1024*1024), 1000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	err = service.CheckModuleQuota(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckModuleQuota_QuotaExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage - modules at limit
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		10, 50, int64(1024*1024*1024), 1000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	err = service.CheckModuleQuota(123)
	assert.Error(t, err)
	assert.True(t, IsQuotaExceeded(err))

	quotaErr, ok := err.(*QuotaExceededError)
	require.True(t, ok)
	assert.Equal(t, "modules", quotaErr.Resource)
	assert.Equal(t, int64(10), quotaErr.Current)
	assert.Equal(t, int64(10), quotaErr.Limit)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckModuleQuota_GetQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckModuleQuota(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get quotas")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckModuleQuota_GetUsageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckModuleQuota(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get usage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckVersionQuota_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock version count query
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(50)
	mock.ExpectQuery("SELECT COUNT(.+) FROM versions v JOIN modules m").
		WithArgs(int64(123), "test-module").
		WillReturnRows(countRow)

	err = service.CheckVersionQuota(123, "test-module")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckVersionQuota_QuotaExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock version count query - at limit
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(100)
	mock.ExpectQuery("SELECT COUNT(.+) FROM versions v JOIN modules m").
		WithArgs(int64(123), "test-module").
		WillReturnRows(countRow)

	err = service.CheckVersionQuota(123, "test-module")
	assert.Error(t, err)
	assert.True(t, IsQuotaExceeded(err))

	quotaErr, ok := err.(*QuotaExceededError)
	require.True(t, ok)
	assert.Equal(t, "versions", quotaErr.Resource)
	assert.Equal(t, int64(100), quotaErr.Current)
	assert.Equal(t, int64(100), quotaErr.Limit)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckVersionQuota_GetQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckVersionQuota(123, "test-module")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get quotas")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckVersionQuota_CountError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	mock.ExpectQuery("SELECT COUNT(.+) FROM versions v JOIN modules m").
		WithArgs(int64(123), "test-module").
		WillReturnError(errors.New("count error"))

	err = service.CheckVersionQuota(123, "test-module")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count versions")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckStorageQuota_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		5, 50, int64(1024*1024*1024), 1000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	err = service.CheckStorageQuota(123, int64(1024*1024*1024))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckStorageQuota_QuotaExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage - storage near limit
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		5, 50, int64(4*1024*1024*1024), 1000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	// Try to add 2GB when only 1GB is available
	err = service.CheckStorageQuota(123, int64(2*1024*1024*1024))
	assert.Error(t, err)
	assert.True(t, IsQuotaExceeded(err))

	quotaErr, ok := err.(*QuotaExceededError)
	require.True(t, ok)
	assert.Equal(t, "storage", quotaErr.Resource)
	assert.Equal(t, int64(6*1024*1024*1024), quotaErr.Current)
	assert.Equal(t, int64(5*1024*1024*1024), quotaErr.Limit)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckStorageQuota_GetQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckStorageQuota(123, int64(1024*1024))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get quotas")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCompileJobQuota_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		5, 50, int64(1024*1024*1024), 1000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	err = service.CheckCompileJobQuota(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCompileJobQuota_QuotaExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock GetUsage - compile jobs at limit
	usageRow := sqlmock.NewRows([]string{
		"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes",
		"compile_jobs_count", "api_requests_count", "created_at", "updated_at",
	}).AddRow(
		1, 123, time.Now(), time.Now().AddDate(0, 1, 0),
		5, 50, int64(1024*1024*1024), 5000, int64(1000), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(usageRow)

	err = service.CheckCompileJobQuota(123)
	assert.Error(t, err)
	assert.True(t, IsQuotaExceeded(err))

	quotaErr, ok := err.(*QuotaExceededError)
	require.True(t, ok)
	assert.Equal(t, "compile_jobs", quotaErr.Resource)
	assert.Equal(t, int64(5000), quotaErr.Current)
	assert.Equal(t, int64(5000), quotaErr.Limit)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAPIRateLimit_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock API request count query
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(1000)
	mock.ExpectQuery("SELECT COUNT(.+) FROM audit_logs WHERE organization_id").
		WithArgs(int64(123)).
		WillReturnRows(countRow)

	err = service.CheckAPIRateLimit(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAPIRateLimit_QuotaExceeded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock API request count query - at limit
	countRow := sqlmock.NewRows([]string{"count"}).AddRow(5000)
	mock.ExpectQuery("SELECT COUNT(.+) FROM audit_logs WHERE organization_id").
		WithArgs(int64(123)).
		WillReturnRows(countRow)

	err = service.CheckAPIRateLimit(123)
	assert.Error(t, err)
	assert.True(t, IsQuotaExceeded(err))

	quotaErr, ok := err.(*QuotaExceededError)
	require.True(t, ok)
	assert.Equal(t, "api_requests", quotaErr.Resource)
	assert.Equal(t, int64(5000), quotaErr.Current)
	assert.Equal(t, int64(5000), quotaErr.Limit)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAPIRateLimit_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock API request count query - ErrNoRows should be handled
	mock.ExpectQuery("SELECT COUNT(.+) FROM audit_logs WHERE organization_id").
		WithArgs(int64(123)).
		WillReturnError(sql.ErrNoRows)

	err = service.CheckAPIRateLimit(123)
	assert.NoError(t, err) // ErrNoRows should be ignored
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementModules_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementModules(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementModules_InitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second update succeeds
	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementModules(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementModules_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.IncrementModules(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment modules")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementVersions_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementVersions(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementVersions_InitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second update succeeds
	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementVersions(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementStorage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementStorage(123, int64(1024*1024))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementStorage_InitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second update succeeds
	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementStorage(123, int64(1024*1024))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementCompileJobs_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementCompileJobs(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementCompileJobs_InitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second update succeeds
	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementCompileJobs(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementAPIRequests_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementAPIRequests(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementAPIRequests_InitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second update succeeds
	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.IncrementAPIRequests(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementModules_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET modules_count = GREATEST").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.DecrementModules(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementModules_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET modules_count = GREATEST").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.DecrementModules(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement modules")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementVersions_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET versions_count = GREATEST").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.DecrementVersions(123)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementVersions_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET versions_count = GREATEST").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.DecrementVersions(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement versions")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementStorage_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET storage_bytes = GREATEST").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.DecrementStorage(123, int64(1024*1024))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDecrementStorage_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET storage_bytes = GREATEST").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.DecrementStorage(123, int64(1024*1024))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decrement storage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementModules_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err = service.IncrementModules(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementVersions_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err = service.IncrementVersions(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementStorage_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err = service.IncrementStorage(123, int64(1024*1024))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementCompileJobs_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err = service.IncrementCompileJobs(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementAPIRequests_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))

	err = service.IncrementAPIRequests(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get rows affected")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementModules_InitializeUsagePeriodError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET modules_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period fails
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	err = service.IncrementModules(123)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAPIRateLimit_GetQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckAPIRateLimit(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get quotas")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckAPIRateLimit_CountError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	// Mock API request count query with error (not ErrNoRows)
	mock.ExpectQuery("SELECT COUNT(.+) FROM audit_logs WHERE organization_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("count error"))

	err = service.CheckAPIRateLimit(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to count API requests")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckStorageQuota_GetUsageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckStorageQuota(123, int64(1024*1024))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get usage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCompileJobQuota_GetQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckCompileJobQuota(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get quotas")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckCompileJobQuota_GetUsageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// Mock GetQuotas
	quotasRow := sqlmock.NewRows([]string{
		"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at",
	}).AddRow(
		1, 123, 10, 100, int64(5*1024*1024*1024), 5000, 5000,
		[]byte("{}"), time.Now(), time.Now(),
	)
	mock.ExpectQuery("SELECT (.+) FROM org_quotas WHERE org_id").
		WithArgs(int64(123)).
		WillReturnRows(quotasRow)

	mock.ExpectQuery("SELECT (.+) FROM org_usage WHERE org_id").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.CheckCompileJobQuota(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get usage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementVersions_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.IncrementVersions(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment versions")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementStorage_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.IncrementStorage(123, int64(1024*1024))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment storage")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementCompileJobs_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.IncrementCompileJobs(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment compile jobs")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementAPIRequests_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnError(errors.New("database error"))

	err = service.IncrementAPIRequests(123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment API requests")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementVersions_InitializeUsagePeriodError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET versions_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period fails
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	err = service.IncrementVersions(123)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementStorage_InitializeUsagePeriodError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET storage_bytes").
		WithArgs(int64(1024*1024), int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period fails
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	err = service.IncrementStorage(123, int64(1024*1024))
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementCompileJobs_InitializeUsagePeriodError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET compile_jobs_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period fails
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	err = service.IncrementCompileJobs(123)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIncrementAPIRequests_InitializeUsagePeriodError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := &PostgresService{db: db}

	// First update returns 0 rows
	mock.ExpectExec("UPDATE org_usage SET api_requests_count").
		WithArgs(int64(123)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Initialize usage period fails
	mock.ExpectExec("INSERT INTO org_usage").
		WithArgs(int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(errors.New("insert error"))

	err = service.IncrementAPIRequests(123)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
