package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
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

// CreateOrganization creates a new organization
func (h *OrgHandlers) CreateOrganization(w http.ResponseWriter, r *http.Request) {
	// Get authenticated user
	authCtx := r.Context().Value("auth").(*auth.AuthContext)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req orgs.CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	org := &orgs.Organization{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		OwnerID:     &authCtx.User.ID,
		PlanTier:    req.PlanTier,
		Settings:    req.Settings,
	}

	if err := h.orgService.CreateOrganization(org); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Add creator as admin member
	if err := h.orgService.AddMember(org.ID, authCtx.User.ID, auth.RoleAdmin, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(org)
}

// ListOrganizations lists organizations for the authenticated user
func (h *OrgHandlers) ListOrganizations(w http.ResponseWriter, r *http.Request) {
	authCtx := r.Context().Value("auth").(*auth.AuthContext)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orgs, err := h.orgService.ListOrganizations(authCtx.User.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orgs)
}

// GetOrganization retrieves an organization by ID
func (h *OrgHandlers) GetOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	org, err := h.orgService.GetOrganization(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

// UpdateOrganization updates an organization
func (h *OrgHandlers) UpdateOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var req orgs.UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.orgService.UpdateOrganization(id, &req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	org, err := h.orgService.GetOrganization(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(org)
}

// DeleteOrganization deletes an organization
func (h *OrgHandlers) DeleteOrganization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	if err := h.orgService.DeleteOrganization(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetQuotas retrieves quotas for an organization
func (h *OrgHandlers) GetQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	quotas, err := h.orgService.GetQuotas(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotas)
}

// UpdateQuotas updates quotas for an organization
func (h *OrgHandlers) UpdateQuotas(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	var quotas orgs.OrgQuotas
	if err := json.NewDecoder(r.Body).Decode(&quotas); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.orgService.UpdateQuotas(id, &quotas); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedQuotas, err := h.orgService.GetQuotas(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedQuotas)
}

// GetUsage retrieves current usage for an organization
func (h *OrgHandlers) GetUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	usage, err := h.orgService.GetUsage(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usage)
}

// GetUsageHistory retrieves usage history for an organization
func (h *OrgHandlers) GetUsageHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	limit := 12 // Default to 12 months
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	history, err := h.orgService.GetUsageHistory(id, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// ListMembers lists members of an organization
func (h *OrgHandlers) ListMembers(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	members, err := h.orgService.ListMembers(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// AddMember adds a member to an organization
func (h *OrgHandlers) AddMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	authCtx := r.Context().Value("auth").(*auth.AuthContext)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		UserID int64     `json:"user_id"`
		Role   auth.Role `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.orgService.AddMember(id, req.UserID, req.Role, &authCtx.User.ID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// UpdateMember updates a member's role
func (h *OrgHandlers) UpdateMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req orgs.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.orgService.UpdateMemberRole(id, userID, req.Role); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember removes a member from an organization
func (h *OrgHandlers) RemoveMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(vars["user_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	if err := h.orgService.RemoveMember(id, userID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateInvitation creates an invitation
func (h *OrgHandlers) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	authCtx := r.Context().Value("auth").(*auth.AuthContext)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req orgs.InviteMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	invitation := &orgs.OrgInvitation{
		OrgID:     id,
		Email:     req.Email,
		Role:      req.Role,
		InvitedBy: authCtx.User.ID,
	}

	if err := h.orgService.CreateInvitation(invitation); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(invitation)
}

// ListInvitations lists invitations for an organization
func (h *OrgHandlers) ListInvitations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid organization ID", http.StatusBadRequest)
		return
	}

	invitations, err := h.orgService.ListInvitations(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(invitations)
}

// RevokeInvitation revokes an invitation
func (h *OrgHandlers) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	invitationID, err := strconv.ParseInt(vars["invitation_id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return
	}

	if err := h.orgService.RevokeInvitation(invitationID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AcceptInvitation accepts an invitation
func (h *OrgHandlers) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]

	authCtx := r.Context().Value("auth").(*auth.AuthContext)
	if authCtx == nil || authCtx.User == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.orgService.AcceptInvitation(token, authCtx.User.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
