package rbac

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/audit"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/middleware"
)

// Handlers provides HTTP handlers for RBAC operations
type Handlers struct {
	store       *Store
	checker     *PermissionChecker
	auditLogger audit.Logger
}

// NewHandlers creates new RBAC handlers
func NewHandlers(db *sql.DB, auditLogger audit.Logger) *Handlers {
	return &Handlers{
		store:       NewStore(db),
		checker:     NewPermissionChecker(db, 5*time.Minute),
		auditLogger: auditLogger,
	}
}

// RegisterRoutes registers all RBAC routes
func (h *Handlers) RegisterRoutes(router *mux.Router) {
	// Role management
	router.HandleFunc("/rbac/roles", h.CreateRole).Methods("POST")
	router.HandleFunc("/rbac/roles", h.ListRoles).Methods("GET")
	router.HandleFunc("/rbac/roles/{id}", h.GetRole).Methods("GET")
	router.HandleFunc("/rbac/roles/{id}", h.UpdateRole).Methods("PUT")
	router.HandleFunc("/rbac/roles/{id}", h.DeleteRole).Methods("DELETE")

	// User role assignments
	router.HandleFunc("/rbac/users/{id}/roles", h.AssignRoleToUser).Methods("POST")
	router.HandleFunc("/rbac/users/{id}/roles", h.GetUserRoles).Methods("GET")
	router.HandleFunc("/rbac/users/{id}/roles/{role_id}", h.RevokeRoleFromUser).Methods("DELETE")
	router.HandleFunc("/rbac/users/{id}/permissions", h.GetUserPermissions).Methods("GET")

	// Permission checking
	router.HandleFunc("/rbac/check", h.CheckPermission).Methods("POST")

	// Team management
	router.HandleFunc("/rbac/teams", h.CreateTeam).Methods("POST")
	router.HandleFunc("/rbac/teams", h.ListTeams).Methods("GET")
	router.HandleFunc("/rbac/teams/{id}", h.GetTeam).Methods("GET")
	router.HandleFunc("/rbac/teams/{id}", h.UpdateTeam).Methods("PUT")
	router.HandleFunc("/rbac/teams/{id}", h.DeleteTeam).Methods("DELETE")

	// Team member management
	router.HandleFunc("/rbac/teams/{id}/members", h.AddTeamMember).Methods("POST")
	router.HandleFunc("/rbac/teams/{id}/members", h.GetTeamMembers).Methods("GET")
	router.HandleFunc("/rbac/teams/{id}/members/{user_id}", h.RemoveTeamMember).Methods("DELETE")

	// Team role assignments
	router.HandleFunc("/rbac/teams/{id}/roles", h.AssignRoleToTeam).Methods("POST")
	router.HandleFunc("/rbac/teams/{id}/roles/{role_id}", h.RevokeRoleFromTeam).Methods("DELETE")

	// Role templates
	router.HandleFunc("/rbac/templates", h.GetRoleTemplates).Methods("GET")
}

