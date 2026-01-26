package marketplace

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/platinummonkey/spoke/pkg/plugins"
)

// Service provides marketplace operations
type Service struct {
	db      *sql.DB
	storage Storage
}

// NewService creates a new marketplace service
func NewService(db *sql.DB, storage Storage) *Service {
	return &Service{
		db:      db,
		storage: storage,
	}
}

// ListPlugins lists plugins with optional filters
func (s *Service) ListPlugins(ctx context.Context, req *PluginListRequest) (*PluginListResponse, error) {
	// Build query
	query := `
		SELECT
			p.id, p.name, p.description, p.author, p.license, p.homepage,
			p.repository, p.type, p.security_level, p.enabled,
			p.created_at, p.updated_at, p.verified_at, p.verified_by, p.download_count,
			(SELECT version FROM plugin_versions WHERE plugin_id = p.id ORDER BY created_at DESC LIMIT 1) as latest_version,
			COALESCE(AVG(r.rating), 0) as avg_rating,
			COUNT(DISTINCT r.id) as review_count
		FROM plugins p
		LEFT JOIN plugin_reviews r ON p.id = r.plugin_id
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if req.Type != "" {
		query += fmt.Sprintf(" AND p.type = $%d", argCount)
		args = append(args, req.Type)
		argCount++
	}

	if req.SecurityLevel != "" {
		query += fmt.Sprintf(" AND p.security_level = $%d", argCount)
		args = append(args, req.SecurityLevel)
		argCount++
	}

	if req.Search != "" {
		query += fmt.Sprintf(" AND (p.name ILIKE $%d OR p.description ILIKE $%d)", argCount, argCount+1)
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argCount += 2
	}

	// Add tag filter if specified
	if len(req.Tags) > 0 {
		query += fmt.Sprintf(` AND p.id IN (
			SELECT plugin_id FROM plugin_tags WHERE tag IN (%s)
		)`, strings.Repeat("?,", len(req.Tags)-1)+"?")
		for _, tag := range req.Tags {
			args = append(args, tag)
		}
	}

	query += " GROUP BY p.id"

	// Apply sorting
	sortBy := "p.created_at"
	if req.SortBy == "downloads" {
		sortBy = "p.download_count"
	} else if req.SortBy == "rating" {
		sortBy = "avg_rating"
	}

	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Apply pagination
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, req.Limit, req.Offset)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query plugins: %w", err)
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		var p Plugin
		err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Author, &p.License, &p.Homepage,
			&p.Repository, &p.Type, &p.SecurityLevel, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt, &p.VerifiedAt, &p.VerifiedBy, &p.DownloadCount,
			&p.LatestVersion, &p.AvgRating, &p.ReviewCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin: %w", err)
		}
		plugins = append(plugins, p)
	}

	// Get total count
	countQuery := "SELECT COUNT(*) FROM plugins WHERE 1=1"
	countArgs := []interface{}{}

	if req.Type != "" {
		countQuery += " AND type = ?"
		countArgs = append(countArgs, req.Type)
	}

	if req.SecurityLevel != "" {
		countQuery += " AND security_level = ?"
		countArgs = append(countArgs, req.SecurityLevel)
	}

	var total int64
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count plugins: %w", err)
	}

	return &PluginListResponse{
		Plugins: plugins,
		Total:   total,
		Limit:   req.Limit,
		Offset:  req.Offset,
	}, nil
}

// GetPlugin retrieves a plugin by ID
func (s *Service) GetPlugin(ctx context.Context, pluginID string) (*Plugin, error) {
	query := `
		SELECT
			p.id, p.name, p.description, p.author, p.license, p.homepage,
			p.repository, p.type, p.security_level, p.enabled,
			p.created_at, p.updated_at, p.verified_at, p.verified_by, p.download_count,
			(SELECT version FROM plugin_versions WHERE plugin_id = p.id ORDER BY created_at DESC LIMIT 1) as latest_version,
			COALESCE(AVG(r.rating), 0) as avg_rating,
			COUNT(DISTINCT r.id) as review_count
		FROM plugins p
		LEFT JOIN plugin_reviews r ON p.id = r.plugin_id
		WHERE p.id = ?
		GROUP BY p.id
	`

	var p Plugin
	err := s.db.QueryRowContext(ctx, query, pluginID).Scan(
		&p.ID, &p.Name, &p.Description, &p.Author, &p.License, &p.Homepage,
		&p.Repository, &p.Type, &p.SecurityLevel, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt, &p.VerifiedAt, &p.VerifiedBy, &p.DownloadCount,
		&p.LatestVersion, &p.AvgRating, &p.ReviewCount,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	return &p, nil
}

// CreatePlugin creates a new plugin
func (s *Service) CreatePlugin(ctx context.Context, plugin *Plugin) error {
	// Validate plugin
	if err := s.validatePlugin(plugin); err != nil {
		return err
	}

	query := `
		INSERT INTO plugins (
			id, name, description, author, license, homepage, repository,
			type, security_level, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		plugin.ID, plugin.Name, plugin.Description, plugin.Author, plugin.License,
		plugin.Homepage, plugin.Repository, plugin.Type, plugin.SecurityLevel,
		plugin.Enabled, now, now,
	)

	if err != nil {
		return fmt.Errorf("failed to create plugin: %w", err)
	}

	return nil
}

