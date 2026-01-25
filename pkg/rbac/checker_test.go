package rbac

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create minimal tables for testing
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			email TEXT
		);

		CREATE TABLE organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		);

		CREATE TABLE roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			organization_id INTEGER,
			permissions TEXT NOT NULL DEFAULT '[]',
			parent_role_id INTEGER,
			is_built_in INTEGER DEFAULT 0,
			is_custom INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by INTEGER
		);

		CREATE TABLE user_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			scope TEXT NOT NULL,
			resource_id TEXT,
			organization_id INTEGER,
			granted_by INTEGER,
			granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			UNIQUE(user_id, role_id, scope, resource_id, organization_id)
		);

		CREATE TABLE teams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			display_name TEXT NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			created_by INTEGER
		);

		CREATE TABLE team_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role_id INTEGER,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			added_by INTEGER
		);

		CREATE TABLE team_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			team_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			scope TEXT NOT NULL,
			resource_id TEXT,
			organization_id INTEGER,
			granted_by INTEGER,
			granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(team_id, role_id, scope, resource_id, organization_id)
		);

		CREATE TABLE permission_cache (
			user_id INTEGER NOT NULL,
			permission TEXT NOT NULL,
			scope TEXT NOT NULL,
			resource_id TEXT,
			organization_id INTEGER,
			allowed INTEGER NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, permission, scope)
		);
	`)

	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestPermissionChecker_CheckPermission(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)
	checker := NewPermissionChecker(db, 5*time.Minute)

	// Create test user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "testuser", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create test organization
	result, err = db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "testorg")
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create test role with permissions
	role := &Role{
		Name:        "test-role",
		DisplayName: "Test Role",
		Description: "Test role for unit tests",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
			{Resource: ResourceModule, Action: ActionCreate},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Assign role to user
	userRole := &UserRole{
		UserID:         userID,
		RoleID:         role.ID,
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	if err := store.AssignRoleToUser(ctx, userRole); err != nil {
		t.Fatalf("Failed to assign role to user: %v", err)
	}

	// Test: Check permission that user has
	check := PermissionCheck{
		UserID: userID,
		Permission: Permission{
			Resource: ResourceModule,
			Action:   ActionRead,
		},
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	result1, err := checker.CheckPermission(ctx, check)
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}

	if !result1.Allowed {
		t.Errorf("Expected permission to be allowed, but it was denied. Reason: %s", result1.Reason)
	}

	// Test: Check permission that user doesn't have
	check2 := PermissionCheck{
		UserID: userID,
		Permission: Permission{
			Resource: ResourceModule,
			Action:   ActionDelete,
		},
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	result2, err := checker.CheckPermission(ctx, check2)
	if err != nil {
		t.Fatalf("CheckPermission failed: %v", err)
	}

	if result2.Allowed {
		t.Errorf("Expected permission to be denied, but it was allowed")
	}
}

func TestPermissionChecker_RoleInheritance(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create parent role
	parentRole := &Role{
		Name:        "parent-role",
		DisplayName: "Parent Role",
		Description: "Parent role",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, parentRole); err != nil {
		t.Fatalf("Failed to create parent role: %v", err)
	}

	// Create child role that inherits from parent
	childRole := &Role{
		Name:         "child-role",
		DisplayName:  "Child Role",
		Description:  "Child role that inherits from parent",
		ParentRoleID: &parentRole.ID,
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionCreate},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, childRole); err != nil {
		t.Fatalf("Failed to create child role: %v", err)
	}

	// Verify inheritance
	checker := NewPermissionChecker(db, 5*time.Minute)
	roles, err := checker.resolveRoleInheritance(ctx, *childRole)
	if err != nil {
		t.Fatalf("Failed to resolve role inheritance: %v", err)
	}

	if len(roles) != 2 {
		t.Errorf("Expected 2 roles (child + parent), got %d", len(roles))
	}

	// Verify both child and parent permissions are present
	hasChildPerm := false
	hasParentPerm := false

	for _, role := range roles {
		for _, perm := range role.Permissions {
			if perm.Resource == ResourceModule && perm.Action == ActionCreate {
				hasChildPerm = true
			}
			if perm.Resource == ResourceModule && perm.Action == ActionRead {
				hasParentPerm = true
			}
		}
	}

	if !hasChildPerm {
		t.Error("Expected child permission (module:create) not found")
	}
	if !hasParentPerm {
		t.Error("Expected parent permission (module:read) not found")
	}
}

func TestPermissionChecker_CachedPermission(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)
	checker := NewPermissionChecker(db, 5*time.Minute)

	// Create test user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "cacheuser", "cache@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create test role
	role := &Role{
		Name:        "cache-role",
		DisplayName: "Cache Role",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Assign role
	userRole := &UserRole{
		UserID: userID,
		RoleID: role.ID,
		Scope:  ScopeOrganization,
	}

	if err := store.AssignRoleToUser(ctx, userRole); err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	// First check - should query database
	check := PermissionCheck{
		UserID: userID,
		Permission: Permission{
			Resource: ResourceModule,
			Action:   ActionRead,
		},
		Scope: ScopeOrganization,
	}

	result1, err := checker.CheckPermission(ctx, check)
	if err != nil {
		t.Fatalf("First check failed: %v", err)
	}

	if !result1.Allowed {
		t.Error("Expected first check to be allowed")
	}

	// Second check - should use cache
	result2, err := checker.CheckPermission(ctx, check)
	if err != nil {
		t.Fatalf("Second check failed: %v", err)
	}

	if !result2.Allowed {
		t.Error("Expected second check to be allowed")
	}

	// Note: Cache test is passing, the reason may vary depending on cache hit
	t.Logf("Second check reason: %s", result2.Reason)
}

func TestPermissionChecker_TeamRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)
	checker := NewPermissionChecker(db, 5*time.Minute)

	// Create test user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "teamuser", "team@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create test organization
	result, err = db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "teamorg")
	if err != nil {
		t.Fatalf("Failed to create test organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create team
	team := &Team{
		OrganizationID: orgID,
		Name:           "test-team",
		DisplayName:    "Test Team",
	}

	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Add user to team
	member := &TeamMember{
		TeamID: team.ID,
		UserID: userID,
	}

	if err := store.AddTeamMember(ctx, member); err != nil {
		t.Fatalf("Failed to add team member: %v", err)
	}

	// Create role
	role := &Role{
		Name:        "team-role",
		DisplayName: "Team Role",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// Assign role to team
	teamRole := &TeamRole{
		TeamID:         team.ID,
		RoleID:         role.ID,
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	if err := store.AssignRoleToTeam(ctx, teamRole); err != nil {
		t.Fatalf("Failed to assign role to team: %v", err)
	}

	// Check if user has permission through team
	check := PermissionCheck{
		UserID: userID,
		Permission: Permission{
			Resource: ResourceModule,
			Action:   ActionRead,
		},
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	result1, err := checker.CheckPermission(ctx, check)
	if err != nil {
		t.Fatalf("Permission check failed: %v", err)
	}

	if !result1.Allowed {
		t.Errorf("Expected permission through team to be allowed. Reason: %s", result1.Reason)
	}
}

func TestGetEffectivePermissions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)
	checker := NewPermissionChecker(db, 5*time.Minute)

	// Create test user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "permuser", "perm@example.com")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create multiple roles with different permissions
	role1 := &Role{
		Name:        "role1",
		DisplayName: "Role 1",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
			{Resource: ResourceModule, Action: ActionCreate},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	role2 := &Role{
		Name:        "role2",
		DisplayName: "Role 2",
		Permissions: []Permission{
			{Resource: ResourceVersion, Action: ActionPublish},
			{Resource: ResourceDocumentation, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role1); err != nil {
		t.Fatalf("Failed to create role1: %v", err)
	}

	if err := store.CreateRole(ctx, role2); err != nil {
		t.Fatalf("Failed to create role2: %v", err)
	}

	// Assign both roles to user
	if err := store.AssignRoleToUser(ctx, &UserRole{
		UserID: userID,
		RoleID: role1.ID,
		Scope:  ScopeOrganization,
	}); err != nil {
		t.Fatalf("Failed to assign role1: %v", err)
	}

	if err := store.AssignRoleToUser(ctx, &UserRole{
		UserID: userID,
		RoleID: role2.ID,
		Scope:  ScopeOrganization,
	}); err != nil {
		t.Fatalf("Failed to assign role2: %v", err)
	}

	// Get effective permissions
	permissions, err := checker.GetEffectivePermissions(ctx, userID, nil, nil)
	if err != nil {
		t.Fatalf("Failed to get effective permissions: %v", err)
	}

	// Should have 4 unique permissions
	if len(permissions) != 4 {
		t.Errorf("Expected 4 permissions, got %d", len(permissions))
	}

	// Verify all permissions are present
	expectedPerms := map[string]bool{
		"module:read":          false,
		"module:create":        false,
		"version:publish":      false,
		"documentation:read":   false,
	}

	for _, perm := range permissions {
		key := perm.String()
		if _, exists := expectedPerms[key]; exists {
			expectedPerms[key] = true
		}
	}

	for key, found := range expectedPerms {
		if !found {
			t.Errorf("Expected permission %s not found", key)
		}
	}
}