// CreateRole creates a new custom role
func (h *Handlers) CreateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)

	// Parse request
	var req struct {
		Name           string       `json:"name"`
		DisplayName    string       `json:"display_name"`
		Description    string       `json:"description"`
		OrganizationID *int64       `json:"organization_id,omitempty"`
		Permissions    []Permission `json:"permissions"`
		ParentRoleID   *int64       `json:"parent_role_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" || req.DisplayName == "" {
		http.Error(w, "Name and display_name are required", http.StatusBadRequest)
		return
	}

	// Create role
	role := &Role{
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		OrganizationID: req.OrganizationID,
		Permissions:    req.Permissions,
		ParentRoleID:   req.ParentRoleID,
		IsBuiltIn:      false,
		IsCustom:       true,
	}

	if authCtx != nil && authCtx.User != nil {
		role.CreatedBy = &authCtx.User.ID
	}

	if err := h.store.CreateRole(ctx, role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, audit.EventTypeAuthzPermissionGrant, "role", strconv.FormatInt(role.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(role)
}

// ListRoles lists all roles
func (h *Handlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get organization_id from query params
	var organizationID *int64
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err == nil {
			organizationID = &orgID
		}
	}

	roles, err := h.store.ListRoles(ctx, organizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

// GetRole retrieves a specific role
func (h *Handlers) GetRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	roleID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	role, err := h.store.GetRole(ctx, roleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

// UpdateRole updates an existing role
func (h *Handlers) UpdateRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	roleID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	// Get existing role
	role, err := h.store.GetRole(ctx, roleID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Check if role is built-in
	if role.IsBuiltIn {
		http.Error(w, "Cannot modify built-in roles", http.StatusForbidden)
		return
	}

	// Parse request
	var req struct {
		DisplayName  string       `json:"display_name"`
		Description  string       `json:"description"`
		Permissions  []Permission `json:"permissions"`
		ParentRoleID *int64       `json:"parent_role_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update role
	role.DisplayName = req.DisplayName
	role.Description = req.Description
	role.Permissions = req.Permissions
	role.ParentRoleID = req.ParentRoleID

	if err := h.store.UpdateRole(ctx, role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, audit.EventTypeAuthzRoleChange, "role", strconv.FormatInt(role.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(role)
}

// DeleteRole deletes a role
func (h *Handlers) DeleteRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	roleID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteRole(ctx, roleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, audit.EventTypeAuthzPermissionRevoke, "role", strconv.FormatInt(roleID, 10), true, nil)

	w.WriteHeader(http.StatusNoContent)
}

// AssignRoleToUser assigns a role to a user
func (h *Handlers) AssignRoleToUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		RoleID         int64           `json:"role_id"`
		Scope          PermissionScope `json:"scope"`
		ResourceID     *string         `json:"resource_id,omitempty"`
		OrganizationID *int64          `json:"organization_id,omitempty"`
		ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create user role
	userRole := &UserRole{
		UserID:         userID,
		RoleID:         req.RoleID,
		Scope:          req.Scope,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
		ExpiresAt:      req.ExpiresAt,
	}

	if authCtx != nil && authCtx.User != nil {
		userRole.GrantedBy = &authCtx.User.ID
	}

	if err := h.store.AssignRoleToUser(ctx, userRole); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate permission cache
	h.checker.InvalidateCache(ctx, userID)

	// Audit log
	h.logAudit(ctx, authCtx, audit.EventTypeAuthzPermissionGrant, "user_role", strconv.FormatInt(userRole.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userRole)
}

// GetUserRoles retrieves all roles for a user
func (h *Handlers) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get organization_id from query params
	var organizationID *int64
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err == nil {
			organizationID = &orgID
		}
	}

	userRoles, err := h.store.GetUserRoles(ctx, userID, organizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Enhance with role details
	type UserRoleWithDetails struct {
		UserRole
		Role *Role `json:"role"`
	}

	rolesWithDetails := make([]UserRoleWithDetails, len(userRoles))
	for i, ur := range userRoles {
		role, err := h.store.GetRole(ctx, ur.RoleID)
		if err != nil {
			continue
		}
		rolesWithDetails[i] = UserRoleWithDetails{
			UserRole: ur,
			Role:     role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rolesWithDetails)
}

// RevokeRoleFromUser revokes a role from a user
func (h *Handlers) RevokeRoleFromUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	userRoleID, err := strconv.ParseInt(vars["role_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := h.store.RevokeRoleFromUser(ctx, userRoleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate permission cache
	h.checker.InvalidateCache(ctx, userID)

	// Audit log
	h.logAudit(ctx, authCtx, audit.EventTypeAuthzPermissionRevoke, "user_role", strconv.FormatInt(userRoleID, 10), true, nil)

	w.WriteHeader(http.StatusNoContent)
}

// GetUserPermissions retrieves all effective permissions for a user
func (h *Handlers) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	userID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get organization_id from query params
	var organizationID *int64
	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
		if err == nil {
			organizationID = &orgID
		}
	}

	// Get resource_id from query params
	var resourceID *string
	if resID := r.URL.Query().Get("resource_id"); resID != "" {
		resourceID = &resID
	}

	permissions, err := h.checker.GetEffectivePermissions(ctx, userID, organizationID, resourceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":     userID,
		"permissions": permissions,
	})
}

// CheckPermission checks if a user has a specific permission
func (h *Handlers) CheckPermission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request
	var check PermissionCheck
	if err := json.NewDecoder(r.Body).Decode(&check); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.checker.CheckPermission(ctx, check)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CreateTeam creates a new team
func (h *Handlers) CreateTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)

	// Parse request
	var req struct {
		OrganizationID int64  `json:"organization_id"`
		Name           string `json:"name"`
		DisplayName    string `json:"display_name"`
		Description    string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate
	if req.Name == "" || req.DisplayName == "" {
		http.Error(w, "Name and display_name are required", http.StatusBadRequest)
		return
	}

	// Create team
	team := &Team{
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
	}

	if authCtx != nil && authCtx.User != nil {
		team.CreatedBy = &authCtx.User.ID
	}

	if err := h.store.CreateTeam(ctx, team); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.create", "team", strconv.FormatInt(team.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(team)
}

