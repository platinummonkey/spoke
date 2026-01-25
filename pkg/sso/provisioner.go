package sso

import (
	"database/sql"
	"fmt"

	"github.com/platinummonkey/spoke/pkg/auth"
)

// UserProvisioner handles JIT (Just-In-Time) user provisioning
type UserProvisioner struct {
	db *sql.DB
}

// NewUserProvisioner creates a new user provisioner
func NewUserProvisioner(db *sql.DB) *UserProvisioner {
	return &UserProvisioner{db: db}
}

// ProvisionUser provisions or updates a user from SSO
func (p *UserProvisioner) ProvisionUser(ssoUser *SSOUser, config *ProviderConfig) (*auth.User, error) {
	if !config.AutoProvision {
		return nil, fmt.Errorf("auto-provisioning is disabled for this provider")
	}

	// Check if user mapping exists
	var internalUserID int64
	err := p.db.QueryRow(`
		SELECT internal_user_id
		FROM sso_user_mappings
		WHERE provider_id = $1 AND external_user_id = $2
	`, config.ID, ssoUser.ExternalID).Scan(&internalUserID)

	if err == sql.ErrNoRows {
		// User doesn't exist, create new user
		return p.createUser(ssoUser, config)
	} else if err != nil {
		return nil, fmt.Errorf("failed to check user mapping: %w", err)
	}

	// User exists, update and return
	return p.updateUser(internalUserID, ssoUser, config)
}

