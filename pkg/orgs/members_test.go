package orgs

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a new mock service
func newMockService(t *testing.T) (*PostgresService, sqlmock.Sqlmock, *sql.DB) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	service := NewPostgresService(db)
	return service, mock, db
}

func TestListMembers(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success with multiple members", func(t *testing.T) {
		orgID := int64(1)
		now := time.Now()
		invitedBy := int64(2)

		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "role", "invited_by", "joined_at", "created_at",
			"username", "email", "full_name", "is_bot",
		}).
			AddRow(1, orgID, 10, auth.RoleAdmin, invitedBy, now, now, "admin_user", "admin@example.com", "Admin User", false).
			AddRow(2, orgID, 11, auth.RoleDeveloper, invitedBy, now, now, "dev_user", "dev@example.com", "Dev User", false).
			AddRow(3, orgID, 12, auth.RoleViewer, nil, now, now, "viewer_user", sql.NullString{}, sql.NullString{}, false)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1
		ORDER BY created_at ASC`).
			WithArgs(orgID).
			WillReturnRows(rows)

		members, err := service.ListMembers(orgID)
		require.NoError(t, err)
		assert.Len(t, members, 3)

		// Check first member
		assert.Equal(t, int64(1), members[0].ID)
		assert.Equal(t, orgID, members[0].OrganizationID)
		assert.Equal(t, int64(10), members[0].UserID)
		assert.Equal(t, auth.RoleAdmin, members[0].Role)
		assert.Equal(t, "admin_user", members[0].Username)
		assert.Equal(t, "admin@example.com", members[0].Email)
		assert.Equal(t, "Admin User", members[0].FullName)
		assert.False(t, members[0].IsBot)

		// Check third member (null email/full name)
		assert.Equal(t, "", members[2].Email)
		assert.Equal(t, "", members[2].FullName)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		orgID := int64(2)

		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "role", "invited_by", "joined_at", "created_at",
			"username", "email", "full_name", "is_bot",
		})

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1`).
			WithArgs(orgID).
			WillReturnRows(rows)

		members, err := service.ListMembers(orgID)
		require.NoError(t, err)
		assert.Empty(t, members)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("query error", func(t *testing.T) {
		orgID := int64(3)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1`).
			WithArgs(orgID).
			WillReturnError(fmt.Errorf("database connection error"))

		members, err := service.ListMembers(orgID)
		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "failed to list members")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		orgID := int64(4)

		// Using wrong number of columns to trigger scan error
		rows := sqlmock.NewRows([]string{
			"id", "organization_id",
		}).AddRow(1, orgID)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1`).
			WithArgs(orgID).
			WillReturnRows(rows)

		members, err := service.ListMembers(orgID)
		require.Error(t, err)
		assert.Nil(t, members)
		assert.Contains(t, err.Error(), "failed to scan member")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetMember(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		now := time.Now()
		invitedBy := int64(2)

		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "role", "invited_by", "joined_at", "created_at",
			"username", "email", "full_name", "is_bot",
		}).AddRow(1, orgID, userID, auth.RoleAdmin, invitedBy, now, now, "admin_user", "admin@example.com", "Admin User", false)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnRows(rows)

		member, err := service.GetMember(orgID, userID)
		require.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, int64(1), member.ID)
		assert.Equal(t, orgID, member.OrganizationID)
		assert.Equal(t, userID, member.UserID)
		assert.Equal(t, auth.RoleAdmin, member.Role)
		assert.Equal(t, "admin_user", member.Username)
		assert.Equal(t, "admin@example.com", member.Email)
		assert.Equal(t, "Admin User", member.FullName)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("member not found", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(999)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnError(sql.ErrNoRows)

		member, err := service.GetMember(orgID, userID)
		require.Error(t, err)
		assert.Nil(t, member)
		assert.Contains(t, err.Error(), "member not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("null email and full name", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(11)
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "organization_id", "user_id", "role", "invited_by", "joined_at", "created_at",
			"username", "email", "full_name", "is_bot",
		}).AddRow(2, orgID, userID, auth.RoleDeveloper, nil, now, now, "bot_user", sql.NullString{}, sql.NullString{}, true)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnRows(rows)

		member, err := service.GetMember(orgID, userID)
		require.NoError(t, err)
		assert.NotNil(t, member)
		assert.Equal(t, "", member.Email)
		assert.Equal(t, "", member.FullName)
		assert.True(t, member.IsBot)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)

		mock.ExpectQuery(`SELECT id, organization_id, user_id, role, invited_by, joined_at, created_at,
		       username, email, full_name, is_bot
		FROM org_members_view
		WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnError(fmt.Errorf("connection lost"))

		member, err := service.GetMember(orgID, userID)
		require.Error(t, err)
		assert.Nil(t, member)
		assert.Contains(t, err.Error(), "failed to get member")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAddMember(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleDeveloper
		invitedBy := int64(2)

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role, invited_by\)
		VALUES \(\$1, \$2, \$3, \$4\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role, &invitedBy).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.AddMember(orgID, userID, role, &invitedBy)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success without inviter", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(11)
		role := auth.RoleViewer

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role, invited_by\)
		VALUES \(\$1, \$2, \$3, \$4\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role, nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.AddMember(orgID, userID, role, nil)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("member already exists", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleDeveloper

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role, invited_by\)
		VALUES \(\$1, \$2, \$3, \$4\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role, nil).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.AddMember(orgID, userID, role, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member already exists")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleDeveloper

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role, invited_by\)
		VALUES \(\$1, \$2, \$3, \$4\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role, nil).
			WillReturnError(fmt.Errorf("constraint violation"))

		err := service.AddMember(orgID, userID, role, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add member")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleDeveloper

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role, invited_by\)
		VALUES \(\$1, \$2, \$3, \$4\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role, nil).
			WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

		err := service.AddMember(orgID, userID, role, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUpdateMemberRole(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleAdmin

		mock.ExpectExec(`UPDATE organization_members SET role = \$1 WHERE organization_id = \$2 AND user_id = \$3`).
			WithArgs(role, orgID, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.UpdateMemberRole(orgID, userID, role)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("member not found", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(999)
		role := auth.RoleAdmin

		mock.ExpectExec(`UPDATE organization_members SET role = \$1 WHERE organization_id = \$2 AND user_id = \$3`).
			WithArgs(role, orgID, userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.UpdateMemberRole(orgID, userID, role)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleAdmin

		mock.ExpectExec(`UPDATE organization_members SET role = \$1 WHERE organization_id = \$2 AND user_id = \$3`).
			WithArgs(role, orgID, userID).
			WillReturnError(fmt.Errorf("database error"))

		err := service.UpdateMemberRole(orgID, userID, role)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update member role")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)
		role := auth.RoleAdmin

		mock.ExpectExec(`UPDATE organization_members SET role = \$1 WHERE organization_id = \$2 AND user_id = \$3`).
			WithArgs(role, orgID, userID).
			WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

		err := service.UpdateMemberRole(orgID, userID, role)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRemoveMember(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)

		mock.ExpectExec(`DELETE FROM organization_members WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.RemoveMember(orgID, userID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("member not found", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(999)

		mock.ExpectExec(`DELETE FROM organization_members WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.RemoveMember(orgID, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "member not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)

		mock.ExpectExec(`DELETE FROM organization_members WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnError(fmt.Errorf("foreign key constraint"))

		err := service.RemoveMember(orgID, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to remove member")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected error", func(t *testing.T) {
		orgID := int64(1)
		userID := int64(10)

		mock.ExpectExec(`DELETE FROM organization_members WHERE organization_id = \$1 AND user_id = \$2`).
			WithArgs(orgID, userID).
			WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

		err := service.RemoveMember(orgID, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCreateInvitation(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success with defaults", func(t *testing.T) {
		invitation := &OrgInvitation{
			OrgID:     1,
			Email:     "newuser@example.com",
			Role:      auth.RoleDeveloper,
			InvitedBy: 2,
		}

		mock.ExpectQuery(`INSERT INTO org_invitations \(org_id, email, role, token, invited_by, invited_at, expires_at\)
		VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)
		ON CONFLICT \(org_id, email\) DO UPDATE
		SET token = EXCLUDED.token, invited_at = EXCLUDED.invited_at, expires_at = EXCLUDED.expires_at
		RETURNING id`).
			WithArgs(
				invitation.OrgID,
				invitation.Email,
				invitation.Role,
				sqlmock.AnyArg(), // token is generated
				invitation.InvitedBy,
				sqlmock.AnyArg(), // invited_at
				sqlmock.AnyArg(), // expires_at
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := service.CreateInvitation(invitation)
		require.NoError(t, err)
		assert.Equal(t, int64(1), invitation.ID)
		assert.NotEmpty(t, invitation.Token)
		assert.False(t, invitation.InvitedAt.IsZero())
		assert.False(t, invitation.ExpiresAt.IsZero())

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success with custom times", func(t *testing.T) {
		now := time.Now()
		expiresAt := now.Add(24 * time.Hour)

		invitation := &OrgInvitation{
			OrgID:     1,
			Email:     "newuser@example.com",
			Role:      auth.RoleViewer,
			InvitedBy: 2,
			InvitedAt: now,
			ExpiresAt: expiresAt,
		}

		mock.ExpectQuery(`INSERT INTO org_invitations \(org_id, email, role, token, invited_by, invited_at, expires_at\)
		VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)
		ON CONFLICT \(org_id, email\) DO UPDATE
		SET token = EXCLUDED.token, invited_at = EXCLUDED.invited_at, expires_at = EXCLUDED.expires_at
		RETURNING id`).
			WithArgs(
				invitation.OrgID,
				invitation.Email,
				invitation.Role,
				sqlmock.AnyArg(),
				invitation.InvitedBy,
				now,
				expiresAt,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))

		err := service.CreateInvitation(invitation)
		require.NoError(t, err)
		assert.Equal(t, int64(2), invitation.ID)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		invitation := &OrgInvitation{
			OrgID:     1,
			Email:     "newuser@example.com",
			Role:      auth.RoleDeveloper,
			InvitedBy: 2,
		}

		mock.ExpectQuery(`INSERT INTO org_invitations \(org_id, email, role, token, invited_by, invited_at, expires_at\)
		VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7\)
		ON CONFLICT \(org_id, email\) DO UPDATE
		SET token = EXCLUDED.token, invited_at = EXCLUDED.invited_at, expires_at = EXCLUDED.expires_at
		RETURNING id`).
			WillReturnError(fmt.Errorf("database error"))

		err := service.CreateInvitation(invitation)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetInvitation(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		token := "abc123"
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "token", "invited_by", "invited_at", "expires_at", "accepted_at", "accepted_by",
		}).AddRow(1, 1, "test@example.com", auth.RoleDeveloper, token, 2, now, now.Add(7*24*time.Hour), nil, nil)

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE token = \$1`).
			WithArgs(token).
			WillReturnRows(rows)

		invitation, err := service.GetInvitation(token)
		require.NoError(t, err)
		assert.NotNil(t, invitation)
		assert.Equal(t, int64(1), invitation.ID)
		assert.Equal(t, int64(1), invitation.OrgID)
		assert.Equal(t, "test@example.com", invitation.Email)
		assert.Equal(t, auth.RoleDeveloper, invitation.Role)
		assert.Equal(t, token, invitation.Token)
		assert.Nil(t, invitation.AcceptedAt)
		assert.Nil(t, invitation.AcceptedBy)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invitation not found", func(t *testing.T) {
		token := "invalid"

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE token = \$1`).
			WithArgs(token).
			WillReturnError(sql.ErrNoRows)

		invitation, err := service.GetInvitation(token)
		require.Error(t, err)
		assert.Nil(t, invitation)
		assert.Contains(t, err.Error(), "invitation not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		token := "abc123"

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE token = \$1`).
			WithArgs(token).
			WillReturnError(fmt.Errorf("database error"))

		invitation, err := service.GetInvitation(token)
		require.Error(t, err)
		assert.Nil(t, invitation)
		assert.Contains(t, err.Error(), "failed to get invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestListInvitations(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success with multiple invitations", func(t *testing.T) {
		orgID := int64(1)
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "token", "invited_by", "invited_at", "expires_at", "accepted_at", "accepted_by",
		}).
			AddRow(1, orgID, "user1@example.com", auth.RoleDeveloper, "token1", 2, now, now.Add(7*24*time.Hour), nil, nil).
			AddRow(2, orgID, "user2@example.com", auth.RoleViewer, "token2", 2, now.Add(-1*time.Hour), now.Add(6*24*time.Hour), nil, nil)

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE org_id = \$1 AND accepted_at IS NULL
		ORDER BY invited_at DESC`).
			WithArgs(orgID).
			WillReturnRows(rows)

		invitations, err := service.ListInvitations(orgID)
		require.NoError(t, err)
		assert.Len(t, invitations, 2)
		assert.Equal(t, "user1@example.com", invitations[0].Email)
		assert.Equal(t, "user2@example.com", invitations[1].Email)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		orgID := int64(2)

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "token", "invited_by", "invited_at", "expires_at", "accepted_at", "accepted_by",
		})

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE org_id = \$1 AND accepted_at IS NULL`).
			WithArgs(orgID).
			WillReturnRows(rows)

		invitations, err := service.ListInvitations(orgID)
		require.NoError(t, err)
		assert.Empty(t, invitations)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		orgID := int64(3)

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE org_id = \$1 AND accepted_at IS NULL`).
			WithArgs(orgID).
			WillReturnError(fmt.Errorf("connection error"))

		invitations, err := service.ListInvitations(orgID)
		require.Error(t, err)
		assert.Nil(t, invitations)
		assert.Contains(t, err.Error(), "failed to list invitations")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("scan error", func(t *testing.T) {
		orgID := int64(4)

		// Using wrong number of columns to trigger scan error
		rows := sqlmock.NewRows([]string{
			"id", "org_id",
		}).AddRow(1, orgID)

		mock.ExpectQuery(`SELECT id, org_id, email, role, token, invited_by, invited_at, expires_at, accepted_at, accepted_by
		FROM org_invitations
		WHERE org_id = \$1 AND accepted_at IS NULL`).
			WithArgs(orgID).
			WillReturnRows(rows)

		invitations, err := service.ListInvitations(orgID)
		require.Error(t, err)
		assert.Nil(t, invitations)
		assert.Contains(t, err.Error(), "failed to scan invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestAcceptInvitation(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		token := "valid_token"
		userID := int64(10)
		orgID := int64(1)
		invitationID := int64(1)
		email := "test@example.com"
		role := auth.RoleDeveloper
		expiresAt := time.Now().Add(24 * time.Hour)

		mock.ExpectBegin()

		// Get invitation
		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "expires_at", "accepted_at",
		}).AddRow(invitationID, orgID, email, role, expiresAt, sql.NullTime{})

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnRows(rows)

		// Add member
		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role\)
		VALUES \(\$1, \$2, \$3\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Mark invitation as accepted
		mock.ExpectExec(`UPDATE org_invitations SET accepted_at = NOW\(\), accepted_by = \$1 WHERE id = \$2`).
			WithArgs(userID, invitationID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		err := service.AcceptInvitation(token, userID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invitation not found", func(t *testing.T) {
		token := "invalid_token"
		userID := int64(10)

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invitation not found")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invitation already accepted", func(t *testing.T) {
		token := "accepted_token"
		userID := int64(10)
		acceptedAt := time.Now()

		mock.ExpectBegin()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "expires_at", "accepted_at",
		}).AddRow(1, 1, "test@example.com", auth.RoleDeveloper, time.Now().Add(24*time.Hour), sql.NullTime{Valid: true, Time: acceptedAt})

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnRows(rows)

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invitation already accepted")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invitation expired", func(t *testing.T) {
		token := "expired_token"
		userID := int64(10)
		expiresAt := time.Now().Add(-24 * time.Hour) // expired yesterday

		mock.ExpectBegin()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "expires_at", "accepted_at",
		}).AddRow(1, 1, "test@example.com", auth.RoleDeveloper, expiresAt, sql.NullTime{})

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnRows(rows)

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invitation expired")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("begin transaction error", func(t *testing.T) {
		token := "token"
		userID := int64(10)

		mock.ExpectBegin().WillReturnError(fmt.Errorf("transaction error"))

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("get invitation error", func(t *testing.T) {
		token := "token"
		userID := int64(10)

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnError(fmt.Errorf("database error"))

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("add member error", func(t *testing.T) {
		token := "token"
		userID := int64(10)
		orgID := int64(1)
		role := auth.RoleDeveloper

		mock.ExpectBegin()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "expires_at", "accepted_at",
		}).AddRow(1, orgID, "test@example.com", role, time.Now().Add(24*time.Hour), sql.NullTime{})

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnRows(rows)

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role\)
		VALUES \(\$1, \$2, \$3\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role).
			WillReturnError(fmt.Errorf("constraint error"))

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add member")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("update invitation error", func(t *testing.T) {
		token := "token"
		userID := int64(10)
		orgID := int64(1)
		invitationID := int64(1)
		role := auth.RoleDeveloper

		mock.ExpectBegin()

		rows := sqlmock.NewRows([]string{
			"id", "org_id", "email", "role", "expires_at", "accepted_at",
		}).AddRow(invitationID, orgID, "test@example.com", role, time.Now().Add(24*time.Hour), sql.NullTime{})

		mock.ExpectQuery(`SELECT id, org_id, email, role, expires_at, accepted_at
		FROM org_invitations
		WHERE token = \$1
		FOR UPDATE`).
			WithArgs(token).
			WillReturnRows(rows)

		mock.ExpectExec(`INSERT INTO organization_members \(organization_id, user_id, role\)
		VALUES \(\$1, \$2, \$3\)
		ON CONFLICT \(organization_id, user_id\) DO NOTHING`).
			WithArgs(orgID, userID, role).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`UPDATE org_invitations SET accepted_at = NOW\(\), accepted_by = \$1 WHERE id = \$2`).
			WithArgs(userID, invitationID).
			WillReturnError(fmt.Errorf("update error"))

		mock.ExpectRollback()

		err := service.AcceptInvitation(token, userID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRevokeInvitation(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		invitationID := int64(1)

		mock.ExpectExec(`DELETE FROM org_invitations WHERE id = \$1 AND accepted_at IS NULL`).
			WithArgs(invitationID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.RevokeInvitation(invitationID)
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invitation not found", func(t *testing.T) {
		invitationID := int64(999)

		mock.ExpectExec(`DELETE FROM org_invitations WHERE id = \$1 AND accepted_at IS NULL`).
			WithArgs(invitationID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.RevokeInvitation(invitationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invitation not found or already accepted")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		invitationID := int64(1)

		mock.ExpectExec(`DELETE FROM org_invitations WHERE id = \$1 AND accepted_at IS NULL`).
			WithArgs(invitationID).
			WillReturnError(fmt.Errorf("database error"))

		err := service.RevokeInvitation(invitationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to revoke invitation")

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rows affected error", func(t *testing.T) {
		invitationID := int64(1)

		mock.ExpectExec(`DELETE FROM org_invitations WHERE id = \$1 AND accepted_at IS NULL`).
			WithArgs(invitationID).
			WillReturnResult(sqlmock.NewErrorResult(fmt.Errorf("rows affected error")))

		err := service.RevokeInvitation(invitationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get rows affected")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCleanupExpiredInvitations(t *testing.T) {
	service, mock, db := newMockService(t)
	defer db.Close()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM org_invitations WHERE expires_at < NOW\(\) AND accepted_at IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 5))

		err := service.CleanupExpiredInvitations()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("no expired invitations", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM org_invitations WHERE expires_at < NOW\(\) AND accepted_at IS NULL`).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := service.CleanupExpiredInvitations()
		require.NoError(t, err)

		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM org_invitations WHERE expires_at < NOW\(\) AND accepted_at IS NULL`).
			WillReturnError(fmt.Errorf("database error"))

		err := service.CleanupExpiredInvitations()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to cleanup expired invitations")

		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// Implement the driver.Result interface for error results
type errorResult struct {
	err error
}

func (r errorResult) LastInsertId() (int64, error) {
	return 0, r.err
}

func (r errorResult) RowsAffected() (int64, error) {
	return 0, r.err
}

// Helper to create error results
func newErrorResult(err error) driver.Result {
	return errorResult{err: err}
}