// ListTeams lists all teams for an organization
func (h *Handlers) ListTeams(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	organizationID, err := strconv.ParseInt(r.URL.Query().Get("organization_id"), 10, 64)
	if err != nil {
		http.Error(w, "organization_id is required", http.StatusBadRequest)
		return
	}

	teams, err := h.store.ListTeams(ctx, organizationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// GetTeam retrieves a specific team
func (h *Handlers) GetTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	team, err := h.store.GetTeam(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// UpdateTeam updates a team
func (h *Handlers) UpdateTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	// Get existing team
	team, err := h.store.GetTeam(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Parse request
	var req struct {
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update team
	team.DisplayName = req.DisplayName
	team.Description = req.Description

	if err := h.store.UpdateTeam(ctx, team); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.update", "team", strconv.FormatInt(team.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(team)
}

// DeleteTeam deletes a team
func (h *Handlers) DeleteTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	if err := h.store.DeleteTeam(ctx, teamID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.delete", "team", strconv.FormatInt(teamID, 10), true, nil)

	w.WriteHeader(http.StatusNoContent)
}

// AddTeamMember adds a user to a team
func (h *Handlers) AddTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		UserID int64  `json:"user_id"`
		RoleID *int64 `json:"role_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create team member
	member := &TeamMember{
		TeamID: teamID,
		UserID: req.UserID,
		RoleID: req.RoleID,
	}

	if authCtx != nil && authCtx.User != nil {
		member.AddedBy = &authCtx.User.ID
	}

	if err := h.store.AddTeamMember(ctx, member); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.member_add", "team_member", strconv.FormatInt(member.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(member)
}

// GetTeamMembers retrieves all members of a team
func (h *Handlers) GetTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	members, err := h.store.GetTeamMembers(ctx, teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// RemoveTeamMember removes a user from a team
func (h *Handlers) RemoveTeamMember(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.store.RemoveTeamMember(ctx, teamID, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.member_remove", "team_member", strconv.FormatInt(teamID, 10), true, nil)

	w.WriteHeader(http.StatusNoContent)
}

// AssignRoleToTeam assigns a role to a team
func (h *Handlers) AssignRoleToTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid team ID", http.StatusBadRequest)
		return
	}

	// Parse request
	var req struct {
		RoleID         int64           `json:"role_id"`
		Scope          PermissionScope `json:"scope"`
		ResourceID     *string         `json:"resource_id,omitempty"`
		OrganizationID *int64          `json:"organization_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create team role
	teamRole := &TeamRole{
		TeamID:         teamID,
		RoleID:         req.RoleID,
		Scope:          req.Scope,
		ResourceID:     req.ResourceID,
		OrganizationID: req.OrganizationID,
	}

	if authCtx != nil && authCtx.User != nil {
		teamRole.GrantedBy = &authCtx.User.ID
	}

	if err := h.store.AssignRoleToTeam(ctx, teamRole); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.role_assign", "team_role", strconv.FormatInt(teamRole.ID, 10), true, nil)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(teamRole)
}

// RevokeRoleFromTeam revokes a role from a team
func (h *Handlers) RevokeRoleFromTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	authCtx := middleware.GetAuthContext(r)
	vars := mux.Vars(r)

	teamRoleID, err := strconv.ParseInt(vars["role_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid role ID", http.StatusBadRequest)
		return
	}

	if err := h.store.RevokeRoleFromTeam(ctx, teamRoleID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit log
	h.logAudit(ctx, authCtx, "team.role_revoke", "team_role", strconv.FormatInt(teamRoleID, 10), true, nil)

	w.WriteHeader(http.StatusNoContent)
}

// GetRoleTemplates returns common role templates
func (h *Handlers) GetRoleTemplates(w http.ResponseWriter, r *http.Request) {
	templates := CommonRoleTemplates()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// logAudit logs an audit event
func (h *Handlers) logAudit(ctx context.Context, authCtx *auth.AuthContext, eventType audit.EventType, resourceType, resourceID string, success bool, err error) {
	if h.auditLogger == nil {
		return
	}

	event := &audit.AuditEvent{
		Timestamp:    time.Now(),
		EventType:    eventType,
		ResourceType: audit.ResourceType(resourceType),
		ResourceID:   resourceID,
	}

	if authCtx != nil && authCtx.User != nil {
		event.UserID = &authCtx.User.ID
		event.Username = authCtx.User.Username
	}

	if success {
		event.Status = audit.EventStatusSuccess
	} else {
		event.Status = audit.EventStatusFailure
		if err != nil {
			event.ErrorMessage = err.Error()
		}
	}

	h.auditLogger.Log(ctx, event)
}
