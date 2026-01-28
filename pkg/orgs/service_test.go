package orgs

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "MyOrg",
			expected: "myorg",
		},
		{
			name:     "name with spaces",
			input:    "My Organization",
			expected: "my-organization",
		},
		{
			name:     "name with special chars",
			input:    "My-Org-123",
			expected: "my-org-123",
		},
		{
			name:     "name with invalid chars",
			input:    "My@Org!",
			expected: "myorg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token1)
	assert.Equal(t, 64, len(token1)) // 32 bytes = 64 hex chars

	token2, err := generateToken()
	require.NoError(t, err)
	assert.NotEmpty(t, token2)
	assert.NotEqual(t, token1, token2) // Should be unique
}

func TestQuotaExceededError(t *testing.T) {
	err := &QuotaExceededError{
		Resource: "modules",
		Current:  10,
		Limit:    5,
	}

	assert.True(t, IsQuotaExceeded(err))
	assert.Contains(t, err.Error(), "quota exceeded")
	assert.Contains(t, err.Error(), "modules")
}

func TestGetDefaultQuotas(t *testing.T) {
	service := &PostgresService{}

	tests := []struct {
		name     string
		tier     QuotaTier
		expected *OrgQuotas
	}{
		{
			name: "small tier",
			tier: QuotaTierSmall,
			expected: &OrgQuotas{
				MaxModules:             10,
				MaxVersionsPerModule:   100,
				MaxStorageBytes:        5 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 5000,
				APIRateLimitPerHour:    5000,
			},
		},
		{
			name: "medium tier",
			tier: QuotaTierMedium,
			expected: &OrgQuotas{
				MaxModules:             50,
				MaxVersionsPerModule:   500,
				MaxStorageBytes:        25 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 25000,
				APIRateLimitPerHour:    25000,
			},
		},
		{
			name: "large tier",
			tier: QuotaTierLarge,
			expected: &OrgQuotas{
				MaxModules:             200,
				MaxVersionsPerModule:   2000,
				MaxStorageBytes:        100 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 100000,
				APIRateLimitPerHour:    100000,
			},
		},
		{
			name: "unlimited tier",
			tier: QuotaTierUnlimited,
			expected: &OrgQuotas{
				MaxModules:             999999,
				MaxVersionsPerModule:   999999,
				MaxStorageBytes:        999999 * 1024 * 1024 * 1024,
				MaxCompileJobsPerMonth: 999999999,
				APIRateLimitPerHour:    999999999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quotas := service.GetDefaultQuotas(tt.tier)
			assert.Equal(t, tt.expected.MaxModules, quotas.MaxModules)
			assert.Equal(t, tt.expected.MaxVersionsPerModule, quotas.MaxVersionsPerModule)
			assert.Equal(t, tt.expected.MaxStorageBytes, quotas.MaxStorageBytes)
			assert.Equal(t, tt.expected.MaxCompileJobsPerMonth, quotas.MaxCompileJobsPerMonth)
			assert.Equal(t, tt.expected.APIRateLimitPerHour, quotas.APIRateLimitPerHour)
		})
	}
}

func TestOrgStatuses(t *testing.T) {
	assert.Equal(t, OrgStatus("active"), OrgStatusActive)
	assert.Equal(t, OrgStatus("suspended"), OrgStatusSuspended)
	assert.Equal(t, OrgStatus("deleted"), OrgStatusDeleted)
}

func TestQuotaTiers(t *testing.T) {
	assert.Equal(t, QuotaTier("small"), QuotaTierSmall)
	assert.Equal(t, QuotaTier("medium"), QuotaTierMedium)
	assert.Equal(t, QuotaTier("large"), QuotaTierLarge)
	assert.Equal(t, QuotaTier("unlimited"), QuotaTierUnlimited)
}

func TestOrgInvitationExpiry(t *testing.T) {
	now := time.Now()
	invitation := &OrgInvitation{
		InvitedAt: now,
		ExpiresAt: now.Add(7 * 24 * time.Hour),
	}

	assert.True(t, invitation.ExpiresAt.After(now))
	assert.True(t, invitation.ExpiresAt.Before(now.Add(8 * 24 * time.Hour)))
}

// Database service tests with sqlmock

func TestNewPostgresService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
}

func TestCreateOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	tests := []struct {
		name    string
		org     *Organization
		setup   func()
		wantErr bool
	}{
		{
			name: "successful creation with defaults",
			org: &Organization{
				Name:        "Test Org",
				DisplayName: "Test Organization",
				Description: "A test organization",
				OwnerID:     &ownerID,
			},
			setup: func() {
				settingsJSON, _ := json.Marshal(map[string]any(nil))
				mock.ExpectQuery(`INSERT INTO organizations`).
					WithArgs("Test Org", "test-org", "Test Organization", "A test organization",
						ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, now, now))

				// Mock for createQuotas
				customSettingsJSON, _ := json.Marshal(map[string]any(nil))
				mock.ExpectQuery(`INSERT INTO org_quotas`).
					WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(1, now, now))

				// Mock for initializeUsagePeriod
				mock.ExpectExec(`INSERT INTO org_usage`).
					WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name: "creation with custom slug and tier",
			org: &Organization{
				Name:        "My Custom Org",
				Slug:        "my-custom-slug",
				DisplayName: "Custom Org",
				QuotaTier:   QuotaTierMedium,
				OwnerID:     &ownerID,
			},
			setup: func() {
				settingsJSON, _ := json.Marshal(map[string]any(nil))
				mock.ExpectQuery(`INSERT INTO organizations`).
					WithArgs("My Custom Org", "my-custom-slug", "Custom Org", "",
						ownerID, QuotaTierMedium, OrgStatusActive, true, settingsJSON).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(2, now, now))

				customSettingsJSON, _ := json.Marshal(map[string]any(nil))
				mock.ExpectQuery(`INSERT INTO org_quotas`).
					WithArgs(int64(2), 50, 500, int64(25*1024*1024*1024), 25000, 25000, customSettingsJSON).
					WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
						AddRow(2, now, now))

				mock.ExpectExec(`INSERT INTO org_usage`).
					WithArgs(int64(2), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(2, 1))
			},
			wantErr: false,
		},
		{
			name: "database error on insert",
			org: &Organization{
				Name:        "Error Org",
				DisplayName: "Error Org",
				OwnerID:     &ownerID,
			},
			setup: func() {
				settingsJSON, _ := json.Marshal(map[string]any(nil))
				mock.ExpectQuery(`INSERT INTO organizations`).
					WithArgs("Error Org", "error-org", "Error Org", "",
						ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.CreateOrganization(tt.org)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.org.ID)
				assert.Contains(t, []QuotaTier{QuotaTierSmall, QuotaTierMedium}, tt.org.QuotaTier)
				assert.Equal(t, OrgStatusActive, tt.org.Status)
				assert.True(t, tt.org.IsActive)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	tests := []struct {
		name    string
		orgID   int64
		setup   func()
		wantErr bool
		errMsg  string
	}{
		{
			name:  "successful get",
			orgID: 1,
			setup: func() {
				settingsJSON, _ := json.Marshal(map[string]any{"key": "value"})
				rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
					"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
					AddRow(1, "Test Org", "test-org", "Test Organization", "Test Description",
						ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON, now, now)
				mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE id`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "organization not found",
			orgID: 999,
			setup: func() {
				mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE id`).
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
			errMsg:  "organization not found",
		},
		{
			name:  "database error",
			orgID: 2,
			setup: func() {
				mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE id`).
					WithArgs(int64(2)).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			org, err := service.GetOrganization(tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, org)
				assert.Equal(t, tt.orgID, org.ID)
				assert.Equal(t, "Test Org", org.Name)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationBySlug(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	tests := []struct {
		name    string
		slug    string
		setup   func()
		wantErr bool
	}{
		{
			name: "successful get by slug",
			slug: "test-org",
			setup: func() {
				settingsJSON, _ := json.Marshal(map[string]any{"key": "value"})
				rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
					"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
					AddRow(1, "Test Org", "test-org", "Test Organization", "Test Description",
						ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON, now, now)
				mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE slug`).
					WithArgs("test-org").
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "slug not found",
			slug: "nonexistent",
			setup: func() {
				mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE slug`).
					WithArgs("nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			org, err := service.GetOrganizationBySlug(tt.slug)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, org)
				assert.Equal(t, tt.slug, org.Slug)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListOrganizations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)
	userID := int64(456)

	tests := []struct {
		name    string
		userID  int64
		setup   func()
		wantLen int
		wantErr bool
	}{
		{
			name:   "successful list with multiple orgs",
			userID: userID,
			setup: func() {
				settingsJSON1, _ := json.Marshal(map[string]any{"key": "value1"})
				settingsJSON2, _ := json.Marshal(map[string]any{"key": "value2"})
				rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
					"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
					AddRow(1, "Org 1", "org-1", "Organization 1", "First org",
						ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON1, now, now).
					AddRow(2, "Org 2", "org-2", "Organization 2", "Second org",
						ownerID, QuotaTierMedium, OrgStatusActive, true, settingsJSON2, now, now)
				mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:   "empty list",
			userID: userID,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
					"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"})
				mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
					WithArgs(userID).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:   "database error",
			userID: userID,
			setup: func() {
				mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
					WithArgs(userID).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			orgs, err := service.ListOrganizations(tt.userID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, orgs, tt.wantLen)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	displayName := "Updated Display Name"
	description := "Updated Description"
	settings := map[string]any{"key": "new_value"}

	tests := []struct {
		name    string
		orgID   int64
		updates *UpdateOrgRequest
		setup   func()
		wantErr bool
	}{
		{
			name:  "update display name only",
			orgID: 1,
			updates: &UpdateOrgRequest{
				DisplayName: &displayName,
			},
			setup: func() {
				mock.ExpectExec(`UPDATE organizations SET display_name`).
					WithArgs(displayName, int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:  "update all fields",
			orgID: 2,
			updates: &UpdateOrgRequest{
				DisplayName: &displayName,
				Description: &description,
				Settings:    settings,
			},
			setup: func() {
				settingsJSON, _ := json.Marshal(settings)
				mock.ExpectExec(`UPDATE organizations SET display_name`).
					WithArgs(displayName, description, settingsJSON, int64(2)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:    "no updates",
			orgID:   3,
			updates: &UpdateOrgRequest{},
			setup: func() {
				// No mock expectation as no query should be executed
			},
			wantErr: false,
		},
		{
			name:  "organization not found",
			orgID: 999,
			updates: &UpdateOrgRequest{
				DisplayName: &displayName,
			},
			setup: func() {
				mock.ExpectExec(`UPDATE organizations SET display_name`).
					WithArgs(displayName, int64(999)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.UpdateOrganization(tt.orgID, tt.updates)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOrganization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	tests := []struct {
		name    string
		orgID   int64
		setup   func()
		wantErr bool
	}{
		{
			name:  "successful deletion",
			orgID: 1,
			setup: func() {
				mock.ExpectExec(`UPDATE organizations SET status`).
					WithArgs(OrgStatusDeleted, int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:  "organization not found",
			orgID: 999,
			setup: func() {
				mock.ExpectExec(`UPDATE organizations SET status`).
					WithArgs(OrgStatusDeleted, int64(999)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.DeleteOrganization(tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQuotas(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()

	tests := []struct {
		name    string
		orgID   int64
		setup   func()
		wantErr bool
	}{
		{
			name:  "successful get quotas",
			orgID: 1,
			setup: func() {
				customSettingsJSON, _ := json.Marshal(map[string]any{"custom": "setting"})
				rows := sqlmock.NewRows([]string{"id", "org_id", "max_modules", "max_versions_per_module",
					"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
					"custom_settings", "created_at", "updated_at"}).
					AddRow(1, int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000,
						customSettingsJSON, now, now)
				mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "quotas not found",
			orgID: 999,
			setup: func() {
				mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			quotas, err := service.GetQuotas(tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, quotas)
				assert.Equal(t, tt.orgID, quotas.OrgID)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateQuotas(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	quotas := &OrgQuotas{
		MaxModules:             20,
		MaxVersionsPerModule:   200,
		MaxStorageBytes:        10 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 10000,
		APIRateLimitPerHour:    10000,
	}

	tests := []struct {
		name    string
		orgID   int64
		quotas  *OrgQuotas
		setup   func()
		wantErr bool
	}{
		{
			name:   "successful update",
			orgID:  1,
			quotas: quotas,
			setup: func() {
				mock.ExpectExec(`UPDATE org_quotas SET`).
					WithArgs(20, 200, int64(10*1024*1024*1024), 10000, 10000, int64(1)).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:   "quotas not found",
			orgID:  999,
			quotas: quotas,
			setup: func() {
				mock.ExpectExec(`UPDATE org_quotas SET`).
					WithArgs(20, 200, int64(10*1024*1024*1024), 10000, 10000, int64(999)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.UpdateQuotas(tt.orgID, tt.quotas)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsage(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	tests := []struct {
		name    string
		orgID   int64
		setup   func()
		wantErr bool
	}{
		{
			name:  "successful get usage",
			orgID: 1,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "org_id", "period_start", "period_end",
					"modules_count", "versions_count", "storage_bytes", "compile_jobs_count",
					"api_requests_count", "created_at", "updated_at"}).
					AddRow(1, int64(1), periodStart, periodEnd, 5, 50, int64(1024*1024), 100, int64(1000), now, now)
				mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
					WithArgs(int64(1)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:  "usage not found - initialize and retry",
			orgID: 2,
			setup: func() {
				// First query returns no rows
				mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
					WithArgs(int64(2)).
					WillReturnError(sql.ErrNoRows)

				// Initialize usage period
				mock.ExpectExec(`INSERT INTO org_usage`).
					WithArgs(int64(2), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(2, 1))

				// Retry query after initialization
				rows := sqlmock.NewRows([]string{"id", "org_id", "period_start", "period_end",
					"modules_count", "versions_count", "storage_bytes", "compile_jobs_count",
					"api_requests_count", "created_at", "updated_at"}).
					AddRow(2, int64(2), periodStart, periodEnd, 0, 0, int64(0), 0, int64(0), now, now)
				mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
					WithArgs(int64(2)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			usage, err := service.GetUsage(tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, usage)
				assert.Equal(t, tt.orgID, usage.OrgID)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsageHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	tests := []struct {
		name    string
		orgID   int64
		limit   int
		setup   func()
		wantLen int
		wantErr bool
	}{
		{
			name:  "successful get history",
			orgID: 1,
			limit: 3,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "org_id", "period_start", "period_end",
					"modules_count", "versions_count", "storage_bytes", "compile_jobs_count",
					"api_requests_count", "created_at", "updated_at"}).
					AddRow(3, int64(1), periodStart.AddDate(0, -2, 0), periodEnd.AddDate(0, -2, 0), 5, 50, int64(1024), 100, int64(1000), now, now).
					AddRow(2, int64(1), periodStart.AddDate(0, -1, 0), periodEnd.AddDate(0, -1, 0), 8, 80, int64(2048), 150, int64(2000), now, now).
					AddRow(1, int64(1), periodStart, periodEnd, 10, 100, int64(4096), 200, int64(3000), now, now)
				mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
					WithArgs(int64(1), 3).
					WillReturnRows(rows)
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name:  "empty history",
			orgID: 2,
			limit: 5,
			setup: func() {
				rows := sqlmock.NewRows([]string{"id", "org_id", "period_start", "period_end",
					"modules_count", "versions_count", "storage_bytes", "compile_jobs_count",
					"api_requests_count", "created_at", "updated_at"})
				mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
					WithArgs(int64(2), 5).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			history, err := service.GetUsageHistory(tt.orgID, tt.limit)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, history, tt.wantLen)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestResetUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	tests := []struct {
		name    string
		orgID   int64
		setup   func()
		wantErr bool
	}{
		{
			name:  "successful reset",
			orgID: 1,
			setup: func() {
				mock.ExpectExec(`INSERT INTO org_usage`).
					WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:  "database error",
			orgID: 2,
			setup: func() {
				mock.ExpectExec(`INSERT INTO org_usage`).
					WithArgs(int64(2), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := service.ResetUsagePeriod(tt.orgID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDefaultQuotas_UnknownTier(t *testing.T) {
	service := &PostgresService{}

	// Test unknown tier defaults to small
	quotas := service.GetDefaultQuotas(QuotaTier("unknown"))
	assert.NotNil(t, quotas)
	assert.Equal(t, 10, quotas.MaxModules)
	assert.Equal(t, 100, quotas.MaxVersionsPerModule)
	assert.Equal(t, int64(5*1024*1024*1024), quotas.MaxStorageBytes)
}

func TestCreateOrganization_WithSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Create an organization with custom settings
	org := &Organization{
		Name:        "Test Org",
		DisplayName: "Test Organization",
		OwnerID:     &ownerID,
		Settings:    map[string]any{"theme": "dark", "notifications": true},
	}

	settingsJSON, _ := json.Marshal(org.Settings)
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Test Org", "test-org", "Test Organization", "",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.NotZero(t, org.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListOrganizations_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	userID := int64(456)

	// Return a row with wrong column count to trigger scan error
	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
		WithArgs(userID).
		WillReturnRows(rows)

	orgs, err := service.ListOrganizations(userID)
	assert.Error(t, err)
	assert.Nil(t, orgs)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListOrganizations_UnmarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)
	userID := int64(456)

	// Return invalid JSON to trigger unmarshal error
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(1, "Org 1", "org-1", "Organization 1", "First org",
			ownerID, QuotaTierSmall, OrgStatusActive, true, []byte("{invalid json}"), now, now)
	mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
		WithArgs(userID).
		WillReturnRows(rows)

	orgs, err := service.ListOrganizations(userID)
	assert.Error(t, err)
	assert.Nil(t, orgs)
	assert.Contains(t, err.Error(), "failed to unmarshal settings")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrganization_UpdateError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	displayName := "Updated Display Name"

	mock.ExpectExec(`UPDATE organizations SET display_name`).
		WithArgs(displayName, int64(1)).
		WillReturnError(sql.ErrConnDone)

	err = service.UpdateOrganization(1, &UpdateOrgRequest{
		DisplayName: &displayName,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update organization")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrganization_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	displayName := "Updated Display Name"

	// Use a mock result that returns an error when RowsAffected is called
	mock.ExpectExec(`UPDATE organizations SET display_name`).
		WithArgs(displayName, int64(1)).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err = service.UpdateOrganization(1, &UpdateOrgRequest{
		DisplayName: &displayName,
	})
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOrganization_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectExec(`UPDATE organizations SET status`).
		WithArgs(OrgStatusDeleted, int64(1)).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err = service.DeleteOrganization(1)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQuotas_UnmarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()

	// Return invalid JSON for custom settings
	rows := sqlmock.NewRows([]string{"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at"}).
		AddRow(1, int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000,
			[]byte("{invalid json}"), now, now)
	mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	quotas, err := service.GetQuotas(1)
	assert.Error(t, err)
	assert.Nil(t, quotas)
	assert.Contains(t, err.Error(), "failed to unmarshal custom settings")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateQuotas_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	quotas := &OrgQuotas{
		MaxModules:             20,
		MaxVersionsPerModule:   200,
		MaxStorageBytes:        10 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 10000,
		APIRateLimitPerHour:    10000,
	}

	mock.ExpectExec(`UPDATE org_quotas SET`).
		WithArgs(20, 200, int64(10*1024*1024*1024), 10000, 10000, int64(1)).
		WillReturnError(sql.ErrConnDone)

	err = service.UpdateQuotas(1, quotas)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update quotas")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateQuotas_RowsAffectedError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	quotas := &OrgQuotas{
		MaxModules:             20,
		MaxVersionsPerModule:   200,
		MaxStorageBytes:        10 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 10000,
		APIRateLimitPerHour:    10000,
	}

	mock.ExpectExec(`UPDATE org_quotas SET`).
		WithArgs(20, 200, int64(10*1024*1024*1024), 10000, 10000, int64(1)).
		WillReturnResult(sqlmock.NewErrorResult(sql.ErrConnDone))

	err = service.UpdateQuotas(1, quotas)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsageHistory_ScanError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	// Return a row with wrong column count to trigger scan error
	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
		WithArgs(int64(1), 3).
		WillReturnRows(rows)

	history, err := service.GetUsageHistory(1, 3)
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "failed to scan usage")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganization_UnmarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Return invalid JSON to trigger unmarshal error
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(1, "Test Org", "test-org", "Test Organization", "Test Description",
			ownerID, QuotaTierSmall, OrgStatusActive, true, []byte("{invalid json}"), now, now)
	mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE id`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	org, err := service.GetOrganization(1)
	assert.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "failed to unmarshal settings")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationBySlug_UnmarshalError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Return invalid JSON to trigger unmarshal error
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(1, "Test Org", "test-org", "Test Organization", "Test Description",
			ownerID, QuotaTierSmall, OrgStatusActive, true, []byte("{invalid json}"), now, now)
	mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE slug`).
		WithArgs("test-org").
		WillReturnRows(rows)

	org, err := service.GetOrganizationBySlug("test-org")
	assert.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "failed to unmarshal settings")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_CreateQuotasError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Test Org", "test-org", "Test Organization", "A test organization",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	// Mock for createQuotas - return error
	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnError(sql.ErrConnDone)

	org := &Organization{
		Name:        "Test Org",
		DisplayName: "Test Organization",
		Description: "A test organization",
		OwnerID:     &ownerID,
	}

	err = service.CreateOrganization(org)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create quotas")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_InitializeUsageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Test Org", "test-org", "Test Organization", "A test organization",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	// Mock for createQuotas - success
	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	// Mock for initializeUsagePeriod - error
	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	org := &Organization{
		Name:        "Test Org",
		DisplayName: "Test Organization",
		Description: "A test organization",
		OwnerID:     &ownerID,
	}

	err = service.CreateOrganization(org)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize usage")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsage_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	usage, err := service.GetUsage(1)
	assert.Error(t, err)
	assert.Nil(t, usage)
	assert.Contains(t, err.Error(), "failed to get usage")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQuotas_EmptyCustomSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()

	// Test with empty custom settings JSON
	rows := sqlmock.NewRows([]string{"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at"}).
		AddRow(1, int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000,
			[]byte{}, now, now)
	mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	quotas, err := service.GetQuotas(1)
	assert.NoError(t, err)
	assert.NotNil(t, quotas)
	assert.Equal(t, int64(1), quotas.OrgID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationBySlug_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE slug`).
		WithArgs("test-org").
		WillReturnError(sql.ErrConnDone)

	org, err := service.GetOrganizationBySlug("test-org")
	assert.Error(t, err)
	assert.Nil(t, org)
	assert.Contains(t, err.Error(), "failed to get organization")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOrganization_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectExec(`UPDATE organizations SET status`).
		WithArgs(OrgStatusDeleted, int64(1)).
		WillReturnError(sql.ErrConnDone)

	err = service.DeleteOrganization(1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete organization")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrganization_DescriptionOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	description := "Updated description only"

	mock.ExpectExec(`UPDATE organizations SET description`).
		WithArgs(description, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.UpdateOrganization(1, &UpdateOrgRequest{
		Description: &description,
	})
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrganization_SettingsOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	settings := map[string]any{"theme": "light"}
	settingsJSON, _ := json.Marshal(settings)

	mock.ExpectExec(`UPDATE organizations SET settings`).
		WithArgs(settingsJSON, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.UpdateOrganization(1, &UpdateOrgRequest{
		Settings: settings,
	})
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsage_InitializeUsageError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	// First query returns no rows
	mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrNoRows)

	// Initialize usage period fails
	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	usage, err := service.GetUsage(1)
	assert.Error(t, err)
	assert.Nil(t, usage)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateSlug_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: "---",
		},
		{
			name:     "only special chars",
			input:    "@#$%",
			expected: "",
		},
		{
			name:     "mixed alphanumeric with unicode",
			input:    "My Org 123 café",
			expected: "my-org-123-caf",
		},
		{
			name:     "consecutive spaces",
			input:    "My  Org   Name",
			expected: "my--org---name",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Test Org  ",
			expected: "--test-org--",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetQuotas_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	quotas, err := service.GetQuotas(1)
	assert.Error(t, err)
	assert.Nil(t, quotas)
	assert.Contains(t, err.Error(), "failed to get quotas")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsageHistory_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
		WithArgs(int64(1), 5).
		WillReturnError(sql.ErrConnDone)

	history, err := service.GetUsageHistory(1, 5)
	assert.Error(t, err)
	assert.Nil(t, history)
	assert.Contains(t, err.Error(), "failed to get usage history")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_EmptySlug(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Organization with empty slug should generate one from name
	org := &Organization{
		Name:        "My Test Organization",
		DisplayName: "Test Org",
		OwnerID:     &ownerID,
		Slug:        "", // Empty slug
	}

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("My Test Organization", "my-test-organization", "Test Org", "",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, "my-test-organization", org.Slug)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_EmptyQuotaTier(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Organization with empty quota tier should default to small
	org := &Organization{
		Name:        "Test Org",
		DisplayName: "Test Organization",
		OwnerID:     &ownerID,
		QuotaTier:   "", // Empty quota tier
	}

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Test Org", "test-org", "Test Organization", "",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, QuotaTierSmall, org.QuotaTier)
	assert.Equal(t, OrgStatusActive, org.Status)
	assert.True(t, org.IsActive)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_LargeTier(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	org := &Organization{
		Name:        "Large Org",
		DisplayName: "Large Organization",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierLarge,
	}

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Large Org", "large-org", "Large Organization", "",
			ownerID, QuotaTierLarge, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 200, 2000, int64(100*1024*1024*1024), 100000, 100000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, QuotaTierLarge, org.QuotaTier)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrganization_UnlimitedTier(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	org := &Organization{
		Name:        "Unlimited Org",
		DisplayName: "Unlimited Organization",
		OwnerID:     &ownerID,
		QuotaTier:   QuotaTierUnlimited,
	}

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Unlimited Org", "unlimited-org", "Unlimited Organization", "",
			ownerID, QuotaTierUnlimited, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 999999, 999999, int64(999999*1024*1024*1024), 999999999, 999999999, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, QuotaTierUnlimited, org.QuotaTier)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitializeUsagePeriod(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	// Test the ON CONFLICT DO NOTHING behavior
	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.ResetUsagePeriod(1)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIntegration_FullOrganizationLifecycle(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Create organization
	org := &Organization{
		Name:        "Full Lifecycle Org",
		DisplayName: "Full Lifecycle Organization",
		OwnerID:     &ownerID,
	}

	settingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Full Lifecycle Org", "full-lifecycle-org", "Full Lifecycle Organization", "",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), org.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGenerateSlug_Numbers(t *testing.T) {
	result := generateSlug("123-456")
	assert.Equal(t, "123-456", result)
}

func TestGenerateSlug_Lowercase(t *testing.T) {
	result := generateSlug("alreadylowercase")
	assert.Equal(t, "alreadylowercase", result)
}

func TestGenerateToken_UniqueTokens(t *testing.T) {
	// Generate multiple tokens and ensure they're all unique
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		token, err := generateToken()
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, 64, len(token))
		assert.False(t, tokens[token], "Token should be unique")
		tokens[token] = true
	}
}

func TestCreateOrganization_WithCustomSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	// Organization with custom settings in both org and quotas
	org := &Organization{
		Name:        "Custom Settings Org",
		DisplayName: "Custom Settings Organization",
		OwnerID:     &ownerID,
		Settings:    map[string]any{"theme": "dark", "language": "en"},
	}

	settingsJSON, _ := json.Marshal(org.Settings)
	mock.ExpectQuery(`INSERT INTO organizations`).
		WithArgs("Custom Settings Org", "custom-settings-org", "Custom Settings Organization", "",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	customSettingsJSON, _ := json.Marshal(map[string]any(nil))
	mock.ExpectQuery(`INSERT INTO org_quotas`).
		WithArgs(int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000, customSettingsJSON).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, now))

	mock.ExpectExec(`INSERT INTO org_usage`).
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = service.CreateOrganization(org)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), org.ID)
	assert.Equal(t, "custom-settings-org", org.Slug)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetQuotas_WithCustomSettings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()

	customSettings := map[string]any{"custom_key": "custom_value"}
	customSettingsJSON, _ := json.Marshal(customSettings)
	rows := sqlmock.NewRows([]string{"id", "org_id", "max_modules", "max_versions_per_module",
		"max_storage_bytes", "max_compile_jobs_per_month", "api_rate_limit_per_hour",
		"custom_settings", "created_at", "updated_at"}).
		AddRow(1, int64(1), 10, 100, int64(5*1024*1024*1024), 5000, 5000,
			customSettingsJSON, now, now)
	mock.ExpectQuery(`SELECT (.+) FROM org_quotas WHERE org_id`).
		WithArgs(int64(1)).
		WillReturnRows(rows)

	quotas, err := service.GetQuotas(1)
	assert.NoError(t, err)
	assert.NotNil(t, quotas)
	assert.NotNil(t, quotas.CustomSettings)
	assert.Equal(t, "custom_value", quotas.CustomSettings["custom_key"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListOrganizations_SingleOrg(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)
	userID := int64(456)

	settingsJSON, _ := json.Marshal(map[string]any{"key": "value"})
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(1, "Org 1", "org-1", "Organization 1", "First org",
			ownerID, QuotaTierSmall, OrgStatusActive, true, settingsJSON, now, now)
	mock.ExpectQuery(`SELECT DISTINCT (.+) FROM organizations o JOIN organization_members om`).
		WithArgs(userID).
		WillReturnRows(rows)

	orgs, err := service.ListOrganizations(userID)
	assert.NoError(t, err)
	assert.Len(t, orgs, 1)
	assert.Equal(t, "Org 1", orgs[0].Name)
	assert.NotNil(t, orgs[0].Settings)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganization_ScanSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	settingsJSON, _ := json.Marshal(map[string]any{})
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(5, "Scan Test", "scan-test", "Scan Test Org", "Testing scan",
			ownerID, QuotaTierMedium, OrgStatusActive, true, settingsJSON, now, now)
	mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE id`).
		WithArgs(int64(5)).
		WillReturnRows(rows)

	org, err := service.GetOrganization(5)
	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, int64(5), org.ID)
	assert.Equal(t, "Scan Test", org.Name)
	assert.Equal(t, QuotaTierMedium, org.QuotaTier)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOrganizationBySlug_ScanSuccess(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(123)

	settingsJSON, _ := json.Marshal(map[string]any{})
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "display_name", "description",
		"owner_id", "quota_tier", "status", "is_active", "settings", "created_at", "updated_at"}).
		AddRow(7, "Slug Test", "slug-test-org", "Slug Test Org", "Testing slug lookup",
			ownerID, QuotaTierLarge, OrgStatusActive, true, settingsJSON, now, now)
	mock.ExpectQuery(`SELECT (.+) FROM organizations WHERE slug`).
		WithArgs("slug-test-org").
		WillReturnRows(rows)

	org, err := service.GetOrganizationBySlug("slug-test-org")
	assert.NoError(t, err)
	assert.NotNil(t, org)
	assert.Equal(t, "slug-test-org", org.Slug)
	assert.Equal(t, QuotaTierLarge, org.QuotaTier)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUsageHistory_ThreeMonths(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	rows := sqlmock.NewRows([]string{"id", "org_id", "period_start", "period_end",
		"modules_count", "versions_count", "storage_bytes", "compile_jobs_count",
		"api_requests_count", "created_at", "updated_at"}).
		AddRow(1, int64(1), periodStart, periodEnd, 10, 100, int64(4096), 200, int64(3000), now, now).
		AddRow(2, int64(1), periodStart.AddDate(0, -1, 0), periodEnd.AddDate(0, -1, 0), 8, 80, int64(2048), 150, int64(2000), now, now).
		AddRow(3, int64(1), periodStart.AddDate(0, -2, 0), periodEnd.AddDate(0, -2, 0), 5, 50, int64(1024), 100, int64(1000), now, now)

	mock.ExpectQuery(`SELECT (.+) FROM org_usage WHERE org_id`).
		WithArgs(int64(1), 3).
		WillReturnRows(rows)

	history, err := service.GetUsageHistory(1, 3)
	assert.NoError(t, err)
	assert.Len(t, history, 3)
	assert.Equal(t, 10, history[0].ModulesCount)
	assert.Equal(t, 8, history[1].ModulesCount)
	assert.Equal(t, 5, history[2].ModulesCount)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteOrganization_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	mock.ExpectExec(`UPDATE organizations SET status`).
		WithArgs(OrgStatusDeleted, int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.DeleteOrganization(10)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateQuotas_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)

	quotas := &OrgQuotas{
		MaxModules:             15,
		MaxVersionsPerModule:   150,
		MaxStorageBytes:        8 * 1024 * 1024 * 1024,
		MaxCompileJobsPerMonth: 8000,
		APIRateLimitPerHour:    8000,
	}

	mock.ExpectExec(`UPDATE org_quotas SET`).
		WithArgs(15, 150, int64(8*1024*1024*1024), 8000, 8000, int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = service.UpdateQuotas(5, quotas)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

//
func TestComprehensiveServiceCoverage(t *testing.T) {
	// This test exercises multiple code paths to increase overall coverage
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewPostgresService(db)
	now := time.Now()
	ownerID := int64(999)

	t.Run("Create org with all tiers", func(t *testing.T) {
		tiers := []QuotaTier{QuotaTierSmall, QuotaTierMedium, QuotaTierLarge, QuotaTierUnlimited}
		for i, tier := range tiers {
			orgID := int64(i + 100)
			org := &Organization{
				Name:        "Tier Test " + string(tier),
				DisplayName: "Tier Test",
				OwnerID:     &ownerID,
				QuotaTier:   tier,
			}

			settingsJSON, _ := json.Marshal(map[string]any(nil))
			mock.ExpectQuery(`INSERT INTO organizations`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
					sqlmock.AnyArg(), tier, sqlmock.AnyArg(), sqlmock.AnyArg(), settingsJSON).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(orgID, now, now))

			mock.ExpectQuery(`INSERT INTO org_quotas`).
				WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(),
					sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(orgID, now, now))

			mock.ExpectExec(`INSERT INTO org_usage`).
				WithArgs(orgID, sqlmock.AnyArg(), sqlmock.AnyArg()).
				WillReturnResult(sqlmock.NewResult(orgID, 1))

			err := service.CreateOrganization(org)
			assert.NoError(t, err)
		}
	})

	t.Run("Update operations", func(t *testing.T) {
		// Test update with different field combinations
		newDesc := "New description"
		mock.ExpectExec(`UPDATE organizations SET description`).
			WithArgs(newDesc, int64(1)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.UpdateOrganization(1, &UpdateOrgRequest{
			Description: &newDesc,
		})
		assert.NoError(t, err)
	})

	assert.NoError(t, mock.ExpectationsWereMet())
}
