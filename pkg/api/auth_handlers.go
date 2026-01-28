package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/httputil"
)

// AuthHandlers handles authentication-related HTTP requests
type AuthHandlers struct {
	db             *sql.DB
	tokenGenerator *auth.TokenGenerator
	tokenManager   *auth.TokenManager
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(db *sql.DB) *AuthHandlers {
	return &AuthHandlers{
		db:             db,
		tokenGenerator: auth.NewTokenGenerator(),
		tokenManager:   auth.NewTokenManager(),
	}
}

// RegisterRoutes registers authentication routes
func (h *AuthHandlers) RegisterRoutes(router *mux.Router) {
	// User routes
	router.HandleFunc("/auth/users", h.createUser).Methods("POST")
	router.HandleFunc("/auth/users/{id}", h.getUser).Methods("GET")
	router.HandleFunc("/auth/users/{id}", h.updateUser).Methods("PUT")
	router.HandleFunc("/auth/users/{id}", h.deleteUser).Methods("DELETE")

	// Token routes
	router.HandleFunc("/auth/tokens", h.createToken).Methods("POST")
	router.HandleFunc("/auth/tokens", h.listTokens).Methods("GET")
	router.HandleFunc("/auth/tokens/{id}", h.getToken).Methods("GET")
	router.HandleFunc("/auth/tokens/{id}", h.revokeToken).Methods("DELETE")

	// Organization routes
	router.HandleFunc("/auth/organizations", h.createOrganization).Methods("POST")
	router.HandleFunc("/auth/organizations/{id}", h.getOrganization).Methods("GET")
	router.HandleFunc("/auth/organizations/{id}/members", h.listOrganizationMembers).Methods("GET")
	router.HandleFunc("/auth/organizations/{id}/members", h.addOrganizationMember).Methods("POST")
	router.HandleFunc("/auth/organizations/{id}/members/{user_id}", h.removeOrganizationMember).Methods("DELETE")

	// Permission routes
	router.HandleFunc("/auth/modules/{module_name}/permissions", h.listModulePermissions).Methods("GET")
	router.HandleFunc("/auth/modules/{module_name}/permissions", h.grantModulePermission).Methods("POST")
	router.HandleFunc("/auth/modules/{module_name}/permissions/{permission_id}", h.revokeModulePermission).Methods("DELETE")
}

// createUser handles POST /auth/users
func (h *AuthHandlers) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		IsBot    bool   `json:"is_bot"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate required fields
	if req.Username == "" {
		http.Error(w, "username is required", http.StatusBadRequest)
		return
	}
	if !req.IsBot && req.Email == "" {
		http.Error(w, "email is required for non-bot users", http.StatusBadRequest)
		return
	}

	// Insert user into database
	var userID int64
	err := h.db.QueryRow(`
		INSERT INTO users (username, email, is_bot, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW())
		RETURNING id
	`, req.Username, req.Email, req.IsBot).Scan(&userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the created user
	user := &auth.User{}
	err = h.db.QueryRow(`
		SELECT id, username, email, is_bot, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w,user)
}

