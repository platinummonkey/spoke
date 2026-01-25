package rbac

import (
	"context"
	"testing"
	"time"
)

func TestStore_RoleCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create
	role := &Role{
		Name:        "test-crud-role",
		DisplayName: "Test CRUD Role",
		Description: "Testing CRUD operations",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	if role.ID == 0 {
		t.Error("Expected role ID to be set after creation")
	}

	// Read
	retrieved, err := store.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole failed: %v", err)
	}

	if retrieved.Name != role.Name {
		t.Errorf("Expected name %s, got %s", role.Name, retrieved.Name)
	}

	// Update
	retrieved.DisplayName = "Updated Display Name"
	retrieved.Permissions = append(retrieved.Permissions, Permission{
		Resource: ResourceModule,
		Action:   ActionCreate,
	})

	if err := store.UpdateRole(ctx, retrieved); err != nil {
		t.Fatalf("UpdateRole failed: %v", err)
	}

	// Verify update
	updated, err := store.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatalf("GetRole after update failed: %v", err)
	}

	if updated.DisplayName != "Updated Display Name" {
		t.Errorf("Expected display name to be updated")
	}

	if len(updated.Permissions) != 2 {
		t.Errorf("Expected 2 permissions after update, got %d", len(updated.Permissions))
	}

	// Delete
	if err := store.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetRole(ctx, role.ID)
	if err == nil {
		t.Error("Expected error when getting deleted role")
	}
}

func TestStore_GetRoleByName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create organization
	result, err := db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "testorg")
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create organization-specific role
	role := &Role{
		Name:           "org-role",
		DisplayName:    "Org Role",
		OrganizationID: &orgID,
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Get by name
	retrieved, err := store.GetRoleByName(ctx, "org-role", &orgID)
	if err != nil {
		t.Fatalf("GetRoleByName failed: %v", err)
	}

	if retrieved.ID != role.ID {
		t.Errorf("Expected role ID %d, got %d", role.ID, retrieved.ID)
	}
}

func TestStore_UserRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "roleuser", "role@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create role
	role := &Role{
		Name:        "user-test-role",
		DisplayName: "User Test Role",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Assign role to user
	userRole := &UserRole{
		UserID: userID,
		RoleID: role.ID,
		Scope:  ScopeOrganization,
	}

	if err := store.AssignRoleToUser(ctx, userRole); err != nil {
		t.Fatalf("AssignRoleToUser failed: %v", err)
	}

	if userRole.ID == 0 {
		t.Error("Expected user role ID to be set")
	}

	// Get user roles
	userRoles, err := store.GetUserRoles(ctx, userID, nil)
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}

	if len(userRoles) != 1 {
		t.Errorf("Expected 1 user role, got %d", len(userRoles))
	}

	if userRoles[0].RoleID != role.ID {
		t.Errorf("Expected role ID %d, got %d", role.ID, userRoles[0].RoleID)
	}

	// Revoke role
	if err := store.RevokeRoleFromUser(ctx, userRole.ID); err != nil {
		t.Fatalf("RevokeRoleFromUser failed: %v", err)
	}

	// Verify revocation
	userRoles, err = store.GetUserRoles(ctx, userID, nil)
	if err != nil {
		t.Fatalf("GetUserRoles after revoke failed: %v", err)
	}

	if len(userRoles) != 0 {
		t.Errorf("Expected 0 user roles after revocation, got %d", len(userRoles))
	}
}

func TestStore_ExpiringRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create user
	result, err := db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "expireuser", "expire@example.com")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	userID, _ := result.LastInsertId()

	// Create role
	role := &Role{
		Name:        "expire-role",
		DisplayName: "Expiring Role",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Assign role with past expiration
	pastExpiry := time.Now().Add(-1 * time.Hour)
	userRole := &UserRole{
		UserID:    userID,
		RoleID:    role.ID,
		Scope:     ScopeOrganization,
		ExpiresAt: &pastExpiry,
	}

	if err := store.AssignRoleToUser(ctx, userRole); err != nil {
		t.Fatalf("AssignRoleToUser failed: %v", err)
	}

	// Get user roles - should not include expired role
	userRoles, err := store.GetUserRoles(ctx, userID, nil)
	if err != nil {
		t.Fatalf("GetUserRoles failed: %v", err)
	}

	if len(userRoles) != 0 {
		t.Errorf("Expected 0 active roles (expired), got %d", len(userRoles))
	}
}

