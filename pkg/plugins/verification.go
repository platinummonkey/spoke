package plugins

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Verifier handles the plugin verification workflow
type Verifier struct {
	db          *sql.DB
	validator   *Validator
	logger      *logrus.Logger
	downloadDir string // Directory for temporary plugin downloads
}

// NewVerifier creates a new plugin verifier
func NewVerifier(db *sql.DB, logger *logrus.Logger) *Verifier {
	downloadDir := os.TempDir() + "/spoke-plugin-verification"
	os.MkdirAll(downloadDir, 0755)

	return &Verifier{
		db:          db,
		validator:   NewValidator(logger),
		logger:      logger,
		downloadDir: downloadDir,
	}
}

// VerificationRequest contains information needed to start verification
type VerificationRequest struct {
	PluginID     string
	Version      string
	SubmittedBy  string
	ManifestURL  string
	DownloadURL  string
	AutoApprove  bool // If true, automatically approve if no critical issues
}

// VerificationResult contains the complete verification outcome
type VerificationResult struct {
	VerificationID   int64
	PluginID         string
	Version          string
	Status           string // pending, in_progress, approved, rejected, review_required
	SecurityLevel    string // verified (if approved)
	ManifestErrors   []ValidationError
	SecurityIssues   []SecurityIssue
	PermissionIssues []ValidationError
	Reason           string
	StartedAt        time.Time
	CompletedAt      time.Time
	ProcessingTime   time.Duration
}

// SubmitForVerification creates a new verification request
func (v *Verifier) SubmitForVerification(ctx context.Context, req *VerificationRequest) (int64, error) {
	v.logger.Infof("Submitting plugin %s v%s for verification", req.PluginID, req.Version)

	query := `
		INSERT INTO plugin_verifications (plugin_id, version, status, submitted_by, submitted_at)
		VALUES ($1, $2, 'pending', $3, CURRENT_TIMESTAMP)
	`

	result, err := v.db.ExecContext(ctx, query, req.PluginID, req.Version, req.SubmittedBy)
	if err != nil {
		return 0, fmt.Errorf("failed to create verification request: %w", err)
	}

	verificationID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get verification ID: %w", err)
	}

	// Record audit entry
	v.recordAuditLog(ctx, verificationID, "submitted", req.SubmittedBy, "Verification request submitted")

	v.logger.Infof("Created verification request #%d for %s v%s", verificationID, req.PluginID, req.Version)
	return verificationID, nil
}

