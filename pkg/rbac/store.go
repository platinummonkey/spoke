package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Store handles RBAC data persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new RBAC store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateRole creates a new role
func (s *Store) CreateRole(ctx context.Context, role *Role) error {
	permissionsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `
		INSERT INTO roles (name, display_name, description, organization_id, permissions, parent_role_id, is_built_in, is_custom, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`

	now := time.Now()
	err = s.db.QueryRowContext(ctx, query,
		role.Name,
		role.DisplayName,
		role.Description,
		role.OrganizationID,
		string(permissionsJSON),
		role.ParentRoleID,
		role.IsBuiltIn,
		role.IsCustom,
		now,
		now,
		role.CreatedBy,
	).Scan(&role.ID)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	role.CreatedAt = now
	role.UpdatedAt = now
	return nil
}

// GetRole retrieves a role by ID
func (s *Store) GetRole(ctx context.Context, roleID int64) (*Role, error) {
	query := `
		SELECT id, name, display_name, description, organization_id, permissions, parent_role_id, is_built_in, is_custom, created_at, updated_at, created_by
		FROM roles
		WHERE id = $1
	`

	var role Role
	var permissionsJSON string
	var parentRoleID, orgID, createdBy sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, roleID).Scan(
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

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found: %d", roleID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Parse permissions
	if err := json.Unmarshal([]byte(permissionsJSON), &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	if parentRoleID.Valid {
		id := parentRoleID.Int64
		role.ParentRoleID = &id
	}
	if orgID.Valid {
		id := orgID.Int64
		role.OrganizationID = &id
	}
	if createdBy.Valid {
		id := createdBy.Int64
		role.CreatedBy = &id
	}

	return &role, nil
}

// GetRoleByName retrieves a role by name
func (s *Store) GetRoleByName(ctx context.Context, name string, organizationID *int64) (*Role, error) {
	query := `
		SELECT id, name, display_name, description, organization_id, permissions, parent_role_id, is_built_in, is_custom, created_at, updated_at, created_by
		FROM roles
		WHERE name = $1 AND (organization_id = $2 OR organization_id IS NULL)
		ORDER BY organization_id DESC NULLS LAST
		LIMIT 1
	`

	var role Role
	var permissionsJSON string
	var parentRoleID, orgID, createdBy sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, name, organizationID).Scan(
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

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Parse permissions
	if err := json.Unmarshal([]byte(permissionsJSON), &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	if parentRoleID.Valid {
		id := parentRoleID.Int64
		role.ParentRoleID = &id
	}
	if orgID.Valid {
		id := orgID.Int64
		role.OrganizationID = &id
	}
	if createdBy.Valid {
		id := createdBy.Int64
		role.CreatedBy = &id
	}

	return &role, nil
}

// ListRoles lists all roles
func (s *Store) ListRoles(ctx context.Context, organizationID *int64) ([]Role, error) {
	query := `
		SELECT id, name, display_name, description, organization_id, permissions, parent_role_id, is_built_in, is_custom, created_at, updated_at, created_by
		FROM roles
		WHERE organization_id = $1 OR organization_id IS NULL OR is_built_in = true
		ORDER BY is_built_in DESC, name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var permissionsJSON string
		var parentRoleID, orgID, createdBy sql.NullInt64

		err := rows.Scan(
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
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}

		// Parse permissions
		if err := json.Unmarshal([]byte(permissionsJSON), &role.Permissions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
		}

		if parentRoleID.Valid {
			id := parentRoleID.Int64
			role.ParentRoleID = &id
		}
		if orgID.Valid {
			id := orgID.Int64
			role.OrganizationID = &id
		}
		if createdBy.Valid {
			id := createdBy.Int64
			role.CreatedBy = &id
		}

		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// UpdateRole updates an existing role
func (s *Store) UpdateRole(ctx context.Context, role *Role) error {
	permissionsJSON, err := json.Marshal(role.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	query := `
		UPDATE roles
		SET display_name = $1, description = $2, permissions = $3, parent_role_id = $4, updated_at = $5
		WHERE id = $6
	`

	role.UpdatedAt = time.Now()
	_, err = s.db.ExecContext(ctx, query,
		role.DisplayName,
		role.Description,
		string(permissionsJSON),
		role.ParentRoleID,
		role.UpdatedAt,
		role.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	return nil
}

// DeleteRole deletes a role
func (s *Store) DeleteRole(ctx context.Context, roleID int64) error {
	// Check if role is built-in
	role, err := s.GetRole(ctx, roleID)
	if err != nil {
		return err
	}
	if role.IsBuiltIn {
		return fmt.Errorf("cannot delete built-in role")
	}

	query := `DELETE FROM roles WHERE id = $1`
	_, err = s.db.ExecContext(ctx, query, roleID)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	return nil
}

// AssignRoleToUser assigns a role to a user
func (s *Store) AssignRoleToUser(ctx context.Context, userRole *UserRole) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, scope, resource_id, organization_id, granted_by, granted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		userRole.UserID,
		userRole.RoleID,
		userRole.Scope,
		userRole.ResourceID,
		userRole.OrganizationID,
		userRole.GrantedBy,
		now,
		userRole.ExpiresAt,
	).Scan(&userRole.ID)

	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	userRole.GrantedAt = now
	return nil
}

// RevokeRoleFromUser revokes a role from a user
func (s *Store) RevokeRoleFromUser(ctx context.Context, userRoleID int64) error {
	query := `DELETE FROM user_roles WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userRoleID)
	if err != nil {
		return fmt.Errorf("failed to revoke role from user: %w", err)
	}
	return nil
}

// GetUserRoles retrieves all roles assigned to a user
func (s *Store) GetUserRoles(ctx context.Context, userID int64, organizationID *int64) ([]UserRole, error) {
	query := `
		SELECT id, user_id, role_id, scope, resource_id, organization_id, granted_by, granted_at, expires_at
		FROM user_roles
		WHERE user_id = $1
		  AND (organization_id = $2 OR organization_id IS NULL)
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		ORDER BY granted_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var userRoles []UserRole
	for rows.Next() {
		var ur UserRole
		var resourceID sql.NullString
		var orgID, grantedBy sql.NullInt64
		var expiresAt sql.NullTime

		err := rows.Scan(
			&ur.ID,
			&ur.UserID,
			&ur.RoleID,
			&ur.Scope,
			&resourceID,
			&orgID,
			&grantedBy,
			&ur.GrantedAt,
			&expiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user role: %w", err)
		}

		if resourceID.Valid {
			rid := resourceID.String
			ur.ResourceID = &rid
		}
		if orgID.Valid {
			oid := orgID.Int64
			ur.OrganizationID = &oid
		}
		if grantedBy.Valid {
			gb := grantedBy.Int64
			ur.GrantedBy = &gb
		}
		if expiresAt.Valid {
			ea := expiresAt.Time
			ur.ExpiresAt = &ea
		}

		userRoles = append(userRoles, ur)
	}

	return userRoles, rows.Err()
}

// CreateTeam creates a new team
func (s *Store) CreateTeam(ctx context.Context, team *Team) error {
	query := `
		INSERT INTO teams (organization_id, name, display_name, description, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		team.OrganizationID,
		team.Name,
		team.DisplayName,
		team.Description,
		now,
		now,
		team.CreatedBy,
	).Scan(&team.ID)

	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	team.CreatedAt = now
	team.UpdatedAt = now
	return nil
}

// GetTeam retrieves a team by ID
func (s *Store) GetTeam(ctx context.Context, teamID int64) (*Team, error) {
	query := `
		SELECT id, organization_id, name, display_name, description, created_at, updated_at, created_by
		FROM teams
		WHERE id = $1
	`

	var team Team
	var createdBy sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, teamID).Scan(
		&team.ID,
		&team.OrganizationID,
		&team.Name,
		&team.DisplayName,
		&team.Description,
		&team.CreatedAt,
		&team.UpdatedAt,
		&createdBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found: %d", teamID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	if createdBy.Valid {
		id := createdBy.Int64
		team.CreatedBy = &id
	}

	return &team, nil
}

