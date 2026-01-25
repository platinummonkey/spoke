package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/audit"
)

// Config holds RBAC configuration
type Config struct {
	// CacheTTL is how long to cache permission checks
	CacheTTL time.Duration

	// EnableTeams enables team functionality
	EnableTeams bool

	// EnableRoleInheritance enables role inheritance
	EnableRoleInheritance bool
}

// DefaultConfig returns default RBAC configuration
func DefaultConfig() Config {
	return Config{
		CacheTTL:              5 * time.Minute,
		EnableTeams:           true,
		EnableRoleInheritance: true,
	}
}

// Manager manages all RBAC components
type Manager struct {
	store       *Store
	checker     *PermissionChecker
	handlers    *Handlers
	middleware  *PermissionMiddleware
	config      Config
}

// NewManager creates a new RBAC manager
func NewManager(db *sql.DB, auditLogger audit.Logger, config Config) *Manager {
	store := NewStore(db)
	checker := NewPermissionChecker(db, config.CacheTTL)
	handlers := NewHandlers(db, auditLogger)
	middleware := NewPermissionMiddleware(checker)

	return &Manager{
		store:      store,
		checker:    checker,
		handlers:   handlers,
		middleware: middleware,
		config:     config,
	}
}

// Initialize sets up RBAC system
func (m *Manager) Initialize(ctx context.Context) error {
	// Run migrations
	if err := RunMigrations(ctx, m.store.db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize built-in roles
	if err := InitializeBuiltInRoles(ctx, m.store); err != nil {
		return fmt.Errorf("failed to initialize built-in roles: %w", err)
	}

	return nil
}

// RegisterRoutes registers RBAC routes with a router
func (m *Manager) RegisterRoutes(router *mux.Router) {
	m.handlers.RegisterRoutes(router)
}

// GetStore returns the RBAC store
func (m *Manager) GetStore() *Store {
	return m.store
}

// GetChecker returns the permission checker
func (m *Manager) GetChecker() *PermissionChecker {
	return m.checker
}

// GetMiddleware returns the permission middleware
func (m *Manager) GetMiddleware() *PermissionMiddleware {
	return m.middleware
}

// CheckPermission is a convenience method for checking permissions
func (m *Manager) CheckPermission(ctx context.Context, userID int64, resource Resource, action Action, scope PermissionScope, resourceID *string, organizationID *int64) (bool, error) {
	check := PermissionCheck{
		UserID: userID,
		Permission: Permission{
			Resource: resource,
			Action:   action,
		},
		Scope:          scope,
		ResourceID:     resourceID,
		OrganizationID: organizationID,
	}

	result, err := m.checker.CheckPermission(ctx, check)
	if err != nil {
		return false, err
	}

	return result.Allowed, nil
}

// AssignRoleToUser is a convenience method for assigning roles
func (m *Manager) AssignRoleToUser(ctx context.Context, userID, roleID int64, scope PermissionScope, resourceID *string, organizationID *int64, grantedBy *int64, expiresAt *time.Time) error {
	userRole := &UserRole{
		UserID:         userID,
		RoleID:         roleID,
		Scope:          scope,
		ResourceID:     resourceID,
		OrganizationID: organizationID,
		GrantedBy:      grantedBy,
		ExpiresAt:      expiresAt,
	}

	if err := m.store.AssignRoleToUser(ctx, userRole); err != nil {
		return err
	}

	// Invalidate cache
	return m.checker.InvalidateCache(ctx, userID)
}

// CreateCustomRole is a convenience method for creating custom roles
func (m *Manager) CreateCustomRole(ctx context.Context, name, displayName, description string, permissions []Permission, organizationID *int64, parentRoleID *int64, createdBy *int64) (*Role, error) {
	role := &Role{
		Name:           name,
		DisplayName:    displayName,
		Description:    description,
		OrganizationID: organizationID,
		Permissions:    permissions,
		ParentRoleID:   parentRoleID,
		IsBuiltIn:      false,
		IsCustom:       true,
		CreatedBy:      createdBy,
	}

	if err := m.store.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

// GetUserRoles returns all roles for a user
func (m *Manager) GetUserRoles(ctx context.Context, userID int64, organizationID *int64) ([]Role, error) {
	return m.checker.GetUserRoles(ctx, userID, organizationID)
}

// GetEffectivePermissions returns all effective permissions for a user
func (m *Manager) GetEffectivePermissions(ctx context.Context, userID int64, organizationID *int64, resourceID *string) ([]Permission, error) {
	return m.checker.GetEffectivePermissions(ctx, userID, organizationID, resourceID)
}

// CreateTeam creates a new team
func (m *Manager) CreateTeam(ctx context.Context, organizationID int64, name, displayName, description string, createdBy *int64) (*Team, error) {
	if !m.config.EnableTeams {
		return nil, fmt.Errorf("teams are not enabled")
	}

	team := &Team{
		OrganizationID: organizationID,
		Name:           name,
		DisplayName:    displayName,
		Description:    description,
		CreatedBy:      createdBy,
	}

	if err := m.store.CreateTeam(ctx, team); err != nil {
		return nil, err
	}

	return team, nil
}

// AddTeamMember adds a user to a team
func (m *Manager) AddTeamMember(ctx context.Context, teamID, userID int64, roleID *int64, addedBy *int64) error {
	if !m.config.EnableTeams {
		return fmt.Errorf("teams are not enabled")
	}

	member := &TeamMember{
		TeamID:  teamID,
		UserID:  userID,
		RoleID:  roleID,
		AddedBy: addedBy,
	}

	if err := m.store.AddTeamMember(ctx, member); err != nil {
		return err
	}

	// Invalidate cache for user
	return m.checker.InvalidateCache(ctx, userID)
}

// AssignRoleToTeam assigns a role to a team
func (m *Manager) AssignRoleToTeam(ctx context.Context, teamID, roleID int64, scope PermissionScope, resourceID *string, organizationID *int64, grantedBy *int64) error {
	if !m.config.EnableTeams {
		return fmt.Errorf("teams are not enabled")
	}

	teamRole := &TeamRole{
		TeamID:         teamID,
		RoleID:         roleID,
		Scope:          scope,
		ResourceID:     resourceID,
		OrganizationID: organizationID,
		GrantedBy:      grantedBy,
	}

	if err := m.store.AssignRoleToTeam(ctx, teamRole); err != nil {
		return err
	}

	// Invalidate cache for all team members
	members, err := m.store.GetTeamMembers(ctx, teamID)
	if err != nil {
		return err
	}

	for _, member := range members {
		m.checker.InvalidateCache(ctx, member.UserID)
	}

	return nil
}

// BootstrapOrganization sets up default roles and admin user for a new organization
func (m *Manager) BootstrapOrganization(ctx context.Context, organizationID, adminUserID int64) error {
	// Get org admin role
	adminRole, err := m.store.GetRoleByName(ctx, RoleOrgAdmin, &organizationID)
	if err != nil {
		// Role doesn't exist, use built-in
		adminRole, err = m.store.GetRoleByName(ctx, RoleOrgAdmin, nil)
		if err != nil {
			return fmt.Errorf("failed to get admin role: %w", err)
		}
	}

	// Assign admin role to user
	userRole := &UserRole{
		UserID:         adminUserID,
		RoleID:         adminRole.ID,
		Scope:          ScopeOrganization,
		OrganizationID: &organizationID,
		GrantedBy:      &adminUserID, // Self-granted during bootstrap
	}

	if err := m.store.AssignRoleToUser(ctx, userRole); err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}

// BootstrapModule sets up default module owner when a module is created
func (m *Manager) BootstrapModule(ctx context.Context, moduleName string, ownerUserID int64, organizationID *int64) error {
	// Get module owner role
	ownerRole, err := m.store.GetRoleByName(ctx, RoleModuleOwner, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get module owner role: %w", err)
	}

	// Assign module owner role
	userRole := &UserRole{
		UserID:         ownerUserID,
		RoleID:         ownerRole.ID,
		Scope:          ScopeModule,
		ResourceID:     &moduleName,
		OrganizationID: organizationID,
		GrantedBy:      &ownerUserID, // Self-granted during bootstrap
	}

	if err := m.store.AssignRoleToUser(ctx, userRole); err != nil {
		return fmt.Errorf("failed to assign module owner role: %w", err)
	}

	return nil
}

// Stats returns statistics about the RBAC system
type Stats struct {
	TotalRoles       int64
	CustomRoles      int64
	TotalUserRoles   int64
	TotalTeams       int64
	TotalTeamMembers int64
	CachedPermissions int64
}

// GetStats returns RBAC statistics
func (m *Manager) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{}

	// Count roles
	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles").Scan(&stats.TotalRoles); err != nil {
		return nil, err
	}

	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM roles WHERE is_custom = true").Scan(&stats.CustomRoles); err != nil {
		return nil, err
	}

	// Count user roles
	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_roles").Scan(&stats.TotalUserRoles); err != nil {
		return nil, err
	}

	// Count teams
	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM teams").Scan(&stats.TotalTeams); err != nil {
		return nil, err
	}

	// Count team members
	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM team_members").Scan(&stats.TotalTeamMembers); err != nil {
		return nil, err
	}

	// Count cached permissions
	if err := m.store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM permission_cache WHERE expires_at > CURRENT_TIMESTAMP").Scan(&stats.CachedPermissions); err != nil {
		return nil, err
	}

	return stats, nil
}

// CleanupExpiredPermissions removes expired permission cache entries
func (m *Manager) CleanupExpiredPermissions(ctx context.Context) (int64, error) {
	result, err := m.store.db.ExecContext(ctx, "DELETE FROM permission_cache WHERE expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
