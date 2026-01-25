package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// This file demonstrates how to integrate audit logging with authentication handlers

// AuditedAuthHandlers wraps authentication handlers with audit logging
type AuditedAuthHandlers struct {
	db     *sql.DB
	logger Logger
}

// NewAuditedAuthHandlers creates auth handlers with audit logging
func NewAuditedAuthHandlers(db *sql.DB, logger Logger) *AuditedAuthHandlers {
	return &AuditedAuthHandlers{
		db:     db,
		logger: logger,
	}
}

// Example: createUser with audit logging
func (h *AuditedAuthHandlers) createUserExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		IsBot    bool   `json:"is_bot"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Username == "" {
		LogFailure(ctx, EventTypeAdminUserCreate, "Username is required", nil)
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}

	// Insert user into database
	var userID int64
	err := h.db.QueryRowContext(ctx, `
		INSERT INTO users (username, email, is_bot, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW())
		RETURNING id
	`, req.Username, req.Email, req.IsBot).Scan(&userID)

	if err != nil {
		LogFailure(ctx, EventTypeAdminUserCreate, "Failed to create user in database", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log successful user creation
	h.logger.LogAdminAction(ctx,
		EventTypeAdminUserCreate,
		getAdminUserID(ctx),
		&userID,
		"User created successfully",
	)

	LogSuccess(ctx, EventTypeAdminUserCreate, "User created", map[string]interface{}{
		"user_id":  userID,
		"username": req.Username,
		"is_bot":   req.IsBot,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"username": req.Username,
		"email":    req.Email,
	})
}

// Example: createToken with audit logging
func (h *AuditedAuthHandlers) createTokenExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UserID int64    `json:"user_id"`
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate token (simplified for example)
	token := "generated-token-value"
	tokenHash := "hashed-token-value"
	tokenPrefix := "spk_"

	// Insert token into database
	var tokenID int64
	err := h.db.QueryRowContext(ctx, `
		INSERT INTO api_tokens (user_id, token_hash, token_prefix, name, scopes, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id
	`, req.UserID, tokenHash, tokenPrefix, req.Name, req.Scopes).Scan(&tokenID)

	if err != nil {
		LogFailure(ctx, EventTypeAuthTokenCreate, "Failed to create token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log token creation
	h.logger.LogAuthentication(ctx,
		EventTypeAuthTokenCreate,
		&req.UserID,
		"",
		EventStatusSuccess,
		"API token created",
	)

	LogSuccess(ctx, EventTypeAuthTokenCreate, "Token created", map[string]interface{}{
		"token_id":     tokenID,
		"user_id":      req.UserID,
		"token_name":   req.Name,
		"scopes_count": len(req.Scopes),
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           tokenID,
		"token":        token, // Only shown once
		"token_prefix": tokenPrefix,
		"name":         req.Name,
	})
}

// Example: revokeToken with audit logging
func (h *AuditedAuthHandlers) revokeTokenExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	tokenID := vars["id"]

	// Revoke token
	_, err := h.db.ExecContext(ctx, `
		UPDATE api_tokens SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, tokenID)

	if err != nil {
		LogFailure(ctx, EventTypeAuthTokenRevoke, "Failed to revoke token", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log token revocation
	h.logger.LogAuthentication(ctx,
		EventTypeAuthTokenRevoke,
		getAdminUserID(ctx),
		"",
		EventStatusSuccess,
		"API token revoked",
	)

	LogSuccess(ctx, EventTypeAuthTokenRevoke, "Token revoked", map[string]interface{}{
		"token_id": tokenID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Example: grantModulePermission with audit logging
func (h *AuditedAuthHandlers) grantModulePermissionExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	moduleName := vars["module_name"]

	var req struct {
		OrganizationID int64  `json:"organization_id"`
		Permission     string `json:"permission"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Grant permission
	var permID int64
	err := h.db.QueryRowContext(ctx, `
		INSERT INTO module_permissions (module_name, organization_id, permission, granted_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`, moduleName, req.OrganizationID, req.Permission).Scan(&permID)

	if err != nil {
		LogFailure(ctx, EventTypeAuthzPermissionGrant, "Failed to grant permission", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log permission grant
	h.logger.LogAuthorization(ctx,
		EventTypeAuthzPermissionGrant,
		getAdminUserID(ctx),
		ResourceTypeModule,
		moduleName,
		EventStatusSuccess,
		"Module permission granted",
	)

	LogSuccess(ctx, EventTypeAuthzPermissionGrant, "Permission granted", map[string]interface{}{
		"permission_id":   permID,
		"module_name":     moduleName,
		"organization_id": req.OrganizationID,
		"permission":      req.Permission,
	})

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":         permID,
		"status":     "granted",
		"permission": req.Permission,
	})
}

// Example: addOrganizationMember with audit logging and change tracking
func (h *AuditedAuthHandlers) addOrganizationMemberExample(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	orgID := vars["id"]

	var req struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if member already exists
	var existingRole string
	var isUpdate bool
	err := h.db.QueryRowContext(ctx, `
		SELECT role FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`, orgID, req.UserID).Scan(&existingRole)

	if err == nil {
		isUpdate = true
	}

	// Add or update member
	_, err = h.db.ExecContext(ctx, `
		INSERT INTO organization_members (organization_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = $3
	`, orgID, req.UserID, req.Role)

	if err != nil {
		LogFailure(ctx, EventTypeAdminOrgMemberAdd, "Failed to add organization member", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Log with appropriate event type and change tracking
	if isUpdate {
		changes := &ChangeDetails{
			Before: map[string]interface{}{"role": existingRole},
			After:  map[string]interface{}{"role": req.Role},
		}

		h.logger.LogAdminAction(ctx,
			EventTypeAdminOrgMemberRoleChange,
			getAdminUserID(ctx),
			&req.UserID,
			"Organization member role changed",
		)

		h.logger.LogDataMutation(ctx,
			EventTypeAdminOrgMemberRoleChange,
			getAdminUserID(ctx),
			ResourceTypeOrganization,
			orgID,
			changes,
			"Member role updated",
		)
	} else {
		h.logger.LogAdminAction(ctx,
			EventTypeAdminOrgMemberAdd,
			getAdminUserID(ctx),
			&req.UserID,
			"Organization member added",
		)

		LogSuccess(ctx, EventTypeAdminOrgMemberAdd, "Member added", map[string]interface{}{
			"organization_id": orgID,
			"user_id":         req.UserID,
			"role":            req.Role,
		})
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "added"})
}

// Example: validateToken with audit logging for failures
func (h *AuditedAuthHandlers) validateTokenExample(token string, r *http.Request) error {
	ctx := r.Context()

	// Validate token (simplified)
	var tokenID int64
	var userID int64
	err := h.db.QueryRowContext(ctx, `
		SELECT id, user_id FROM api_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
		AND (expires_at IS NULL OR expires_at > NOW())
	`, hashToken(token)).Scan(&tokenID, &userID)

	if err != nil {
		// Log failed validation
		h.logger.LogAuthentication(ctx,
			EventTypeAuthTokenValidateFail,
			nil,
			"",
			EventStatusFailure,
			"Invalid or expired token",
		)

		LogFailure(ctx, EventTypeAuthTokenValidateFail, "Token validation failed", err)
		return err
	}

	// Log successful validation (optional, may be noisy)
	// h.logger.LogAuthentication(ctx,
	// 	EventTypeAuthTokenValidate,
	// 	&userID,
	// 	"",
	// 	EventStatusSuccess,
	// 	"Token validated successfully",
	// )

	return nil
}

// Helper function to get admin user ID from context
func getAdminUserID(ctx context.Context) *int64 {
	userID, _, _, _ := GetAuditContext(ctx)
	return userID
}

// Helper function to hash token (simplified for example)
func hashToken(token string) string {
	// In real implementation, use crypto/sha256
	return "hashed-" + token
}