// CreatePluginVersion creates a new plugin version
func (s *Service) CreatePluginVersion(ctx context.Context, version *PluginVersion) error {
	query := `
		INSERT INTO plugin_versions (
			plugin_id, version, api_version, manifest_url, download_url,
			checksum, size_bytes, downloads, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		version.PluginID, version.Version, version.APIVersion, version.ManifestURL,
		version.DownloadURL, version.Checksum, version.SizeBytes, 0, time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create plugin version: %w", err)
	}

	return nil
}

// ListPluginVersions lists all versions of a plugin
func (s *Service) ListPluginVersions(ctx context.Context, pluginID string) ([]PluginVersion, error) {
	query := `
		SELECT id, plugin_id, version, api_version, manifest_url, download_url,
		       checksum, size_bytes, downloads, created_at
		FROM plugin_versions
		WHERE plugin_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to query versions: %w", err)
	}
	defer rows.Close()

	var versions []PluginVersion
	for rows.Next() {
		var v PluginVersion
		err := rows.Scan(
			&v.ID, &v.PluginID, &v.Version, &v.APIVersion, &v.ManifestURL,
			&v.DownloadURL, &v.Checksum, &v.SizeBytes, &v.Downloads, &v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// RecordDownload increments download counters
func (s *Service) RecordDownload(ctx context.Context, pluginID, version string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Increment plugin download count
	_, err = tx.ExecContext(ctx, "UPDATE plugins SET download_count = download_count + 1 WHERE id = ?", pluginID)
	if err != nil {
		return fmt.Errorf("failed to update plugin download count: %w", err)
	}

	// Increment version download count
	_, err = tx.ExecContext(ctx, "UPDATE plugin_versions SET downloads = downloads + 1 WHERE plugin_id = ? AND version = ?", pluginID, version)
	if err != nil {
		return fmt.Errorf("failed to update version download count: %w", err)
	}

	return tx.Commit()
}

// CreateReview creates or updates a review
func (s *Service) CreateReview(ctx context.Context, review *PluginReview) error {
	// Validate rating
	if review.Rating < 1 || review.Rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	query := `
		INSERT INTO plugin_reviews (plugin_id, user_id, rating, review, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE rating = ?, review = ?, updated_at = ?
	`

	now := time.Now()
	_, err := s.db.ExecContext(ctx, query,
		review.PluginID, review.UserID, review.Rating, review.Review, now, now,
		review.Rating, review.Review, now,
	)

	if err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}

	return nil
}

// ListReviews lists reviews for a plugin
func (s *Service) ListReviews(ctx context.Context, pluginID string, limit, offset int) ([]PluginReview, error) {
	query := `
		SELECT id, plugin_id, user_id, rating, review, created_at, updated_at
		FROM plugin_reviews
		WHERE plugin_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, pluginID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}
	defer rows.Close()

	var reviews []PluginReview
	for rows.Next() {
		var r PluginReview
		err := rows.Scan(&r.ID, &r.PluginID, &r.UserID, &r.Rating, &r.Review, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review: %w", err)
		}
		reviews = append(reviews, r)
	}

	return reviews, nil
}

// validatePlugin validates plugin data
func (s *Service) validatePlugin(plugin *Plugin) error {
	if plugin.ID == "" {
		return fmt.Errorf("plugin ID is required")
	}

	if plugin.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if plugin.Author == "" {
		return fmt.Errorf("plugin author is required")
	}

	// Validate type
	validTypes := map[string]bool{
		string(plugins.PluginTypeLanguage):  true,
		string(plugins.PluginTypeValidator): true,
		string(plugins.PluginTypeGenerator): true,
		string(plugins.PluginTypeRunner):    true,
		string(plugins.PluginTypeTransform): true,
	}

	if !validTypes[plugin.Type] {
		return fmt.Errorf("invalid plugin type: %s", plugin.Type)
	}

	// Validate security level
	validSecurityLevels := map[string]bool{
		"official":  true,
		"verified":  true,
		"community": true,
	}

	if plugin.SecurityLevel == "" {
		plugin.SecurityLevel = "community"
	}

	if !validSecurityLevels[plugin.SecurityLevel] {
		return fmt.Errorf("invalid security level: %s", plugin.SecurityLevel)
	}

	return nil
}