// RunVerification performs the complete verification process
func (v *Verifier) RunVerification(ctx context.Context, verificationID int64, downloadURL string) (*VerificationResult, error) {
	startTime := time.Now()
	v.logger.Infof("Starting verification #%d", verificationID)

	// Update status to in_progress
	if err := v.updateVerificationStatus(ctx, verificationID, "in_progress"); err != nil {
		return nil, err
	}

	// Get verification details
	var pluginID, version string
	err := v.db.QueryRowContext(ctx,
		"SELECT plugin_id, version FROM plugin_verifications WHERE id = $1",
		verificationID).Scan(&pluginID, &version)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification details: %w", err)
	}

	result := &VerificationResult{
		VerificationID: verificationID,
		PluginID:       pluginID,
		Version:        version,
		Status:         "in_progress",
		StartedAt:      startTime,
	}

	// Step 1: Download plugin
	pluginPath, cleanup, err := v.downloadPlugin(ctx, pluginID, version, downloadURL)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		result.Status = "rejected"
		result.Reason = fmt.Sprintf("Failed to download plugin: %v", err)
		v.completeVerification(ctx, result)
		return result, err
	}

	// Record scan start
	scanID, _ := v.recordScanStart(ctx, pluginID, version, "full-validation")

	// Step 2: Load and validate manifest
	manifestPath := filepath.Join(pluginPath, "plugin.yaml")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		result.Status = "rejected"
		result.Reason = fmt.Sprintf("Invalid manifest: %v", err)
		v.recordScanComplete(ctx, scanID, "failed", 0, 0, err.Error())
		v.completeVerification(ctx, result)
		return result, err
	}

	// Validate manifest
	manifestErrors := v.validator.ValidateManifest(manifest)
	result.ManifestErrors = manifestErrors

	// Store manifest errors
	for _, err := range manifestErrors {
		v.storeValidationError(ctx, verificationID, err)
	}

	// Step 3: Security scan
	securityIssues, err := v.validator.ScanForSecurityIssues(ctx, pluginPath)
	if err != nil {
		v.logger.Warnf("Security scan failed: %v", err)
		// Continue with partial results
	}
	result.SecurityIssues = securityIssues

	// Store security issues
	for _, issue := range securityIssues {
		v.storeSecurityIssue(ctx, verificationID, issue)
	}

	// Step 4: Determine verification outcome
	criticalIssues := 0
	highIssues := 0
	for _, issue := range securityIssues {
		switch issue.Severity {
		case "critical":
			criticalIssues++
		case "high":
			highIssues++
		}
	}

	criticalManifestErrors := 0
	for _, err := range manifestErrors {
		if err.Severity == "error" {
			criticalManifestErrors++
		}
	}

	// Record scan completion
	v.recordScanComplete(ctx, scanID, "completed", len(securityIssues), criticalIssues, "")

	// Decision logic
	if criticalManifestErrors > 0 {
		result.Status = "rejected"
		result.Reason = fmt.Sprintf("Found %d critical manifest errors", criticalManifestErrors)
	} else if criticalIssues > 0 {
		result.Status = "rejected"
		result.Reason = fmt.Sprintf("Found %d critical security issues", criticalIssues)
	} else if highIssues > 3 || len(securityIssues) > 10 {
		result.Status = "review_required"
		result.Reason = fmt.Sprintf("Found %d high-severity and %d total security issues requiring manual review", highIssues, len(securityIssues))
	} else {
		result.Status = "approved"
		result.SecurityLevel = "verified"
	}

	result.CompletedAt = time.Now()
	result.ProcessingTime = result.CompletedAt.Sub(result.StartedAt)

	// Step 5: Update database
	if err := v.completeVerification(ctx, result); err != nil {
		return result, err
	}

	v.logger.Infof("Verification #%d completed with status: %s (took %v)", verificationID, result.Status, result.ProcessingTime)
	return result, nil
}

// ApproveVerification manually approves a verification (for manual review)
func (v *Verifier) ApproveVerification(ctx context.Context, verificationID int64, approvedBy, reason string) error {
	v.logger.Infof("Manually approving verification #%d by %s", verificationID, approvedBy)

	query := `
		UPDATE plugin_verifications
		SET status = 'approved',
		    security_level = 'verified',
		    verified_by = $1,
		    completed_at = CURRENT_TIMESTAMP,
		    reason = $2
		WHERE id = $3
	`

	_, err := v.db.ExecContext(ctx, query, approvedBy, reason, verificationID)
	if err != nil {
		return fmt.Errorf("failed to approve verification: %w", err)
	}

	v.recordAuditLog(ctx, verificationID, "approved", approvedBy, reason)
	return nil
}

// RejectVerification manually rejects a verification
func (v *Verifier) RejectVerification(ctx context.Context, verificationID int64, rejectedBy, reason string) error {
	v.logger.Infof("Rejecting verification #%d by %s", verificationID, rejectedBy)

	query := `
		UPDATE plugin_verifications
		SET status = 'rejected',
		    verified_by = $1,
		    completed_at = CURRENT_TIMESTAMP,
		    reason = $2
		WHERE id = $3
	`

	_, err := v.db.ExecContext(ctx, query, rejectedBy, reason, verificationID)
	if err != nil {
		return fmt.Errorf("failed to reject verification: %w", err)
	}

	v.recordAuditLog(ctx, verificationID, "rejected", rejectedBy, reason)
	return nil
}

