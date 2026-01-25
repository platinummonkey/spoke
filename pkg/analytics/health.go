package analytics

import (
	"context"
	"database/sql"
	"math"
)

// HealthScorer calculates schema health metrics
type HealthScorer struct {
	db *sql.DB
}

// NewHealthScorer creates a new health scorer
func NewHealthScorer(db *sql.DB) *HealthScorer {
	return &HealthScorer{db: db}
}

// ModuleHealth represents schema health assessment
type ModuleHealth struct {
	ModuleName           string   `json:"module_name"`
	Version              string   `json:"version"`
	HealthScore          float64  `json:"health_score"`           // 0-100 (higher is better)
	ComplexityScore      float64  `json:"complexity_score"`       // 0-100 (lower is better)
	MaintainabilityIndex float64  `json:"maintainability_index"`  // 0-100 (higher is better)
	UnusedFields         []string `json:"unused_fields"`
	DeprecatedFieldCount int      `json:"deprecated_field_count"`
	BreakingChanges30d   int      `json:"breaking_changes_30d"`
	DependentsCount      int      `json:"dependents_count"`
	Recommendations      []string `json:"recommendations"`
}

// CalculateHealth computes health metrics for a module version
func (h *HealthScorer) CalculateHealth(ctx context.Context, moduleName, version string) (*ModuleHealth, error) {
	health := &ModuleHealth{
		ModuleName: moduleName,
		Version:    version,
	}

	// Get version ID
	versionID, err := h.getVersionID(ctx, moduleName, version)
	if err != nil {
		return nil, err
	}

	// Calculate complexity score
	health.ComplexityScore, err = h.calculateComplexity(ctx, versionID)
	if err != nil {
		return nil, err
	}

	// Find unused fields
	health.UnusedFields, err = h.findUnusedFields(ctx, moduleName, version)
	if err != nil {
		return nil, err
	}

	// Count deprecated fields
	health.DeprecatedFieldCount, err = h.countDeprecatedFields(ctx, versionID)
	if err != nil {
		return nil, err
	}

	// Count breaking changes in last 30 days
	health.BreakingChanges30d, err = h.countRecentBreakingChanges(ctx, moduleName)
	if err != nil {
		return nil, err
	}

	// Count dependents
	health.DependentsCount, err = h.countDependents(ctx, moduleName, version)
	if err != nil {
		return nil, err
	}

	// Calculate maintainability index
	health.MaintainabilityIndex = h.calculateMaintainability(health)

	// Calculate overall health score
	health.HealthScore = h.calculateOverallHealth(health)

	// Generate recommendations
	health.Recommendations = h.generateRecommendations(health)

	return health, nil
}

// getVersionID retrieves the version ID for a module version
func (h *HealthScorer) getVersionID(ctx context.Context, moduleName, version string) (int64, error) {
	var versionID int64
	query := `
		SELECT v.id
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1 AND v.version = $2
	`
	err := h.db.QueryRowContext(ctx, query, moduleName, version).Scan(&versionID)
	if err != nil {
		return 0, err
	}
	return versionID, nil
}

// calculateComplexity measures schema complexity (0-100, lower is better)
func (h *HealthScorer) calculateComplexity(ctx context.Context, versionID int64) (float64, error) {
	var (
		messageCount int
		enumCount    int
		serviceCount int
		fieldCount   int
		methodCount  int
	)

	query := `
		SELECT
			COUNT(*) FILTER (WHERE entity_type = 'message') AS message_count,
			COUNT(*) FILTER (WHERE entity_type = 'enum') AS enum_count,
			COUNT(*) FILTER (WHERE entity_type = 'service') AS service_count,
			COUNT(*) FILTER (WHERE entity_type = 'field') AS field_count,
			COUNT(*) FILTER (WHERE entity_type = 'method') AS method_count
		FROM proto_search_index
		WHERE version_id = $1
	`

	err := h.db.QueryRowContext(ctx, query, versionID).Scan(
		&messageCount, &enumCount, &serviceCount, &fieldCount, &methodCount,
	)
	if err != nil {
		return 0, err
	}

	// Simple complexity calculation
	totalEntities := messageCount + enumCount + serviceCount
	avgFieldsPerMessage := 0.0
	if messageCount > 0 {
		avgFieldsPerMessage = float64(fieldCount) / float64(messageCount)
	}

	// Normalize to 0-100 scale
	// Assume: <10 entities = low (0-20), 10-50 = medium (20-70), >50 = high (70-100)
	entityComplexity := math.Min(float64(totalEntities)/50.0*100, 100)

	// Assume: <5 fields/msg = low, 5-15 = medium, >15 = high
	fieldComplexity := math.Min(avgFieldsPerMessage/15.0*100, 100)

	// Weighted average (entities 60%, field density 40%)
	complexity := 0.6*entityComplexity + 0.4*fieldComplexity

	return complexity, nil
}

// findUnusedFields identifies fields with very low usage
func (h *HealthScorer) findUnusedFields(ctx context.Context, moduleName, version string) ([]string, error) {
	// Check bookmarks for field-level usage
	// Fields never bookmarked in last 90 days = potentially unused
	query := `
		SELECT DISTINCT psi.entity_name
		FROM proto_search_index psi
		JOIN versions v ON psi.version_id = v.id
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1
		  AND v.version = $2
		  AND psi.entity_type = 'field'
		  AND NOT EXISTS (
			  SELECT 1 FROM bookmarks b
			  WHERE b.module_name = m.name
				AND b.version = v.version
				AND b.entity_path LIKE '%' || psi.entity_name || '%'
				AND b.created_at >= NOW() - INTERVAL '90 days'
		  )
		  AND NOT EXISTS (
			  SELECT 1 FROM search_history sh
			  WHERE sh.query LIKE '%' || psi.entity_name || '%'
				AND sh.searched_at >= NOW() - INTERVAL '90 days'
		  )
		ORDER BY psi.entity_name
		LIMIT 20
	`

	rows, err := h.db.QueryContext(ctx, query, moduleName, version)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unused []string
	for rows.Next() {
		var fieldName string
		if err := rows.Scan(&fieldName); err != nil {
			return nil, err
		}
		unused = append(unused, fieldName)
	}

	return unused, nil
}