// ListTeams lists all teams for an organization
func (s *Store) ListTeams(ctx context.Context, organizationID int64) ([]Team, error) {
	query := `
		SELECT id, organization_id, name, display_name, description, created_at, updated_at, created_by
		FROM teams
		WHERE organization_id = $1
		ORDER BY name ASC
	`

	rows, err := s.db.QueryContext(ctx, query, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer rows.Close()

	var teams []Team
	for rows.Next() {
		var team Team
		var createdBy sql.NullInt64

		err := rows.Scan(
			&team.ID,
			&team.OrganizationID,
			&team.Name,
			&team.DisplayName,
			&team.Description,
			&team.CreatedAt,
			&team.UpdatedAt,
			&createdBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}

		if createdBy.Valid {
			id := createdBy.Int64
			team.CreatedBy = &id
		}

		teams = append(teams, team)
	}

	return teams, rows.Err()
}

// UpdateTeam updates a team
func (s *Store) UpdateTeam(ctx context.Context, team *Team) error {
	query := `
		UPDATE teams
		SET display_name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`

	team.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, query,
		team.DisplayName,
		team.Description,
		team.UpdatedAt,
		team.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	return nil
}

// DeleteTeam deletes a team
func (s *Store) DeleteTeam(ctx context.Context, teamID int64) error {
	query := `DELETE FROM teams WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, teamID)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}
	return nil
}

// AddTeamMember adds a user to a team
func (s *Store) AddTeamMember(ctx context.Context, member *TeamMember) error {
	query := `
		INSERT INTO team_members (team_id, user_id, role_id, added_at, added_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		member.TeamID,
		member.UserID,
		member.RoleID,
		now,
		member.AddedBy,
	).Scan(&member.ID)

	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}

	member.AddedAt = now
	return nil
}