// GetVerificationStatus retrieves the current status of a verification
func (v *Verifier) GetVerificationStatus(ctx context.Context, verificationID int64) (*VerificationResult, error) {
	result := &VerificationResult{
		VerificationID: verificationID,
	}

	query := `
		SELECT plugin_id, version, status, security_level, reason,
		       submitted_at, started_at, completed_at
		FROM plugin_verifications
		WHERE id = $1
	`

	var startedAt, completedAt sql.NullTime
	var securityLevel sql.NullString
	var reason sql.NullString

	err := v.db.QueryRowContext(ctx, query, verificationID).Scan(
		&result.PluginID,
		&result.Version,
		&result.Status,
		&securityLevel,
		&reason,
		&result.StartedAt,
		&startedAt,
		&completedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get verification status: %w", err)
	}

	if securityLevel.Valid {
		result.SecurityLevel = securityLevel.String
	}
	if reason.Valid {
		result.Reason = reason.String
	}
	if startedAt.Valid {
		result.StartedAt = startedAt.Time
	}
	if completedAt.Valid {
		result.CompletedAt = completedAt.Time
		result.ProcessingTime = result.CompletedAt.Sub(result.StartedAt)
	}

	// Load validation errors
	result.ManifestErrors, _ = v.loadValidationErrors(ctx, verificationID)

	// Load security issues
	result.SecurityIssues, _ = v.loadSecurityIssues(ctx, verificationID)

	return result, nil
}