// countDeprecatedFields counts fields marked as deprecated in proto
func (h *HealthScorer) countDeprecatedFields(ctx context.Context, versionID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM proto_search_index
		WHERE version_id = $1
		  AND entity_type = 'field'
		  AND (
			  metadata->>'deprecated' = 'true'
			  OR description ILIKE '%deprecated%'
		  )
	`

	var count int
	err := h.db.QueryRowContext(ctx, query, versionID).Scan(&count)
	return count, err
}

// countRecentBreakingChanges counts breaking changes in last 30 days
func (h *HealthScorer) countRecentBreakingChanges(ctx context.Context, moduleName string) (int, error) {
	// Check audit logs for breaking change events or version bumps
	query := `
		SELECT COUNT(*)
		FROM versions v
		JOIN modules m ON v.module_id = m.id
		WHERE m.name = $1
		  AND v.created_at >= NOW() - INTERVAL '30 days'
		  AND (
			  -- Major version change (e.g., v1.x.x -> v2.x.x)
			  v.version ~ '^v?[0-9]+\.0\.0$'
		  )
	`

	var count int
	err := h.db.QueryRowContext(ctx, query, moduleName).Scan(&count)
	return count, err
}

// countDependents counts modules depending on this module
func (h *HealthScorer) countDependents(ctx context.Context, moduleName, version string) (int, error) {
	// Count versions that have this module in their dependencies
	query := `
		SELECT COUNT(DISTINCT v.id)
		FROM versions v
		WHERE v.dependencies @> jsonb_build_array($1)
	`

	dependency := moduleName + "@" + version

	var count int
	err := h.db.QueryRowContext(ctx, query, dependency).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

// calculateMaintainability computes maintainability index (0-100, higher is better)
func (h *HealthScorer) calculateMaintainability(health *ModuleHealth) float64 {
	score := 100.0

	// Penalty for high complexity (max 30 points)
	score -= health.ComplexityScore * 0.3

	// Penalty for unused fields (2 points each, max 20 points)
	unusedPenalty := math.Min(float64(len(health.UnusedFields))*2, 20)
	score -= unusedPenalty

	// Penalty for deprecated fields (3 points each, max 15 points)
	deprecatedPenalty := math.Min(float64(health.DeprecatedFieldCount)*3, 15)
	score -= deprecatedPenalty

	// Penalty for breaking changes (5 points each, max 15 points)
	breakingPenalty := math.Min(float64(health.BreakingChanges30d)*5, 15)
	score -= breakingPenalty

	return math.Max(score, 0)
}

// calculateOverallHealth computes overall health score (0-100, higher is better)
func (h *HealthScorer) calculateOverallHealth(health *ModuleHealth) float64 {
	// Weighted average of factors
	weights := map[string]float64{
		"complexity":      0.25,
		"maintainability": 0.35,
		"unused":          0.15,
		"deprecated":      0.10,
		"breaking":        0.15,
	}

	// Invert complexity (lower is better -> higher score)
	complexityScore := 100 - health.ComplexityScore

	// Unused fields score (fewer is better)
	unusedScore := math.Max(100-float64(len(health.UnusedFields))*2, 0)

	// Deprecated fields score (fewer is better)
	deprecatedScore := math.Max(100-float64(health.DeprecatedFieldCount)*3, 0)

	// Breaking changes score (fewer is better)
	breakingScore := math.Max(100-float64(health.BreakingChanges30d)*5, 0)

	// Weighted sum
	score := weights["complexity"]*complexityScore +
		weights["maintainability"]*health.MaintainabilityIndex +
		weights["unused"]*unusedScore +
		weights["deprecated"]*deprecatedScore +
		weights["breaking"]*breakingScore

	return math.Round(score*10) / 10
}

// generateRecommendations creates actionable suggestions
func (h *HealthScorer) generateRecommendations(health *ModuleHealth) []string {
	var recommendations []string

	if health.ComplexityScore > 70 {
		recommendations = append(recommendations,
			"Consider splitting this module into smaller, focused modules to reduce complexity.")
	}

	if len(health.UnusedFields) > 5 {
		recommendations = append(recommendations,
			"Remove unused fields to simplify the schema and reduce maintenance burden.")
	}

	if health.DeprecatedFieldCount > 3 {
		recommendations = append(recommendations,
			"Remove deprecated fields in the next major version to clean up technical debt.")
	}

	if health.BreakingChanges30d > 2 {
		recommendations = append(recommendations,
			"Frequent breaking changes detected. Consider backward-compatible changes or better versioning.")
	}

	if health.DependentsCount > 10 && health.BreakingChanges30d > 0 {
		recommendations = append(recommendations,
			"This module has many dependents. Breaking changes require careful coordination.")
	}

	if health.HealthScore > 80 {
		recommendations = append(recommendations,
			"Schema health is excellent! Keep following protobuf best practices.")
	} else if health.HealthScore < 50 {
		recommendations = append(recommendations,
			"Schema health needs attention. Review the metrics above and prioritize improvements.")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"Schema health is good. Continue monitoring and maintaining best practices.")
	}

	return recommendations
}
