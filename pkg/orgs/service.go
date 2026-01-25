package orgs

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PostgresService implements the Service interface using PostgreSQL
type PostgresService struct {
	db *sql.DB
}

// NewPostgresService creates a new PostgresService
func NewPostgresService(db *sql.DB) *PostgresService {
	return &PostgresService{db: db}
}

// CreateOrganization creates a new organization
func (s *PostgresService) CreateOrganization(org *Organization) error {
	// Generate slug from name if not provided
	if org.Slug == "" {
		org.Slug = generateSlug(org.Name)
	}

	// Set defaults
	if org.QuotaTier == "" {
		org.QuotaTier = QuotaTierSmall
	}
	if org.Status == "" {
		org.Status = OrgStatusActive
	}
	org.IsActive = true

	settingsJSON, err := json.Marshal(org.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	query := `
		INSERT INTO organizations (name, slug, display_name, description, owner_id, quota_tier, status, is_active, settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`
	err = s.db.QueryRow(query, org.Name, org.Slug, org.DisplayName, org.Description,
		org.OwnerID, org.QuotaTier, org.Status, org.IsActive, settingsJSON).
		Scan(&org.ID, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	// Create default quotas
	quotas := s.GetDefaultQuotas(org.QuotaTier)
	quotas.OrgID = org.ID
	if err := s.createQuotas(quotas); err != nil {
		return fmt.Errorf("failed to create quotas: %w", err)
	}

	// Initialize usage tracking
	if err := s.initializeUsagePeriod(org.ID); err != nil {
		return fmt.Errorf("failed to initialize usage: %w", err)
	}

	return nil
}

// GetOrganization retrieves an organization by ID
func (s *PostgresService) GetOrganization(id int64) (*Organization, error) {
	query := `
		SELECT id, name, slug, display_name, description, owner_id, quota_tier, status,
		       is_active, settings, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`
	org := &Organization{}
	var settingsJSON []byte
	err := s.db.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Slug, &org.DisplayName, &org.Description,
		&org.OwnerID, &org.QuotaTier, &org.Status, &org.IsActive, &settingsJSON,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &org.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return org, nil
}

// GetOrganizationBySlug retrieves an organization by slug
func (s *PostgresService) GetOrganizationBySlug(slug string) (*Organization, error) {
	query := `
		SELECT id, name, slug, display_name, description, owner_id, quota_tier, status,
		       is_active, settings, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`
	org := &Organization{}
	var settingsJSON []byte
	err := s.db.QueryRow(query, slug).Scan(
		&org.ID, &org.Name, &org.Slug, &org.DisplayName, &org.Description,
		&org.OwnerID, &org.QuotaTier, &org.Status, &org.IsActive, &settingsJSON,
		&org.CreatedAt, &org.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if err := json.Unmarshal(settingsJSON, &org.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	return org, nil
}

// ListOrganizations lists organizations for a user
func (s *PostgresService) ListOrganizations(userID int64) ([]*Organization, error) {
	query := `
		SELECT DISTINCT o.id, o.name, o.slug, o.display_name, o.description, o.owner_id,
		       o.quota_tier, o.status, o.is_active, o.settings, o.created_at, o.updated_at
		FROM organizations o
		JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND o.is_active = true
		ORDER BY o.created_at DESC
	`
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		var settingsJSON []byte
		if err := rows.Scan(
			&org.ID, &org.Name, &org.Slug, &org.DisplayName, &org.Description,
			&org.OwnerID, &org.QuotaTier, &org.Status, &org.IsActive, &settingsJSON,
			&org.CreatedAt, &org.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		if err := json.Unmarshal(settingsJSON, &org.Settings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// UpdateOrganization updates an organization
func (s *PostgresService) UpdateOrganization(id int64, updates *UpdateOrgRequest) error {
	setClauses := []string{}
	args := []interface{}{}
	argPos := 1

	if updates.DisplayName != nil {
		setClauses = append(setClauses, fmt.Sprintf("display_name = $%d", argPos))
		args = append(args, *updates.DisplayName)
		argPos++
	}
	if updates.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *updates.Description)
		argPos++
	}
	if updates.Settings != nil {
		settingsJSON, err := json.Marshal(updates.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("settings = $%d", argPos))
		args = append(args, settingsJSON)
		argPos++
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE organizations SET %s WHERE id = $%d", strings.Join(setClauses, ", "), argPos)

	result, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// DeleteOrganization soft deletes an organization
func (s *PostgresService) DeleteOrganization(id int64) error {
	query := `UPDATE organizations SET status = $1, is_active = false WHERE id = $2`
	result, err := s.db.Exec(query, OrgStatusDeleted, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

// GetQuotas retrieves quotas for an organization
func (s *PostgresService) GetQuotas(orgID int64) (*OrgQuotas, error) {
	query := `
		SELECT id, org_id, max_modules, max_versions_per_module, max_storage_bytes,
		       max_compile_jobs_per_month, api_rate_limit_per_hour, custom_settings, created_at, updated_at
		FROM org_quotas
		WHERE org_id = $1
	`
	quotas := &OrgQuotas{}
	var customSettingsJSON []byte
	err := s.db.QueryRow(query, orgID).Scan(
		&quotas.ID, &quotas.OrgID, &quotas.MaxModules, &quotas.MaxVersionsPerModule,
		&quotas.MaxStorageBytes, &quotas.MaxCompileJobsPerMonth, &quotas.APIRateLimitPerHour,
		&customSettingsJSON, &quotas.CreatedAt, &quotas.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("quotas not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get quotas: %w", err)
	}

	if len(customSettingsJSON) > 0 {
		if err := json.Unmarshal(customSettingsJSON, &quotas.CustomSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal custom settings: %w", err)
		}
	}

	return quotas, nil
}

// UpdateQuotas updates quotas for an organization
func (s *PostgresService) UpdateQuotas(orgID int64, quotas *OrgQuotas) error {
	query := `
		UPDATE org_quotas
		SET max_modules = $1, max_versions_per_module = $2, max_storage_bytes = $3,
		    max_compile_jobs_per_month = $4, api_rate_limit_per_hour = $5
		WHERE org_id = $6
	`
	result, err := s.db.Exec(query, quotas.MaxModules, quotas.MaxVersionsPerModule,
		quotas.MaxStorageBytes, quotas.MaxCompileJobsPerMonth, quotas.APIRateLimitPerHour, orgID)
	if err != nil {
		return fmt.Errorf("failed to update quotas: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("quotas not found")
	}

	return nil
}

// GetDefaultQuotas returns default quotas for a quota tier (free, no billing)
func (s *PostgresService) GetDefaultQuotas(quotaTier QuotaTier) *OrgQuotas {
	switch quotaTier {
	case QuotaTierSmall:
		return &OrgQuotas{
			MaxModules:             10,
			MaxVersionsPerModule:   100,
			MaxStorageBytes:        5 * 1024 * 1024 * 1024, // 5GB
			MaxCompileJobsPerMonth: 5000,
			APIRateLimitPerHour:    5000,
		}
	case QuotaTierMedium:
		return &OrgQuotas{
			MaxModules:             50,
			MaxVersionsPerModule:   500,
			MaxStorageBytes:        25 * 1024 * 1024 * 1024, // 25GB
			MaxCompileJobsPerMonth: 25000,
			APIRateLimitPerHour:    25000,
		}
	case QuotaTierLarge:
		return &OrgQuotas{
			MaxModules:             200,
			MaxVersionsPerModule:   2000,
			MaxStorageBytes:        100 * 1024 * 1024 * 1024, // 100GB
			MaxCompileJobsPerMonth: 100000,
			APIRateLimitPerHour:    100000,
		}
	case QuotaTierUnlimited:
		return &OrgQuotas{
			MaxModules:             999999,
			MaxVersionsPerModule:   999999,
			MaxStorageBytes:        999999 * 1024 * 1024 * 1024, // Effectively unlimited
			MaxCompileJobsPerMonth: 999999999,
			APIRateLimitPerHour:    999999999,
		}
	default:
		// Default to small tier
		return &OrgQuotas{
			MaxModules:             10,
			MaxVersionsPerModule:   100,
			MaxStorageBytes:        5 * 1024 * 1024 * 1024,
			MaxCompileJobsPerMonth: 5000,
			APIRateLimitPerHour:    5000,
		}
	}
}

// createQuotas creates quotas for an organization
func (s *PostgresService) createQuotas(quotas *OrgQuotas) error {
	customSettingsJSON, err := json.Marshal(quotas.CustomSettings)
	if err != nil {
		return fmt.Errorf("failed to marshal custom settings: %w", err)
	}

	query := `
		INSERT INTO org_quotas (org_id, max_modules, max_versions_per_module, max_storage_bytes,
		                        max_compile_jobs_per_month, api_rate_limit_per_hour, custom_settings)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	return s.db.QueryRow(query, quotas.OrgID, quotas.MaxModules, quotas.MaxVersionsPerModule,
		quotas.MaxStorageBytes, quotas.MaxCompileJobsPerMonth, quotas.APIRateLimitPerHour, customSettingsJSON).
		Scan(&quotas.ID, &quotas.CreatedAt, &quotas.UpdatedAt)
}

// GetUsage retrieves current usage for an organization
func (s *PostgresService) GetUsage(orgID int64) (*OrgUsage, error) {
	query := `
		SELECT id, org_id, period_start, period_end, modules_count, versions_count,
		       storage_bytes, compile_jobs_count, api_requests_count, created_at, updated_at
		FROM org_usage
		WHERE org_id = $1 AND period_end > NOW()
		ORDER BY period_start DESC
		LIMIT 1
	`
	usage := &OrgUsage{}
	err := s.db.QueryRow(query, orgID).Scan(
		&usage.ID, &usage.OrgID, &usage.PeriodStart, &usage.PeriodEnd,
		&usage.ModulesCount, &usage.VersionsCount, &usage.StorageBytes,
		&usage.CompileJobsCount, &usage.APIRequestsCount,
		&usage.CreatedAt, &usage.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Initialize if not found
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return nil, err
		}
		return s.GetUsage(orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	return usage, nil
}

// GetUsageHistory retrieves usage history for an organization
func (s *PostgresService) GetUsageHistory(orgID int64, limit int) ([]*OrgUsage, error) {
	query := `
		SELECT id, org_id, period_start, period_end, modules_count, versions_count,
		       storage_bytes, compile_jobs_count, api_requests_count, created_at, updated_at
		FROM org_usage
		WHERE org_id = $1
		ORDER BY period_start DESC
		LIMIT $2
	`
	rows, err := s.db.Query(query, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage history: %w", err)
	}
	defer rows.Close()

	var usages []*OrgUsage
	for rows.Next() {
		usage := &OrgUsage{}
		if err := rows.Scan(
			&usage.ID, &usage.OrgID, &usage.PeriodStart, &usage.PeriodEnd,
			&usage.ModulesCount, &usage.VersionsCount, &usage.StorageBytes,
			&usage.CompileJobsCount, &usage.APIRequestsCount,
			&usage.CreatedAt, &usage.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan usage: %w", err)
		}
		usages = append(usages, usage)
	}

	return usages, nil
}

// ResetUsagePeriod creates a new usage period for an organization
func (s *PostgresService) ResetUsagePeriod(orgID int64) error {
	return s.initializeUsagePeriod(orgID)
}

// initializeUsagePeriod initializes a usage period for an organization
func (s *PostgresService) initializeUsagePeriod(orgID int64) error {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.AddDate(0, 1, 0)

	query := `
		INSERT INTO org_usage (org_id, period_start, period_end)
		VALUES ($1, $2, $3)
		ON CONFLICT (org_id, period_start) DO NOTHING
	`
	_, err := s.db.Exec(query, orgID, periodStart, periodEnd)
	return err
}

// Continue in next part...
// Helper function to generate slug from name
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	return slug
}

// generateToken generates a random token
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Member management methods are in the next section...