// createUser creates a new user from SSO data
func (p *UserProvisioner) createUser(ssoUser *SSOUser, config *ProviderConfig) (*auth.User, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Determine full name
	fullName := ssoUser.FullName
	if fullName == "" && ssoUser.FirstName != "" && ssoUser.LastName != "" {
		fullName = ssoUser.FirstName + " " + ssoUser.LastName
	}

	// Insert user
	var userID int64
	err = tx.QueryRow(`
		INSERT INTO users (username, email, full_name, is_bot, is_active, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, false, true, NOW(), NOW(), NOW())
		RETURNING id
	`, ssoUser.Username, ssoUser.Email, fullName).Scan(&userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create SSO user mapping
	_, err = tx.Exec(`
		INSERT INTO sso_user_mappings (provider_id, external_user_id, internal_user_id, last_login_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW(), NOW())
	`, config.ID, ssoUser.ExternalID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to create user mapping: %w", err)
	}

	// Apply group mappings
	if len(ssoUser.Groups) > 0 && len(config.GroupMapping) > 0 {
		if err := p.applyGroupMappings(tx, userID, ssoUser.Groups, config.GroupMapping); err != nil {
			return nil, fmt.Errorf("failed to apply group mappings: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return created user
	user := &auth.User{}
	err = p.db.QueryRow(`
		SELECT id, username, email, full_name, is_bot, is_active, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.IsBot,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created user: %w", err)
	}

	return user, nil
}

// updateUser updates an existing user from SSO data
func (p *UserProvisioner) updateUser(userID int64, ssoUser *SSOUser, config *ProviderConfig) (*auth.User, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Determine full name
	fullName := ssoUser.FullName
	if fullName == "" && ssoUser.FirstName != "" && ssoUser.LastName != "" {
		fullName = ssoUser.FirstName + " " + ssoUser.LastName
	}

	// Update user information
	_, err = tx.Exec(`
		UPDATE users
		SET email = $1, full_name = $2, updated_at = NOW(), last_login_at = NOW()
		WHERE id = $3
	`, ssoUser.Email, fullName, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Update SSO mapping last login
	_, err = tx.Exec(`
		UPDATE sso_user_mappings
		SET last_login_at = NOW(), updated_at = NOW()
		WHERE provider_id = $1 AND external_user_id = $2
	`, config.ID, ssoUser.ExternalID)
	if err != nil {
		return nil, fmt.Errorf("failed to update user mapping: %w", err)
	}

	// Update group mappings if configured
	if len(ssoUser.Groups) > 0 && len(config.GroupMapping) > 0 {
		if err := p.applyGroupMappings(tx, userID, ssoUser.Groups, config.GroupMapping); err != nil {
			return nil, fmt.Errorf("failed to apply group mappings: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch and return updated user
	user := &auth.User{}
	err = p.db.QueryRow(`
		SELECT id, username, email, full_name, is_bot, is_active, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.FullName, &user.IsBot,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updated user: %w", err)
	}

	return user, nil
}

// applyGroupMappings applies group-to-role mappings
func (p *UserProvisioner) applyGroupMappings(tx *sql.Tx, userID int64, userGroups []string, groupMappings []GroupMap) error {
	// Build map of SSO groups to Spoke roles
	groupToRole := make(map[string]string)
	for _, mapping := range groupMappings {
		groupToRole[mapping.SSOGroup] = mapping.SpokeRole
	}

	// Determine which roles the user should have based on their groups
	rolesToAssign := make(map[string]bool)
	for _, group := range userGroups {
		if role, ok := groupToRole[group]; ok {
			rolesToAssign[role] = true
		}
	}

	// If no roles match, assign default role if configured
	if len(rolesToAssign) == 0 {
		// Default role handling is done at a higher level
		return nil
	}

	// For simplicity, assign the highest privilege role
	// Priority: admin > developer > viewer
	var finalRole string
	if rolesToAssign[string(auth.RoleAdmin)] {
		finalRole = string(auth.RoleAdmin)
	} else if rolesToAssign[string(auth.RoleDeveloper)] {
		finalRole = string(auth.RoleDeveloper)
	} else if rolesToAssign[string(auth.RoleViewer)] {
		finalRole = string(auth.RoleViewer)
	}

	if finalRole == "" {
		return nil
	}

	// Get or create default organization for SSO users
	var orgID int64
	err := tx.QueryRow(`
		SELECT id FROM organizations WHERE name = 'sso-users' LIMIT 1
	`).Scan(&orgID)

	if err == sql.ErrNoRows {
		// Create default organization
		err = tx.QueryRow(`
			INSERT INTO organizations (name, display_name, description, is_active, created_at, updated_at)
			VALUES ('sso-users', 'SSO Users', 'Default organization for SSO users', true, NOW(), NOW())
			RETURNING id
		`).Scan(&orgID)
		if err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to query organization: %w", err)
	}

	// Assign user to organization with role
	_, err = tx.Exec(`
		INSERT INTO organization_members (organization_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = $3
	`, orgID, userID, finalRole)
	if err != nil {
		return fmt.Errorf("failed to assign organization membership: %w", err)
	}

	return nil
}

// GetUserMapping retrieves the SSO user mapping
func (p *UserProvisioner) GetUserMapping(providerID int64, externalUserID string) (*SSOUserMapping, error) {
	mapping := &SSOUserMapping{}
	err := p.db.QueryRow(`
		SELECT id, provider_id, external_user_id, internal_user_id, last_login_at, created_at, updated_at
		FROM sso_user_mappings
		WHERE provider_id = $1 AND external_user_id = $2
	`, providerID, externalUserID).Scan(
		&mapping.ID, &mapping.ProviderID, &mapping.ExternalUserID,
		&mapping.InternalUserID, &mapping.LastLoginAt, &mapping.CreatedAt, &mapping.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return mapping, nil
}

// DeleteUserMapping removes an SSO user mapping
func (p *UserProvisioner) DeleteUserMapping(providerID int64, externalUserID string) error {
	_, err := p.db.Exec(`
		DELETE FROM sso_user_mappings
		WHERE provider_id = $1 AND external_user_id = $2
	`, providerID, externalUserID)
	return err
}

// ListUserMappings lists all user mappings for a provider
func (p *UserProvisioner) ListUserMappings(providerID int64) ([]*SSOUserMapping, error) {
	rows, err := p.db.Query(`
		SELECT id, provider_id, external_user_id, internal_user_id, last_login_at, created_at, updated_at
		FROM sso_user_mappings
		WHERE provider_id = $1
		ORDER BY created_at DESC
	`, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var mappings []*SSOUserMapping
	for rows.Next() {
		mapping := &SSOUserMapping{}
		err := rows.Scan(
			&mapping.ID, &mapping.ProviderID, &mapping.ExternalUserID,
			&mapping.InternalUserID, &mapping.LastLoginAt, &mapping.CreatedAt, &mapping.UpdatedAt)
		if err != nil {
			return nil, err
		}
		mappings = append(mappings, mapping)
	}

	return mappings, rows.Err()
}

// SessionManager manages SSO sessions
type SessionManager struct {
	db *sql.DB
}

// NewSessionManager creates a new session manager
func NewSessionManager(db *sql.DB) *SessionManager {
	return &SessionManager{db: db}
}

// CreateSession creates a new SSO session
func (sm *SessionManager) CreateSession(session *SSOSession) error {
	_, err := sm.db.Exec(`
		INSERT INTO sso_sessions (id, provider_id, user_id, external_user_id, saml_session_index, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, session.ID, session.ProviderID, session.UserID, session.ExternalUserID,
		session.SAMLSessionIndex, session.CreatedAt, session.ExpiresAt)
	return err
}

// GetSession retrieves an SSO session
func (sm *SessionManager) GetSession(sessionID string) (*SSOSession, error) {
	session := &SSOSession{}
	err := sm.db.QueryRow(`
		SELECT id, provider_id, user_id, external_user_id, saml_session_index, created_at, expires_at
		FROM sso_sessions
		WHERE id = $1 AND expires_at > NOW()
	`, sessionID).Scan(
		&session.ID, &session.ProviderID, &session.UserID, &session.ExternalUserID,
		&session.SAMLSessionIndex, &session.CreatedAt, &session.ExpiresAt)

	if err != nil {
		return nil, err
	}

	return session, nil
}

// DeleteSession deletes an SSO session
func (sm *SessionManager) DeleteSession(sessionID string) error {
	_, err := sm.db.Exec(`DELETE FROM sso_sessions WHERE id = $1`, sessionID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (sm *SessionManager) CleanupExpiredSessions() (int64, error) {
	result, err := sm.db.Exec(`DELETE FROM sso_sessions WHERE expires_at < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