// ListPendingVerifications returns all verifications awaiting review
func (v *Verifier) ListPendingVerifications(ctx context.Context, limit int) ([]*VerificationResult, error) {
	query := `
		SELECT id, plugin_id, version, status, submitted_at
		FROM plugin_verifications
		WHERE status IN ('pending', 'review_required')
		ORDER BY submitted_at ASC
		LIMIT $1
	`

	rows, err := v.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending verifications: %w", err)
	}
	defer rows.Close()

	var results []*VerificationResult
	for rows.Next() {
		result := &VerificationResult{}
		err := rows.Scan(&result.VerificationID, &result.PluginID, &result.Version, &result.Status, &result.StartedAt)
		if err != nil {
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// Helper methods

func (v *Verifier) downloadPlugin(ctx context.Context, pluginID, version, downloadURL string) (string, func(), error) {
	// Create temporary directory for this plugin
	pluginDir := filepath.Join(v.downloadDir, fmt.Sprintf("%s-%s-%d", pluginID, version, time.Now().Unix()))
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return "", nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	cleanup := func() {
		os.RemoveAll(pluginDir)
	}

	// Download plugin archive
	archivePath := filepath.Join(pluginDir, "plugin.tar.gz")
	if err := v.downloadFile(ctx, downloadURL, archivePath); err != nil {
		return "", cleanup, fmt.Errorf("failed to download plugin: %w", err)
	}

	// Extract archive
	extractDir := filepath.Join(pluginDir, "extracted")
	if err := extractTarGz(archivePath, extractDir); err != nil {
		return "", cleanup, fmt.Errorf("failed to extract plugin: %w", err)
	}

	return extractDir, cleanup, nil
}

func (v *Verifier) downloadFile(ctx context.Context, url, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (v *Verifier) updateVerificationStatus(ctx context.Context, verificationID int64, status string) error {
	var query string
	if status == "in_progress" {
		query = `UPDATE plugin_verifications SET status = $1, started_at = CURRENT_TIMESTAMP WHERE id = $2`
	} else {
		query = `UPDATE plugin_verifications SET status = $1 WHERE id = $2`
	}

	_, err := v.db.ExecContext(ctx, query, status, verificationID)
	return err
}

func (v *Verifier) completeVerification(ctx context.Context, result *VerificationResult) error {
	query := `
		UPDATE plugin_verifications
		SET status = $1,
		    security_level = $2,
		    completed_at = CURRENT_TIMESTAMP,
		    reason = $3
		WHERE id = $4
	`

	_, err := v.db.ExecContext(ctx, query,
		result.Status,
		sql.NullString{String: result.SecurityLevel, Valid: result.SecurityLevel != ""},
		sql.NullString{String: result.Reason, Valid: result.Reason != ""},
		result.VerificationID,
	)

	if err != nil {
		return fmt.Errorf("failed to complete verification: %w", err)
	}

	v.recordAuditLog(ctx, result.VerificationID, "completed", "system",
		fmt.Sprintf("Verification completed with status: %s", result.Status))

	return nil
}

func (v *Verifier) storeValidationError(ctx context.Context, verificationID int64, err ValidationError) error {
	query := `
		INSERT INTO plugin_validation_errors (verification_id, field, message, severity)
		VALUES ($1, $2, $3, $4)
	`
	_, dbErr := v.db.ExecContext(ctx, query, verificationID, err.Field, err.Message, err.Severity)
	return dbErr
}

func (v *Verifier) storeSecurityIssue(ctx context.Context, verificationID int64, issue SecurityIssue) error {
	query := `
		INSERT INTO plugin_security_issues
		(verification_id, severity, category, description, file, line_number, recommendation, cwe_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := v.db.ExecContext(ctx, query,
		verificationID,
		issue.Severity,
		issue.Category,
		issue.Description,
		sql.NullString{String: issue.File, Valid: issue.File != ""},
		sql.NullInt64{Int64: int64(issue.Line), Valid: issue.Line > 0},
		sql.NullString{String: issue.Recommendation, Valid: issue.Recommendation != ""},
		sql.NullString{String: issue.CWEID, Valid: issue.CWEID != ""},
	)
	return err
}

func (v *Verifier) loadValidationErrors(ctx context.Context, verificationID int64) ([]ValidationError, error) {
	query := `SELECT field, message, severity FROM plugin_validation_errors WHERE verification_id = $1`
	rows, err := v.db.QueryContext(ctx, query, verificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []ValidationError
	for rows.Next() {
		var err ValidationError
		if scanErr := rows.Scan(&err.Field, &err.Message, &err.Severity); scanErr == nil {
			errors = append(errors, err)
		}
	}
	return errors, nil
}

func (v *Verifier) loadSecurityIssues(ctx context.Context, verificationID int64) ([]SecurityIssue, error) {
	query := `
		SELECT severity, category, description, file, line_number, recommendation, cwe_id
		FROM plugin_security_issues
		WHERE verification_id = $1
	`
	rows, err := v.db.QueryContext(ctx, query, verificationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []SecurityIssue
	for rows.Next() {
		var issue SecurityIssue
		var file, recommendation, cweID sql.NullString
		var lineNumber sql.NullInt64

		err := rows.Scan(&issue.Severity, &issue.Category, &issue.Description,
			&file, &lineNumber, &recommendation, &cweID)
		if err != nil {
			continue
		}

		if file.Valid {
			issue.File = file.String
		}
		if lineNumber.Valid {
			issue.Line = int(lineNumber.Int64)
		}
		if recommendation.Valid {
			issue.Recommendation = recommendation.String
		}
		if cweID.Valid {
			issue.CWEID = cweID.String
		}

		issues = append(issues, issue)
	}
	return issues, nil
}

func (v *Verifier) recordAuditLog(ctx context.Context, verificationID int64, action, actor, details string) {
	query := `
		INSERT INTO plugin_verification_audit (verification_id, action, actor, details)
		VALUES ($1, $2, $3, $4)
	`
	v.db.ExecContext(ctx, query, verificationID, action, actor, details)
}

func (v *Verifier) recordScanStart(ctx context.Context, pluginID, version, scanType string) (int64, error) {
	query := `
		INSERT INTO plugin_scan_history (plugin_id, version, scan_type, status, started_at)
		VALUES ($1, $2, $3, 'in_progress', CURRENT_TIMESTAMP)
	`
	result, err := v.db.ExecContext(ctx, query, pluginID, version, scanType)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (v *Verifier) recordScanComplete(ctx context.Context, scanID int64, status string, issuesFound, criticalIssues int, errorMsg string) {
	query := `
		UPDATE plugin_scan_history
		SET status = $1,
		    issues_found = $2,
		    critical_issues = $3,
		    error_message = $4,
		    completed_at = CURRENT_TIMESTAMP,
		    scan_duration_ms = EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - started_at)) * 1000
		WHERE id = $5
	`
	v.db.ExecContext(ctx, query, status, issuesFound, criticalIssues,
		sql.NullString{String: errorMsg, Valid: errorMsg != ""}, scanID)
}

// extractTarGz extracts a tar.gz archive (stub implementation)
func extractTarGz(archivePath, destDir string) error {
	// TODO: Implement tar.gz extraction
	// For now, assume plugin is already in correct format
	os.MkdirAll(destDir, 0755)
	return nil
}

// GetDB returns the underlying database connection (for API handlers)
func (v *Verifier) GetDB() *sql.DB {
	return v.db
}
