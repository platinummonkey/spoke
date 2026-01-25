package orgs

import (
	"database/sql"
	"fmt"
)

// CheckModuleQuota checks if organization can create a new module
func (s *PostgresService) CheckModuleQuota(orgID int64) error {
	quotas, err := s.GetQuotas(orgID)
	if err != nil {
		return fmt.Errorf("failed to get quotas: %w", err)
	}

	usage, err := s.GetUsage(orgID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	if usage.ModulesCount >= quotas.MaxModules {
		return &QuotaExceededError{
			Resource: "modules",
			Current:  int64(usage.ModulesCount),
			Limit:    int64(quotas.MaxModules),
		}
	}

	return nil
}

// CheckVersionQuota checks if organization can create a new version
func (s *PostgresService) CheckVersionQuota(orgID int64, moduleName string) error {
	quotas, err := s.GetQuotas(orgID)
	if err != nil {
		return fmt.Errorf("failed to get quotas: %w", err)
	}

	// Count versions for this module
	query := `
		SELECT COUNT(*)
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.org_id = $1 AND m.name = $2
	`
	var count int
	if err := s.db.QueryRow(query, orgID, moduleName).Scan(&count); err != nil {
		return fmt.Errorf("failed to count versions: %w", err)
	}

	if count >= quotas.MaxVersionsPerModule {
		return &QuotaExceededError{
			Resource: "versions",
			Current:  int64(count),
			Limit:    int64(quotas.MaxVersionsPerModule),
		}
	}

	return nil
}

// CheckStorageQuota checks if organization can store additional bytes
func (s *PostgresService) CheckStorageQuota(orgID int64, additionalBytes int64) error {
	quotas, err := s.GetQuotas(orgID)
	if err != nil {
		return fmt.Errorf("failed to get quotas: %w", err)
	}

	usage, err := s.GetUsage(orgID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	if usage.StorageBytes+additionalBytes > quotas.MaxStorageBytes {
		return &QuotaExceededError{
			Resource: "storage",
			Current:  usage.StorageBytes + additionalBytes,
			Limit:    quotas.MaxStorageBytes,
		}
	}

	return nil
}

// CheckCompileJobQuota checks if organization can run another compile job
func (s *PostgresService) CheckCompileJobQuota(orgID int64) error {
	quotas, err := s.GetQuotas(orgID)
	if err != nil {
		return fmt.Errorf("failed to get quotas: %w", err)
	}

	usage, err := s.GetUsage(orgID)
	if err != nil {
		return fmt.Errorf("failed to get usage: %w", err)
	}

	if usage.CompileJobsCount >= quotas.MaxCompileJobsPerMonth {
		return &QuotaExceededError{
			Resource: "compile_jobs",
			Current:  int64(usage.CompileJobsCount),
			Limit:    int64(quotas.MaxCompileJobsPerMonth),
		}
	}

	return nil
}

// CheckAPIRateLimit checks if organization is within API rate limit
func (s *PostgresService) CheckAPIRateLimit(orgID int64) error {
	quotas, err := s.GetQuotas(orgID)
	if err != nil {
		return fmt.Errorf("failed to get quotas: %w", err)
	}

	// Count API requests in the last hour
	query := `
		SELECT COUNT(*)
		FROM audit_logs
		WHERE organization_id = $1 AND created_at > NOW() - INTERVAL '1 hour'
	`
	var count int64
	if err := s.db.QueryRow(query, orgID).Scan(&count); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to count API requests: %w", err)
	}

	if count >= int64(quotas.APIRateLimitPerHour) {
		return &QuotaExceededError{
			Resource: "api_requests",
			Current:  count,
			Limit:    int64(quotas.APIRateLimitPerHour),
		}
	}

	return nil
}

// IncrementModules increments the module count for an organization
func (s *PostgresService) IncrementModules(orgID int64) error {
	query := `
		UPDATE org_usage
		SET modules_count = modules_count + 1
		WHERE org_id = $1 AND period_end > NOW()
	`
	result, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to increment modules: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Initialize usage period if not found
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return err
		}
		return s.IncrementModules(orgID)
	}

	return nil
}

// IncrementVersions increments the version count for an organization
func (s *PostgresService) IncrementVersions(orgID int64) error {
	query := `
		UPDATE org_usage
		SET versions_count = versions_count + 1
		WHERE org_id = $1 AND period_end > NOW()
	`
	result, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to increment versions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return err
		}
		return s.IncrementVersions(orgID)
	}

	return nil
}

// IncrementStorage increments the storage usage for an organization
func (s *PostgresService) IncrementStorage(orgID int64, bytes int64) error {
	query := `
		UPDATE org_usage
		SET storage_bytes = storage_bytes + $1
		WHERE org_id = $2 AND period_end > NOW()
	`
	result, err := s.db.Exec(query, bytes, orgID)
	if err != nil {
		return fmt.Errorf("failed to increment storage: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return err
		}
		return s.IncrementStorage(orgID, bytes)
	}

	return nil
}

// IncrementCompileJobs increments the compile job count for an organization
func (s *PostgresService) IncrementCompileJobs(orgID int64) error {
	query := `
		UPDATE org_usage
		SET compile_jobs_count = compile_jobs_count + 1
		WHERE org_id = $1 AND period_end > NOW()
	`
	result, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to increment compile jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return err
		}
		return s.IncrementCompileJobs(orgID)
	}

	return nil
}

// IncrementAPIRequests increments the API request count for an organization
func (s *PostgresService) IncrementAPIRequests(orgID int64) error {
	query := `
		UPDATE org_usage
		SET api_requests_count = api_requests_count + 1
		WHERE org_id = $1 AND period_end > NOW()
	`
	result, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to increment API requests: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		if err := s.initializeUsagePeriod(orgID); err != nil {
			return err
		}
		return s.IncrementAPIRequests(orgID)
	}

	return nil
}

// DecrementModules decrements the module count for an organization
func (s *PostgresService) DecrementModules(orgID int64) error {
	query := `
		UPDATE org_usage
		SET modules_count = GREATEST(modules_count - 1, 0)
		WHERE org_id = $1 AND period_end > NOW()
	`
	_, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to decrement modules: %w", err)
	}
	return nil
}

// DecrementVersions decrements the version count for an organization
func (s *PostgresService) DecrementVersions(orgID int64) error {
	query := `
		UPDATE org_usage
		SET versions_count = GREATEST(versions_count - 1, 0)
		WHERE org_id = $1 AND period_end > NOW()
	`
	_, err := s.db.Exec(query, orgID)
	if err != nil {
		return fmt.Errorf("failed to decrement versions: %w", err)
	}
	return nil
}

// DecrementStorage decrements the storage usage for an organization
func (s *PostgresService) DecrementStorage(orgID int64, bytes int64) error {
	query := `
		UPDATE org_usage
		SET storage_bytes = GREATEST(storage_bytes - $1, 0)
		WHERE org_id = $2 AND period_end > NOW()
	`
	_, err := s.db.Exec(query, bytes, orgID)
	if err != nil {
		return fmt.Errorf("failed to decrement storage: %w", err)
	}
	return nil
}