func TestStore_TeamCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create organization
	result, err := db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "teamorg")
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create team
	team := &Team{
		OrganizationID: orgID,
		Name:           "test-team-crud",
		DisplayName:    "Test Team CRUD",
		Description:    "Testing team CRUD",
	}

	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	if team.ID == 0 {
		t.Error("Expected team ID to be set")
	}

	// Read
	retrieved, err := store.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam failed: %v", err)
	}

	if retrieved.Name != team.Name {
		t.Errorf("Expected name %s, got %s", team.Name, retrieved.Name)
	}

	// Update
	retrieved.DisplayName = "Updated Team Name"
	if err := store.UpdateTeam(ctx, retrieved); err != nil {
		t.Fatalf("UpdateTeam failed: %v", err)
	}

	// Verify update
	updated, err := store.GetTeam(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeam after update failed: %v", err)
	}

	if updated.DisplayName != "Updated Team Name" {
		t.Error("Expected display name to be updated")
	}

	// List teams
	teams, err := store.ListTeams(ctx, orgID)
	if err != nil {
		t.Fatalf("ListTeams failed: %v", err)
	}

	if len(teams) != 1 {
		t.Errorf("Expected 1 team, got %d", len(teams))
	}

	// Delete
	if err := store.DeleteTeam(ctx, team.ID); err != nil {
		t.Fatalf("DeleteTeam failed: %v", err)
	}

	// Verify deletion
	_, err = store.GetTeam(ctx, team.ID)
	if err == nil {
		t.Error("Expected error when getting deleted team")
	}
}

func TestStore_TeamMembers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create organization
	result, err := db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "memberorg")
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create team
	team := &Team{
		OrganizationID: orgID,
		Name:           "member-team",
		DisplayName:    "Member Team",
	}

	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	// Create users
	result, err = db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "member1", "member1@example.com")
	if err != nil {
		t.Fatalf("Failed to create user1: %v", err)
	}
	user1ID, _ := result.LastInsertId()

	result, err = db.ExecContext(ctx, "INSERT INTO users (username, email) VALUES (?, ?)", "member2", "member2@example.com")
	if err != nil {
		t.Fatalf("Failed to create user2: %v", err)
	}
	user2ID, _ := result.LastInsertId()

	// Add members
	member1 := &TeamMember{
		TeamID: team.ID,
		UserID: user1ID,
	}

	if err := store.AddTeamMember(ctx, member1); err != nil {
		t.Fatalf("AddTeamMember for user1 failed: %v", err)
	}

	member2 := &TeamMember{
		TeamID: team.ID,
		UserID: user2ID,
	}

	if err := store.AddTeamMember(ctx, member2); err != nil {
		t.Fatalf("AddTeamMember for user2 failed: %v", err)
	}

	// Get team members
	members, err := store.GetTeamMembers(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeamMembers failed: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 team members, got %d", len(members))
	}

	// Remove member
	if err := store.RemoveTeamMember(ctx, team.ID, user1ID); err != nil {
		t.Fatalf("RemoveTeamMember failed: %v", err)
	}

	// Verify removal
	members, err = store.GetTeamMembers(ctx, team.ID)
	if err != nil {
		t.Fatalf("GetTeamMembers after removal failed: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 team member after removal, got %d", len(members))
	}

	if members[0].UserID != user2ID {
		t.Errorf("Expected remaining member to be user2 (%d), got %d", user2ID, members[0].UserID)
	}
}

func TestStore_TeamRoles(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)

	// Create organization
	result, err := db.ExecContext(ctx, "INSERT INTO organizations (name) VALUES (?)", "teamroleorg")
	if err != nil {
		t.Fatalf("Failed to create organization: %v", err)
	}
	orgID, _ := result.LastInsertId()

	// Create team
	team := &Team{
		OrganizationID: orgID,
		Name:           "role-team",
		DisplayName:    "Role Team",
	}

	if err := store.CreateTeam(ctx, team); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	// Create role
	role := &Role{
		Name:        "team-role-test",
		DisplayName: "Team Role Test",
		Permissions: []Permission{
			{Resource: ResourceModule, Action: ActionRead},
		},
		IsBuiltIn: false,
		IsCustom:  true,
	}

	if err := store.CreateRole(ctx, role); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// Assign role to team
	teamRole := &TeamRole{
		TeamID:         team.ID,
		RoleID:         role.ID,
		Scope:          ScopeOrganization,
		OrganizationID: &orgID,
	}

	if err := store.AssignRoleToTeam(ctx, teamRole); err != nil {
		t.Fatalf("AssignRoleToTeam failed: %v", err)
	}

	if teamRole.ID == 0 {
		t.Error("Expected team role ID to be set")
	}

	// Revoke role from team
	if err := store.RevokeRoleFromTeam(ctx, teamRole.ID); err != nil {
		t.Fatalf("RevokeRoleFromTeam failed: %v", err)
	}
}

func TestBuiltInRoles(t *testing.T) {
	roles := BuiltInRoles()

	if len(roles) == 0 {
		t.Error("Expected built-in roles to be defined")
	}

	// Verify each built-in role has required fields
	expectedRoles := []string{
		RoleOrgAdmin,
		RoleOrgDeveloper,
		RoleOrgViewer,
		RoleModuleOwner,
		RoleModuleContributor,
		RoleModuleViewer,
	}

	foundRoles := make(map[string]bool)
	for _, role := range roles {
		if !role.IsBuiltIn {
			t.Errorf("Role %s should be marked as built-in", role.Name)
		}
		if len(role.Permissions) == 0 {
			t.Errorf("Role %s should have at least one permission", role.Name)
		}
		foundRoles[role.Name] = true
	}

	for _, expectedRole := range expectedRoles {
		if !foundRoles[expectedRole] {
			t.Errorf("Expected built-in role %s not found", expectedRole)
		}
	}
}
