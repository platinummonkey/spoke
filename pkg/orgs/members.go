package orgs

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/platinummonkey/spoke/pkg/auth"
)

// ListMembers retrieves all members of an organization
func (s *PostgresService) ListMembers(orgID int64) ([]*OrgMember, error) {
	query := `
		SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = $1
		ORDER BY created_at ASC
	`
	rows, err := s.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list members: %w", err)
	}
	defer rows.Close()

	var members []*OrgMember
	for rows.Next() {
		member := &OrgMember{}
		var email sql.NullString
		var fullName sql.NullString
		if err := rows.Scan(
			&member.ID, &member.OrganizationID, &member.UserID, &member.Role,
			&member.InvitedBy, &member.JoinedAt, &member.CreatedAt,
			&member.Username, &email, &fullName, &member.IsBot,
		); err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}
		if email.Valid {
			member.Email = email.String
		}
		if fullName.Valid {
			member.FullName = fullName.String
		}
		members = append(members, member)
	}

	return members, nil
}

// GetMember retrieves a specific member
func (s *PostgresService) GetMember(orgID, userID int64) (*OrgMember, error) {
	query := `
		SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = $1 AND user_id = $2
	`
	member := &OrgMember{}
	var email sql.NullString
	var fullName sql.NullString
	err := s.db.QueryRow(query, orgID, userID).Scan(
		&member.ID, &member.OrganizationID, &member.UserID, &member.Role,
		&member.InvitedBy, &member.JoinedAt, &member.CreatedAt,
		&member.Username, &email, &fullName, &member.IsBot,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get member: %w", err)
	}

	if email.Valid {
		member.Email = email.String
	}
	if fullName.Valid {
		member.FullName = fullName.String
	}

	return member, nil
}

// AddMember adds a user to an organization
func (s *PostgresService) AddMember(orgID, userID int64, role auth.Role, invitedBy *int64) error {
	query := `
		INSERT INTO organization_members (organization_id, user_id, role, invited_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id, user_id) DO NOTHING
	`
	result, err := s.db.Exec(query, orgID, userID, role, invitedBy)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("member already exists")
	}

	return nil
}

// UpdateMemberRole updates a member's role
func (s *PostgresService) UpdateMemberRole(orgID, userID int64, role auth.Role) error {
	query := `UPDATE organization_members SET role = $1 WHERE organization_id = $2 AND user_id = $3`
	result, err := s.db.Exec(query, role, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// RemoveMember removes a user from an organization
func (s *PostgresService) RemoveMember(orgID, userID int64) error {
	query := `DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2`
	result, err := s.db.Exec(query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	return nil
}

// CreateInvitation creates a new invitation
func (s *PostgresService) CreateInvitation(invitation *OrgInvitation) error {
	// Generate token
	token, err := generateToken()
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}
	invitation.Token = token

	// Set defaults
	if invitation.InvitedAt.IsZero() {
		invitation.InvitedAt = time.Now()
	}
	if invitation.ExpiresAt.IsZero() {
		invitation.ExpiresAt = time.Now().Add(7 * 24 * time.Hour) // 7 days
	}

	query := `
		INSERT INTO org_invitations (org_id, email, role, token, invited_by, invited_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (org_id, email) DO UPDATE
		SET token = EXCLUDED.token, invited_at = EXCLUDED.invited_at, expires_at = EXCLUDED.expires_at
		RETURNING id
	`
	err = s.db.QueryRow(query, invitation.OrgID, invitation.Email, invitation.Role,
		invitation.Token, invitation.InvitedBy, invitation.InvitedAt, invitation.ExpiresAt).
		Scan(&invitation.ID)
	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	return nil
}

// GetInvitation retrieves an invitation by token
func (s *PostgresService) GetInvitation(token string) (*OrgInvitation, error) {
	query := `
		SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE token = $1
	`
	invitation := &OrgInvitation{}
	err := s.db.QueryRow(query, token).Scan(
		&invitation.ID, &invitation.OrgID, &invitation.Email, &invitation.Role,
		&invitation.Token, &invitation.InvitedBy, &invitation.InvitedAt, &invitation.ExpiresAt,
		&invitation.AcceptedAt, &invitation.AcceptedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return invitation, nil
}

// ListInvitations lists all invitations for an organization
func (s *PostgresService) ListInvitations(orgID int64) ([]*OrgInvitation, error) {
	query := `
		SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE org_id = $1 AND accepted_at IS NULL
		ORDER BY invited_at DESC
	`
	rows, err := s.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}
	defer rows.Close()

	var invitations []*OrgInvitation
	for rows.Next() {
		invitation := &OrgInvitation{}
		if err := rows.Scan(
			&invitation.ID, &invitation.OrgID, &invitation.Email, &invitation.Role,
			&invitation.Token, &invitation.InvitedBy, &invitation.InvitedAt, &invitation.ExpiresAt,
			&invitation.AcceptedAt, &invitation.AcceptedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}
		invitations = append(invitations, invitation)
	}

	return invitations, nil
}

// AcceptInvitation accepts an invitation and adds the user to the organization
func (s *PostgresService) AcceptInvitation(token string, userID int64) error {
	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get invitation
	query := `
		SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = $1
		FOR UPDATE
	`
	var id, orgID int64
	var email string
	var role auth.Role
	var expiresAt time.Time
	var acceptedAt sql.NullTime

	err = tx.QueryRow(query, token).Scan(&id, &orgID, &email, &role, &expiresAt, &acceptedAt)
	if err == sql.ErrNoRows {
		return fmt.Errorf("invitation not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get invitation: %w", err)
	}

	// Check if already accepted
	if acceptedAt.Valid {
		return fmt.Errorf("invitation already accepted")
	}

	// Check if expired
	if time.Now().After(expiresAt) {
		return fmt.Errorf("invitation expired")
	}

	// Add member
	query = `
		INSERT INTO organization_members (organization_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, user_id) DO NOTHING
	`
	_, err = tx.Exec(query, orgID, userID, role)
	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	// Mark invitation as accepted
	query = `UPDATE org_invitations SET accepted_at = NOW(), accepted_by = $1 WHERE id = $2`
	_, err = tx.Exec(query, userID, id)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	return tx.Commit()
}

// RevokeInvitation revokes an invitation
func (s *PostgresService) RevokeInvitation(id int64) error {
	query := `DELETE FROM org_invitations WHERE id = $1 AND accepted_at IS NULL`
	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke invitation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found or already accepted")
	}

	return nil
}

// CleanupExpiredInvitations removes expired invitations
func (s *PostgresService) CleanupExpiredInvitations() error {
	query := `DELETE FROM org_invitations WHERE expires_at < NOW() AND accepted_at IS NULL`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired invitations: %w", err)
	}
	return nil
}