// RemoveTeamMember removes a user from a team
func (s *Store) RemoveTeamMember(ctx context.Context, teamID, userID int64) error {
	query := `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`
	_, err := s.db.ExecContext(ctx, query, teamID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove team member: %w", err)
	}
	return nil
}

// GetTeamMembers retrieves all members of a team
func (s *Store) GetTeamMembers(ctx context.Context, teamID int64) ([]TeamMember, error) {
	query := `
		SELECT id, team_id, user_id, role_id, added_at, added_by
		FROM team_members
		WHERE team_id = $1
		ORDER BY added_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var member TeamMember
		var roleID, addedBy sql.NullInt64

		err := rows.Scan(
			&member.ID,
			&member.TeamID,
			&member.UserID,
			&roleID,
			&member.AddedAt,
			&addedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team member: %w", err)
		}

		if roleID.Valid {
			rid := roleID.Int64
			member.RoleID = &rid
		}
		if addedBy.Valid {
			ab := addedBy.Int64
			member.AddedBy = &ab
		}

		members = append(members, member)
	}

	return members, rows.Err()
}

// AssignRoleToTeam assigns a role to a team
func (s *Store) AssignRoleToTeam(ctx context.Context, teamRole *TeamRole) error {
	query := `
		INSERT INTO team_roles (team_id, role_id, scope, resource_id, organization_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	err := s.db.QueryRowContext(ctx, query,
		teamRole.TeamID,
		teamRole.RoleID,
		teamRole.Scope,
		teamRole.ResourceID,
		teamRole.OrganizationID,
		teamRole.GrantedBy,
		now,
	).Scan(&teamRole.ID)

	if err != nil {
		return fmt.Errorf("failed to assign role to team: %w", err)
	}

	teamRole.GrantedAt = now
	return nil
}

// RevokeRoleFromTeam revokes a role from a team
func (s *Store) RevokeRoleFromTeam(ctx context.Context, teamRoleID int64) error {
	query := `DELETE FROM team_roles WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, teamRoleID)
	if err != nil {
		return fmt.Errorf("failed to revoke role from team: %w", err)
	}
	return nil
}
