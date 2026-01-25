package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Checker handles permission checking and evaluation
type Checker interface {
	// CheckPermission checks if a user has a specific permission
	CheckPermission(ctx context.Context, check PermissionCheck) (*PermissionCheckResult, error)

	// GetUserRoles returns all roles assigned to a user
	GetUserRoles(ctx context.Context, userID int64, organizationID *int64) ([]Role, error)

	// GetEffectivePermissions returns all permissions for a user (including inherited)
	GetEffectivePermissions(ctx context.Context, userID int64, organizationID *int64, resourceID *string) ([]Permission, error)

	// InvalidateCache invalidates permission cache for a user
	InvalidateCache(ctx context.Context, userID int64) error
}

// PermissionChecker implements the Checker interface
type PermissionChecker struct {
	db        *sql.DB
	store     *Store
	cacheTTL  time.Duration
	cacheEnabled bool
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(db *sql.DB, cacheTTL time.Duration) *PermissionChecker {
	return &PermissionChecker{
		db:           db,
		store:        NewStore(db),
		cacheTTL:     cacheTTL,
		cacheEnabled: cacheTTL > 0,
	}
}

// CheckPermission checks if a user has a specific permission
func (pc *PermissionChecker) CheckPermission(ctx context.Context, check PermissionCheck) (*PermissionCheckResult, error) {
	// Check cache first
	if pc.cacheEnabled {
		cached, err := pc.getCachedPermission(ctx, check)
		if err == nil && cached != nil && cached.ExpiresAt.After(time.Now()) {
			return &PermissionCheckResult{
				Allowed:   cached.Allowed,
				Reason:    "cached result",
				CheckedAt: time.Now(),
			}, nil
		}
	}

	// Get user roles
	roles, err := pc.GetUserRoles(ctx, check.UserID, check.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Check if any role grants the permission
	var matchedRoles []string
	allowed := false

	for _, role := range roles {
		// Check if role has the required permission
		if pc.roleHasPermission(role, check.Permission) {
			// Check if the role applies to the requested scope
			if pc.roleScopeMatches(role, check) {
				allowed = true
				matchedRoles = append(matchedRoles, role.Name)
			}
		}
	}

	result := &PermissionCheckResult{
		Allowed:      allowed,
		MatchedRoles: matchedRoles,
		CheckedAt:    time.Now(),
	}

	if allowed {
		result.Reason = fmt.Sprintf("granted by roles: %v", matchedRoles)
	} else {
		result.Reason = "no matching role found"
	}

	// Cache the result
	if pc.cacheEnabled {
		pc.cachePermissionResult(ctx, check, result)
	}

	return result, nil
}

// GetUserRoles returns all roles assigned to a user
func (pc *PermissionChecker) GetUserRoles(ctx context.Context, userID int64, organizationID *int64) ([]Role, error) {
	// Get direct user role assignments
	userRoles, err := pc.store.GetUserRoles(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Get roles through team membership
	teamRoles, err := pc.getUserTeamRoles(ctx, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team roles: %w", err)
	}

	// Combine and deduplicate roles
	roleMap := make(map[int64]Role)

	for _, ur := range userRoles {
		role, err := pc.store.GetRole(ctx, ur.RoleID)
		if err != nil {
			continue
		}
		roleMap[role.ID] = *role
	}

	for _, role := range teamRoles {
		roleMap[role.ID] = role
	}

	// Resolve role inheritance
	finalRoles := make([]Role, 0, len(roleMap))
	for _, role := range roleMap {
		// Get parent roles recursively
		inheritedRoles, err := pc.resolveRoleInheritance(ctx, role)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve role inheritance: %w", err)
		}
		finalRoles = append(finalRoles, inheritedRoles...)
	}

	return finalRoles, nil
}

// GetEffectivePermissions returns all permissions for a user
func (pc *PermissionChecker) GetEffectivePermissions(ctx context.Context, userID int64, organizationID *int64, resourceID *string) ([]Permission, error) {
	roles, err := pc.GetUserRoles(ctx, userID, organizationID)
	if err != nil {
		return nil, err
	}

	// Collect all unique permissions
	permMap := make(map[string]Permission)
	for _, role := range roles {
		for _, perm := range role.Permissions {
			permMap[perm.String()] = perm
		}
	}

	permissions := make([]Permission, 0, len(permMap))
	for _, perm := range permMap {
		permissions = append(permissions, perm)
	}

	return permissions, nil
}

// InvalidateCache invalidates permission cache for a user
func (pc *PermissionChecker) InvalidateCache(ctx context.Context, userID int64) error {
	if !pc.cacheEnabled {
		return nil
	}

	query := `DELETE FROM permission_cache WHERE user_id = $1`
	_, err := pc.db.ExecContext(ctx, query, userID)
	return err
}

// roleHasPermission checks if a role has a specific permission
func (pc *PermissionChecker) roleHasPermission(role Role, permission Permission) bool {
	for _, p := range role.Permissions {
		if p.Resource == permission.Resource && p.Action == permission.Action {
			return true
		}
	}
	return false
}

// roleScopeMatches checks if a role's scope matches the permission check
func (pc *PermissionChecker) roleScopeMatches(role Role, check PermissionCheck) bool {
	// For now, simplified scope matching
	// In a full implementation, this would check:
	// - Organization-level roles apply to all resources in that org
	// - Module-level roles apply to specific modules
	// - Global roles apply everywhere
	return true
}

// getUserTeamRoles gets roles assigned through team membership
func (pc *PermissionChecker) getUserTeamRoles(ctx context.Context, userID int64, organizationID *int64) ([]Role, error) {
	query := `
		SELECT DISTINCT r.id, r.name, r.display_name, r.description, r.organization_id,
		       r.permissions, r.parent_role_id, r.is_built_in, r.is_custom, r.created_at, r.updated_at, r.created_by
		FROM roles r
		JOIN team_roles tr ON r.id = tr.role_id
		JOIN team_members tm ON tr.team_id = tm.team_id
		WHERE tm.user_id = $1
	`

	args := []interface{}{userID}
	if organizationID != nil {
		query += ` AND (r.organization_id = $2 OR r.organization_id IS NULL)`
		args = append(args, *organizationID)
	}

	rows, err := pc.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *role)
	}

	return roles, rows.Err()
}

