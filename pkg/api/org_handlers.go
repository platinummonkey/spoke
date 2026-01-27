package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/httputil"
	"github.com/platinummonkey/spoke/pkg/orgs"
)

// OrgHandlers handles organization-related HTTP requests
type OrgHandlers struct {
	orgService orgs.Service
}

// NewOrgHandlers creates a new OrgHandlers
func NewOrgHandlers(orgService orgs.Service) *OrgHandlers {
	return &OrgHandlers{
		orgService: orgService,
	}
}

// RegisterRoutes registers organization routes
func (h *OrgHandlers) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/orgs", h.CreateOrganization).Methods("POST")
	router.HandleFunc("/orgs", h.ListOrganizations).Methods("GET")
	router.HandleFunc("/orgs/{id}", h.GetOrganization).Methods("GET")
	router.HandleFunc("/orgs/{id}", h.UpdateOrganization).Methods("PUT")
	router.HandleFunc("/orgs/{id}", h.DeleteOrganization).Methods("DELETE")

	// Quotas
	router.HandleFunc("/orgs/{id}/quotas", h.GetQuotas).Methods("GET")
	router.HandleFunc("/orgs/{id}/quotas", h.UpdateQuotas).Methods("PUT")

	// Usage
	router.HandleFunc("/orgs/{id}/usage", h.GetUsage).Methods("GET")
	router.HandleFunc("/orgs/{id}/usage/history", h.GetUsageHistory).Methods("GET")

	// Members
	router.HandleFunc("/orgs/{id}/members", h.ListMembers).Methods("GET")
	router.HandleFunc("/orgs/{id}/members", h.AddMember).Methods("POST")
	router.HandleFunc("/orgs/{id}/members/{user_id}", h.UpdateMember).Methods("PUT")
	router.HandleFunc("/orgs/{id}/members/{user_id}", h.RemoveMember).Methods("DELETE")

	// Invitations
	router.HandleFunc("/orgs/{id}/invitations", h.CreateInvitation).Methods("POST")
	router.HandleFunc("/orgs/{id}/invitations", h.ListInvitations).Methods("GET")
	router.HandleFunc("/orgs/{id}/invitations/{invitation_id}", h.RevokeInvitation).Methods("DELETE")
	router.HandleFunc("/invitations/{token}/accept", h.AcceptInvitation).Methods("POST")
}

// getAuthContext is a helper to extract and validate auth context
func getAuthContext(r *http.Request) (*auth.AuthContext, bool) {
	authCtx, ok := r.Context().Value("auth").(*auth.AuthContext)
	if !ok || authCtx == nil || authCtx.User == nil {
		return nil, false
	}
	return authCtx, true
}

// CreateOrganization creates a new organization
func (h *OrgHandlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := getAuthContext(r)
	if !ok {
		httputil.WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req orgs.CreateOrgRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	org := &orgs.Organization{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		OwnerID:     &authCtx.User.ID,
		QuotaTier:   req.QuotaTier,
		Settings:    req.Settings,
	}

	if err := h.orgService.CreateOrganization(org); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	// Add creator as admin member
	if err := h.orgService.AddMember(org.ID, authCtx.User.ID, auth.RoleAdmin, nil); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, org)
}

// ListOrganizations lists organizations for the authenticated user
func (h *OrgHandlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := getAuthContext(r)
	if !ok {
		httputil.WriteUnauthorized(w, "Unauthorized")
		return
	}

	orgs, err := h.orgService.ListOrganizations(authCtx.User.ID)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, orgs)
}

// GetOrganization retrieves an organization by ID
func (h *OrgHandlers) GetOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	org, err := h.orgService.GetOrganization(id)
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, org)
}

// UpdateOrganization updates an organization
func (h *OrgHandlers) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var req orgs.UpdateOrgRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if err := h.orgService.UpdateOrganization(id, &req); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	org, err := h.orgService.GetOrganization(id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, org)
}