// getUser handles GET /auth/users/{id}
func (h *AuthHandlers) getUser(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	userID := vars["id"]

	user := &auth.User{}
	err := h.db.QueryRow(`
		SELECT id, username, email, is_bot, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`, userID).Scan(&user.ID, &user.Username, &user.Email, &user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "user not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, user)
}

// updateUser handles PUT /auth/users/{id}
func (h *AuthHandlers) updateUser(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	userID := vars["id"]

	var req struct {
		Email    *string `json:"email"`
		IsActive *bool   `json:"is_active"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Build dynamic update query
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	argCount := 1

	if req.Email != nil {
		query += ", email = $" + string(rune('0'+argCount))
		args = append(args, *req.Email)
		argCount++
	}
	if req.IsActive != nil {
		query += ", is_active = $" + string(rune('0'+argCount))
		args = append(args, *req.IsActive)
		argCount++
	}

	query += " WHERE id = $" + string(rune('0'+argCount))
	args = append(args, userID)

	_, err := h.db.Exec(query, args...)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, map[string]string{"status": "updated"})
}

// deleteUser handles DELETE /auth/users/{id}
func (h *AuthHandlers) deleteUser(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	userID := vars["id"]

	// Soft delete by setting is_active to false
	_, err := h.db.Exec(`
		UPDATE users SET is_active = false, updated_at = NOW()
		WHERE id = $1
	`, userID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// createToken handles POST /auth/tokens
func (h *AuthHandlers) createToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID    int64     `json:"user_id"`
		Name      string    `json:"name"`
		Scopes    []string  `json:"scopes"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate required fields
	if req.UserID == 0 {
		httputil.WriteBadRequest(w, "user_id is required")
		return
	}
	if !httputil.RequireNonEmpty(w, req.Name, "name") {
		return
	}

	// Generate token
	token, tokenHash, tokenPrefix, err := h.tokenGenerator.GenerateToken()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert scopes from []string to database array format
	scopeStrings := make([]string, len(req.Scopes))
	copy(scopeStrings, req.Scopes)

	// Insert token into database
	var tokenID int64
	err = h.db.QueryRow(`
		INSERT INTO api_tokens (user_id, token_hash, token_prefix, name, scopes, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id
	`, req.UserID, tokenHash, tokenPrefix, req.Name, scopeStrings, req.ExpiresAt).Scan(&tokenID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the token (only time it's visible)
	response := struct {
		ID           int64     `json:"id"`
		Token        string    `json:"token"`
		TokenPrefix  string    `json:"token_prefix"`
		Name         string    `json:"name"`
		Scopes       []string  `json:"scopes"`
		ExpiresAt    *time.Time `json:"expires_at"`
		CreatedAt    time.Time `json:"created_at"`
	}{
		ID:          tokenID,
		Token:       token,
		TokenPrefix: tokenPrefix,
		Name:        req.Name,
		Scopes:      req.Scopes,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now(),
	}

	httputil.WriteJSON(w, http.StatusCreated, response)
}

// listTokens handles GET /auth/tokens
func (h *AuthHandlers) listTokens(w http.ResponseWriter, r *http.Request) {
	// Get user_id from query params
	userIDStr := r.URL.Query().Get("user_id")
	if !httputil.RequireNonEmpty(w, userIDStr, "user_id query parameter") {
		return
	}

	rows, err := h.db.Query(`
		SELECT id, user_id, token_prefix, name, scopes, expires_at, last_used_at, created_at, revoked_at
		FROM api_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, userIDStr)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	tokens := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id          int64
			userID      int64
			prefix      string
			name        string
			scopes      []string
			expiresAt   *time.Time
			lastUsedAt  *time.Time
			createdAt   time.Time
			revokedAt   *time.Time
		)

		err := rows.Scan(&id, &userID, &prefix, &name, &scopes, &expiresAt, &lastUsedAt, &createdAt, &revokedAt)
		if err != nil {
			httputil.WriteInternalError(w, err)
			return
		}

		tokens = append(tokens, map[string]interface{}{
			"id":           id,
			"user_id":      userID,
			"token_prefix": prefix,
			"name":         name,
			"scopes":       scopes,
			"expires_at":   expiresAt,
			"last_used_at": lastUsedAt,
			"created_at":   createdAt,
		})
	}

	httputil.WriteSuccess(w, tokens)
}

// getToken handles GET /auth/tokens/{id}
func (h *AuthHandlers) getToken(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	tokenID := vars["id"]

	var (
		id         int64
		userID     int64
		prefix     string
		name       string
		scopes     []string
		expiresAt  *time.Time
		lastUsedAt *time.Time
		createdAt  time.Time
		revokedAt  *time.Time
	)

	err := h.db.QueryRow(`
		SELECT id, user_id, token_prefix, name, scopes, expires_at, last_used_at, created_at, revoked_at
		FROM api_tokens WHERE id = $1
	`, tokenID).Scan(&id, &userID, &prefix, &name, &scopes, &expiresAt, &lastUsedAt, &createdAt, &revokedAt)
	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "token not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	response := map[string]interface{}{
		"id":           id,
		"user_id":      userID,
		"token_prefix": prefix,
		"name":         name,
		"scopes":       scopes,
		"expires_at":   expiresAt,
		"last_used_at": lastUsedAt,
		"created_at":   createdAt,
		"revoked_at":   revokedAt,
	}

	httputil.WriteSuccess(w, response)
}

// revokeToken handles DELETE /auth/tokens/{id}
func (h *AuthHandlers) revokeToken(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	tokenID := vars["id"]

	_, err := h.db.Exec(`
		UPDATE api_tokens SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`, tokenID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// createOrganization handles POST /auth/organizations
func (h *AuthHandlers) createOrganization(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if !httputil.RequireNonEmpty(w, req.Name, "name") {
		return
	}

	var orgID int64
	err := h.db.QueryRow(`
		INSERT INTO organizations (name, description, is_active, created_at, updated_at)
		VALUES ($1, $2, true, NOW(), NOW())
		RETURNING id
	`, req.Name, req.Description).Scan(&orgID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org := &auth.Organization{}
	err = h.db.QueryRow(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w,org)
}

// getOrganization handles GET /auth/organizations/{id}
func (h *AuthHandlers) getOrganization(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	orgID := vars["id"]

	org := &auth.Organization{}
	err := h.db.QueryRow(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM organizations WHERE id = $1
	`, orgID).Scan(&org.ID, &org.Name, &org.Description, &org.IsActive, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		httputil.WriteNotFoundError(w, "organization not found")
		return
	}
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, org)
}

// listOrganizationMembers handles GET /auth/organizations/{id}/members
func (h *AuthHandlers) listOrganizationMembers(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	orgID := vars["id"]

	rows, err := h.db.Query(`
		SELECT om.user_id, om.role, om.joined_at, u.username, u.email
		FROM organization_members om
		JOIN users u ON om.user_id = u.id
		WHERE om.organization_id = $1
		ORDER BY om.joined_at DESC
	`, orgID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	members := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			userID   int64
			role     string
			joinedAt time.Time
			username string
			email    string
		)

		err := rows.Scan(&userID, &role, &joinedAt, &username, &email)
		if err != nil {
			httputil.WriteInternalError(w, err)
			return
		}

		members = append(members, map[string]interface{}{
			"user_id":   userID,
			"role":      role,
			"joined_at": joinedAt,
			"username":  username,
			"email":     email,
		})
	}

	httputil.WriteSuccess(w, members)
}

// addOrganizationMember handles POST /auth/organizations/{id}/members
func (h *AuthHandlers) addOrganizationMember(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	orgID := vars["id"]

	var req struct {
		UserID int64  `json:"user_id"`
		Role   string `json:"role"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate role
	validRoles := map[string]bool{
		string(auth.RoleAdmin):     true,
		string(auth.RoleDeveloper): true,
		string(auth.RoleViewer):    true,
	}
	if !validRoles[req.Role] {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	_, err := h.db.Exec(`
		INSERT INTO organization_members (organization_id, user_id, role, joined_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (organization_id, user_id) DO UPDATE SET role = $3
	`, orgID, req.UserID, req.Role)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w,map[string]string{"status": "added"})
}

// removeOrganizationMember handles DELETE /auth/organizations/{id}/members/{user_id}
func (h *AuthHandlers) removeOrganizationMember(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	orgID := vars["id"]
	userID := vars["user_id"]

	_, err := h.db.Exec(`
		DELETE FROM organization_members
		WHERE organization_id = $1 AND user_id = $2
	`, orgID, userID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// listModulePermissions handles GET /auth/modules/{module_name}/permissions
func (h *AuthHandlers) listModulePermissions(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	moduleName := vars["module_name"]

	rows, err := h.db.Query(`
		SELECT mp.id, mp.organization_id, mp.permission, mp.granted_at, o.name
		FROM module_permissions mp
		JOIN organizations o ON mp.organization_id = o.id
		WHERE mp.module_name = $1
		ORDER BY mp.granted_at DESC
	`, moduleName)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}
	defer rows.Close()

	permissions := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id        int64
			orgID     int64
			perm      string
			grantedAt time.Time
			orgName   string
		)

		err := rows.Scan(&id, &orgID, &perm, &grantedAt, &orgName)
		if err != nil {
			httputil.WriteInternalError(w, err)
			return
		}

		permissions = append(permissions, map[string]interface{}{
			"id":              id,
			"organization_id": orgID,
			"organization_name": orgName,
			"permission":      perm,
			"granted_at":      grantedAt,
		})
	}

	httputil.WriteSuccess(w, permissions)
}

// grantModulePermission handles POST /auth/modules/{module_name}/permissions
func (h *AuthHandlers) grantModulePermission(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	moduleName := vars["module_name"]

	var req struct {
		OrganizationID int64  `json:"organization_id"`
		Permission     string `json:"permission"`
	}

	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	// Validate permission
	validPerms := map[string]bool{
		string(auth.PermissionRead):   true,
		string(auth.PermissionWrite):  true,
		string(auth.PermissionDelete): true,
		string(auth.PermissionAdmin):  true,
	}
	if !validPerms[req.Permission] {
		http.Error(w, "invalid permission", http.StatusBadRequest)
		return
	}

	var permID int64
	err := h.db.QueryRow(`
		INSERT INTO module_permissions (module_name, organization_id, permission, granted_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id
	`, moduleName, req.OrganizationID, req.Permission).Scan(&permID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w,map[string]interface{}{
		"id":         permID,
		"status":     "granted",
		"permission": req.Permission,
	})
}

// revokeModulePermission handles DELETE /auth/modules/{module_name}/permissions/{permission_id}
func (h *AuthHandlers) revokeModulePermission(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	permID := vars["permission_id"]

	_, err := h.db.Exec(`
		DELETE FROM module_permissions WHERE id = $1
	`, permID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