// resolveRoleInheritance recursively resolves role inheritance
func (pc *PermissionChecker) resolveRoleInheritance(ctx context.Context, role Role) ([]Role, error) {
	roles := []Role{role}

	// If role has a parent, recursively get parent roles
	if role.ParentRoleID != nil {
		parentRole, err := pc.store.GetRole(ctx, *role.ParentRoleID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent role: %w", err)
		}

		parentRoles, err := pc.resolveRoleInheritance(ctx, *parentRole)
		if err != nil {
			return nil, err
		}

		roles = append(roles, parentRoles...)
	}

	return roles, nil
}

// getCachedPermission retrieves a cached permission check result
func (pc *PermissionChecker) getCachedPermission(ctx context.Context, check PermissionCheck) (*PermissionCacheEntry, error) {
	query := `
		SELECT user_id, permission, scope, resource_id, organization_id, allowed, expires_at, created_at
		FROM permission_cache
		WHERE user_id = $1
		  AND permission = $2
		  AND scope = $3
		  AND (resource_id = $4 OR (resource_id IS NULL AND $4 IS NULL))
		  AND (organization_id = $5 OR (organization_id IS NULL AND $5 IS NULL))
		  AND expires_at > NOW()
	`

	var entry PermissionCacheEntry
	err := pc.db.QueryRowContext(ctx, query,
		check.UserID,
		check.Permission.String(),
		check.Scope,
		check.ResourceID,
		check.OrganizationID,
	).Scan(
		&entry.UserID,
		&entry.Permission,
		&entry.Scope,
		&entry.ResourceID,
		&entry.OrganizationID,
		&entry.Allowed,
		&entry.ExpiresAt,
		&entry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// cachePermissionResult caches a permission check result
func (pc *PermissionChecker) cachePermissionResult(ctx context.Context, check PermissionCheck, result *PermissionCheckResult) {
	query := `
		INSERT INTO permission_cache (user_id, permission, scope, resource_id, organization_id, allowed, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id, permission, scope, COALESCE(resource_id, ''), COALESCE(organization_id, 0))
		DO UPDATE SET allowed = EXCLUDED.allowed, expires_at = EXCLUDED.expires_at, created_at = EXCLUDED.created_at
	`

	expiresAt := time.Now().Add(pc.cacheTTL)
	_, _ = pc.db.ExecContext(ctx, query,
		check.UserID,
		check.Permission.String(),
		check.Scope,
		check.ResourceID,
		check.OrganizationID,
		result.Allowed,
		expiresAt,
		time.Now(),
	)
}

// scanRole scans a role from a database row
func scanRole(scanner interface {
	Scan(dest ...interface{}) error
}) (*Role, error) {
	var role Role
	var permissionsJSON string
	var parentRoleID sql.NullInt64
	var orgID sql.NullInt64
	var createdBy sql.NullInt64

	err := scanner.Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&role.Description,
		&orgID,
		&permissionsJSON,
		&parentRoleID,
		&role.IsBuiltIn,
		&role.IsCustom,
		&role.CreatedAt,
		&role.UpdatedAt,
		&createdBy,
	)

	if err != nil {
		return nil, err
	}

	if parentRoleID.Valid {
		roleID := parentRoleID.Int64
		role.ParentRoleID = &roleID
	}

	if orgID.Valid {
		oID := orgID.Int64
		role.OrganizationID = &oID
	}

	if createdBy.Valid {
		cID := createdBy.Int64
		role.CreatedBy = &cID
	}

	// Parse permissions JSON
	if permissionsJSON != "" {
		if err := json.Unmarshal([]byte(permissionsJSON), &role.Permissions); err != nil {
			// If JSON parsing fails, log and continue with empty permissions
			role.Permissions = []Permission{}
		}
	} else {
		role.Permissions = []Permission{}
	}

	return &role, nil
}