// DeleteOrganization deletes an organization
func (h *OrgHandlers) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	if err := h.orgService.DeleteOrganization(id); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// GetQuotas retrieves quotas for an organization
func (h *OrgHandlers) GetQuotas(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	quotas, err := h.orgService.GetQuotas(id)
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, quotas)
}

// UpdateQuotas updates quotas for an organization
func (h *OrgHandlers) UpdateQuotas(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	var quotas orgs.OrgQuotas
	if !httputil.ParseJSONOrError(w, r, &quotas) {
		return
	}

	if err := h.orgService.UpdateQuotas(id, &quotas); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	updatedQuotas, err := h.orgService.GetQuotas(id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, updatedQuotas)
}

// GetUsage retrieves current usage for an organization
func (h *OrgHandlers) GetUsage(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	usage, err := h.orgService.GetUsage(id)
	if err != nil {
		httputil.WriteNotFoundError(w, err.Error())
		return
	}

	httputil.WriteSuccess(w, usage)
}

// GetUsageHistory retrieves usage history for an organization
func (h *OrgHandlers) GetUsageHistory(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	limit, err := httputil.ParseQueryInt(r, "limit", 12)
	if err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}

	history, err := h.orgService.GetUsageHistory(id, limit)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, history)
}

// ListMembers lists members of an organization
func (h *OrgHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	members, err := h.orgService.ListMembers(id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, members)
}

// AddMember adds a member to an organization
func (h *OrgHandlers) AddMember(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	authCtx, ok := getAuthContext(r)
	if !ok {
		httputil.WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req struct {
		UserID int64     `json:"user_id"`
		Role   auth.Role `json:"role"`
	}
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if err := h.orgService.AddMember(id, req.UserID, req.Role, &authCtx.User.ID); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UpdateMember updates a member's role
func (h *OrgHandlers) UpdateMember(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	userID, ok := httputil.ParsePathInt64OrError(w, r, "user_id")
	if !ok {
		return
	}

	var req orgs.UpdateMemberRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	if err := h.orgService.UpdateMemberRole(id, userID, req.Role); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// RemoveMember removes a member from an organization
func (h *OrgHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	userID, ok := httputil.ParsePathInt64OrError(w, r, "user_id")
	if !ok {
		return
	}

	if err := h.orgService.RemoveMember(id, userID); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// CreateInvitation creates an invitation
func (h *OrgHandlers) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	authCtx, ok := getAuthContext(r)
	if !ok {
		httputil.WriteUnauthorized(w, "Unauthorized")
		return
	}

	var req orgs.InviteMemberRequest
	if !httputil.ParseJSONOrError(w, r, &req) {
		return
	}

	invitation := &orgs.OrgInvitation{
		OrgID:     id,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: authCtx.User.ID,
	}

	if err := h.orgService.CreateInvitation(invitation); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteCreated(w, invitation)
}

// ListInvitations lists invitations for an organization
func (h *OrgHandlers) ListInvitations(w http.ResponseWriter, r *http.Request) {
	id, ok := httputil.ParsePathInt64OrError(w, r, "id")
	if !ok {
		return
	}

	invitations, err := h.orgService.ListInvitations(id)
	if err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteSuccess(w, invitations)
}

// RevokeInvitation revokes an invitation
func (h *OrgHandlers) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	invitationID, ok := httputil.ParsePathInt64OrError(w, r, "invitation_id")
	if !ok {
		return
	}

	if err := h.orgService.RevokeInvitation(invitationID); err != nil {
		httputil.WriteInternalError(w, err)
		return
	}

	httputil.WriteNoContent(w)
}

// AcceptInvitation accepts an invitation
func (h *OrgHandlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	vars := httputil.GetPathVars(r)
	token := vars["token"]

	authCtx, ok := getAuthContext(r)
	if !ok {
		httputil.WriteUnauthorized(w, "Unauthorized")
		return
	}

	if err := h.orgService.AcceptInvitation(token, authCtx.User.ID); err != nil {
		httputil.WriteBadRequest(w, err.Error())
		return
	}

	httputil.WriteNoContent(w)
}
