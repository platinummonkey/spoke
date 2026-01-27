package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/orgs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOrgService is a mock implementation of orgs.Service for testing
type mockOrgService struct {
	createOrganizationFunc  func(org *orgs.Organization) error
	getOrganizationFunc     func(id int64) (*orgs.Organization, error)
	listOrganizationsFunc   func(userID int64) ([]*orgs.Organization, error)
	updateOrganizationFunc  func(id int64, updates *orgs.UpdateOrgRequest) error
	deleteOrganizationFunc  func(id int64) error
	getQuotasFunc           func(orgID int64) (*orgs.OrgQuotas, error)
	updateQuotasFunc        func(orgID int64, quotas *orgs.OrgQuotas) error
	getUsageFunc            func(orgID int64) (*orgs.OrgUsage, error)
	getUsageHistoryFunc     func(orgID int64, limit int) ([]*orgs.OrgUsage, error)
	listMembersFunc         func(orgID int64) ([]*orgs.OrgMember, error)
	addMemberFunc           func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error
	updateMemberFunc        func(orgID int64, userID int64, role auth.Role) error
	removeMemberFunc        func(orgID int64, userID int64) error
	createInvitationFunc    func(inv *orgs.OrgInvitation) error
	listInvitationsFunc     func(orgID int64) ([]*orgs.OrgInvitation, error)
	revokeInvitationFunc    func(invitationID int64) error
	acceptInvitationFunc    func(token string, userID int64) (*orgs.Organization, error)
}

func (m *mockOrgService) CreateOrganization(org *orgs.Organization) error {
	if m.createOrganizationFunc != nil {
		return m.createOrganizationFunc(org)
	}
	return nil
}

func (m *mockOrgService) GetOrganization(id int64) (*orgs.Organization, error) {
	if m.getOrganizationFunc != nil {
		return m.getOrganizationFunc(id)
	}
	return &orgs.Organization{ID: id}, nil
}

func (m *mockOrgService) GetOrganizationBySlug(slug string) (*orgs.Organization, error) {
	return &orgs.Organization{}, nil
}

func (m *mockOrgService) ListOrganizations(userID int64) ([]*orgs.Organization, error) {
	if m.listOrganizationsFunc != nil {
		return m.listOrganizationsFunc(userID)
	}
	return []*orgs.Organization{}, nil
}

func (m *mockOrgService) UpdateOrganization(id int64, updates *orgs.UpdateOrgRequest) error {
	if m.updateOrganizationFunc != nil {
		return m.updateOrganizationFunc(id, updates)
	}
	return nil
}

func (m *mockOrgService) DeleteOrganization(id int64) error {
	if m.deleteOrganizationFunc != nil {
		return m.deleteOrganizationFunc(id)
	}
	return nil
}

func (m *mockOrgService) GetQuotas(orgID int64) (*orgs.OrgQuotas, error) {
	if m.getQuotasFunc != nil {
		return m.getQuotasFunc(orgID)
	}
	return &orgs.OrgQuotas{}, nil
}

func (m *mockOrgService) UpdateQuotas(orgID int64, quotas *orgs.OrgQuotas) error {
	if m.updateQuotasFunc != nil {
		return m.updateQuotasFunc(orgID, quotas)
	}
	return nil
}

func (m *mockOrgService) GetDefaultQuotas(quotaTier orgs.QuotaTier) *orgs.OrgQuotas {
	return &orgs.OrgQuotas{}
}

func (m *mockOrgService) GetUsage(orgID int64) (*orgs.OrgUsage, error) {
	if m.getUsageFunc != nil {
		return m.getUsageFunc(orgID)
	}
	return &orgs.OrgUsage{}, nil
}

func (m *mockOrgService) GetUsageHistory(orgID int64, limit int) ([]*orgs.OrgUsage, error) {
	if m.getUsageHistoryFunc != nil {
		return m.getUsageHistoryFunc(orgID, limit)
	}
	return []*orgs.OrgUsage{}, nil
}

func (m *mockOrgService) ResetUsagePeriod(orgID int64) error {
	return nil
}

func (m *mockOrgService) ListMembers(orgID int64) ([]*orgs.OrgMember, error) {
	if m.listMembersFunc != nil {
		return m.listMembersFunc(orgID)
	}
	return []*orgs.OrgMember{}, nil
}

func (m *mockOrgService) AddMember(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(orgID, userID, role, invitedBy)
	}
	return nil
}

func (m *mockOrgService) UpdateMember(orgID int64, userID int64, role auth.Role) error {
	if m.updateMemberFunc != nil {
		return m.updateMemberFunc(orgID, userID, role)
	}
	return nil
}

func (m *mockOrgService) RemoveMember(orgID int64, userID int64) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(orgID, userID)
	}
	return nil
}

func (m *mockOrgService) CreateInvitation(inv *orgs.OrgInvitation) error {
	if m.createInvitationFunc != nil {
		return m.createInvitationFunc(inv)
	}
	return nil
}

func (m *mockOrgService) ListInvitations(orgID int64) ([]*orgs.OrgInvitation, error) {
	if m.listInvitationsFunc != nil {
		return m.listInvitationsFunc(orgID)
	}
	return []*orgs.OrgInvitation{}, nil
}

func (m *mockOrgService) RevokeInvitation(invitationID int64) error {
	if m.revokeInvitationFunc != nil {
		return m.revokeInvitationFunc(invitationID)
	}
	return nil
}

func (m *mockOrgService) AcceptInvitation(token string, userID int64) error {
	if m.acceptInvitationFunc != nil {
		_, err := m.acceptInvitationFunc(token, userID)
		return err
	}
	return nil
}

func (m *mockOrgService) CleanupExpiredInvitations() error {
	return nil
}

func (m *mockOrgService) CheckAPIRateLimit(orgID int64) error {
	return nil
}

func (m *mockOrgService) CheckCompileJobQuota(orgID int64) error {
	return nil
}

func (m *mockOrgService) IncrementAPICall(orgID int64) error {
	return nil
}

func (m *mockOrgService) IncrementCompileJob(orgID int64) error {
	return nil
}

func (m *mockOrgService) CheckModuleQuota(orgID int64) error {
	return nil
}

func (m *mockOrgService) CheckVersionQuota(orgID int64, moduleName string) error {
	return nil
}

func (m *mockOrgService) CheckStorageQuota(orgID int64, additionalBytes int64) error {
	return nil
}

func (m *mockOrgService) IncrementStorage(orgID int64, bytes int64) error {
	return nil
}

func (m *mockOrgService) IncrementCompileJobs(orgID int64) error {
	return nil
}

func (m *mockOrgService) IncrementAPIRequests(orgID int64) error {
	return nil
}

func (m *mockOrgService) DecrementModules(orgID int64) error {
	return nil
}

func (m *mockOrgService) DecrementVersions(orgID int64) error {
	return nil
}

func (m *mockOrgService) DecrementStorage(orgID int64, bytes int64) error {
	return nil
}

func (m *mockOrgService) IncrementModules(orgID int64) error {
	return nil
}

func (m *mockOrgService) IncrementVersions(orgID int64) error {
	return nil
}

func (m *mockOrgService) UpdateMemberRole(orgID, userID int64, role auth.Role) error {
	if m.updateMemberFunc != nil {
		return m.updateMemberFunc(orgID, userID, role)
	}
	return nil
}

func (m *mockOrgService) GetMember(orgID, userID int64) (*orgs.OrgMember, error) {
	return &orgs.OrgMember{}, nil
}

func (m *mockOrgService) GetInvitation(token string) (*orgs.OrgInvitation, error) {
	return &orgs.OrgInvitation{}, nil
}

// createAuthContext creates a test auth context
func createAuthContext(userID int64, username string) *auth.AuthContext {
	return &auth.AuthContext{
		User: &auth.User{
			ID:       userID,
			Username: username,
		},
	}
}

// createAuthRequest creates a request with auth context
func createAuthRequest(method, url string, body []byte, authCtx *auth.AuthContext) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	if authCtx != nil {
		ctx := context.WithValue(req.Context(), "auth", authCtx)
		req = req.WithContext(ctx)
	}
	return req
}

// TestNewOrgHandlers verifies handler initialization
func TestNewOrgHandlers(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.orgService)
}

// TestOrgHandlers_RegisterRoutes verifies all routes are registered
func TestOrgHandlers_RegisterRoutes(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		{"POST", "/orgs"},
		{"GET", "/orgs"},
		{"GET", "/orgs/123"},
		{"PUT", "/orgs/123"},
		{"DELETE", "/orgs/123"},
		{"GET", "/orgs/123/quotas"},
		{"PUT", "/orgs/123/quotas"},
		{"GET", "/orgs/123/usage"},
		{"GET", "/orgs/123/usage/history"},
		{"GET", "/orgs/123/members"},
		{"POST", "/orgs/123/members"},
		{"PUT", "/orgs/123/members/456"},
		{"DELETE", "/orgs/123/members/456"},
		{"POST", "/orgs/123/invitations"},
		{"GET", "/orgs/123/invitations"},
		{"DELETE", "/orgs/123/invitations/789"},
		{"POST", "/invitations/test-token/accept"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			var match mux.RouteMatch
			matched := router.Match(req, &match)
			assert.True(t, matched, "Route %s %s should be registered", tt.method, tt.path)
		})
	}
}

// TestCreateOrganization_Success tests successful organization creation
func TestCreateOrganization_Success(t *testing.T) {
	service := &mockOrgService{
		createOrganizationFunc: func(org *orgs.Organization) error {
			org.ID = 123
			return nil
		},
		addMemberFunc: func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
			assert.Equal(t, int64(123), orgID)
			assert.Equal(t, int64(1), userID)
			assert.Equal(t, auth.RoleAdmin, role)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	orgReq := orgs.CreateOrgRequest{
		Name:        "test-org",
		DisplayName: "Test Organization",
		Description: "Test description",
	}
	body, err := json.Marshal(orgReq)
	require.NoError(t, err)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs", body, authCtx)
	w := httptest.NewRecorder()

	handlers.CreateOrganization(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "test-org")
}

// TestCreateOrganization_Unauthorized tests organization creation without auth
func TestCreateOrganization_Unauthorized(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	orgReq := orgs.CreateOrgRequest{Name: "test-org"}
	body, err := json.Marshal(orgReq)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/orgs", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handlers.CreateOrganization(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestCreateOrganization_InvalidJSON tests organization creation with invalid JSON
func TestCreateOrganization_InvalidJSON(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs", []byte("invalid json"), authCtx)
	w := httptest.NewRecorder()

	handlers.CreateOrganization(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateOrganization_CreateError tests organization creation error
func TestCreateOrganization_CreateError(t *testing.T) {
	service := &mockOrgService{
		createOrganizationFunc: func(org *orgs.Organization) error {
			return errors.New("database error")
		},
	}
	handlers := NewOrgHandlers(service)

	orgReq := orgs.CreateOrgRequest{Name: "test-org"}
	body, err := json.Marshal(orgReq)
	require.NoError(t, err)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs", body, authCtx)
	w := httptest.NewRecorder()

	handlers.CreateOrganization(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCreateOrganization_AddMemberError tests organization creation with member add error
func TestCreateOrganization_AddMemberError(t *testing.T) {
	service := &mockOrgService{
		createOrganizationFunc: func(org *orgs.Organization) error {
			org.ID = 123
			return nil
		},
		addMemberFunc: func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
			return errors.New("failed to add member")
		},
	}
	handlers := NewOrgHandlers(service)

	orgReq := orgs.CreateOrgRequest{Name: "test-org"}
	body, err := json.Marshal(orgReq)
	require.NoError(t, err)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs", body, authCtx)
	w := httptest.NewRecorder()

	handlers.CreateOrganization(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListOrganizations_Success tests successful organization listing
func TestListOrganizations_Success(t *testing.T) {
	service := &mockOrgService{
		listOrganizationsFunc: func(userID int64) ([]*orgs.Organization, error) {
			assert.Equal(t, int64(1), userID)
			return []*orgs.Organization{
				{ID: 1, Name: "org1"},
				{ID: 2, Name: "org2"},
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("GET", "/orgs", nil, authCtx)
	w := httptest.NewRecorder()

	handlers.ListOrganizations(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "org1")
	assert.Contains(t, w.Body.String(), "org2")
}

// TestListOrganizations_Unauthorized tests listing without auth
func TestListOrganizations_Unauthorized(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs", nil)
	w := httptest.NewRecorder()

	handlers.ListOrganizations(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestListOrganizations_Error tests listing error
func TestListOrganizations_Error(t *testing.T) {
	service := &mockOrgService{
		listOrganizationsFunc: func(userID int64) ([]*orgs.Organization, error) {
			return nil, errors.New("database error")
		},
	}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("GET", "/orgs", nil, authCtx)
	w := httptest.NewRecorder()

	handlers.ListOrganizations(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetOrganization_Success tests successful organization retrieval
func TestGetOrganization_Success(t *testing.T) {
	service := &mockOrgService{
		getOrganizationFunc: func(id int64) (*orgs.Organization, error) {
			return &orgs.Organization{
				ID:   id,
				Name: "test-org",
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetOrganization(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-org")
}

// TestGetOrganization_NotFound tests organization not found
func TestGetOrganization_NotFound(t *testing.T) {
	service := &mockOrgService{
		getOrganizationFunc: func(id int64) (*orgs.Organization, error) {
			return nil, errors.New("not found")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.GetOrganization(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetOrganization_InvalidID tests invalid organization ID
func TestGetOrganization_InvalidID(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.GetOrganization(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateOrganization_Success tests successful organization update
func TestUpdateOrganization_Success(t *testing.T) {
	service := &mockOrgService{
		updateOrganizationFunc: func(id int64, updates *orgs.UpdateOrgRequest) error {
			assert.Equal(t, int64(123), id)
			return nil
		},
		getOrganizationFunc: func(id int64) (*orgs.Organization, error) {
			return &orgs.Organization{
				ID:          id,
				Name:        "test-org",
				DisplayName: "Updated Org",
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	updateReq := orgs.UpdateOrgRequest{
		DisplayName: strPtr("Updated Org"),
	}
	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/orgs/123", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.UpdateOrganization(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestUpdateOrganization_InvalidJSON tests update with invalid JSON
func TestUpdateOrganization_InvalidJSON(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("PUT", "/orgs/123", bytes.NewReader([]byte("{")))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.UpdateOrganization(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateOrganization_Error tests update error
func TestUpdateOrganization_Error(t *testing.T) {
	service := &mockOrgService{
		updateOrganizationFunc: func(id int64, updates *orgs.UpdateOrgRequest) error {
			return errors.New("database error")
		},
	}
	handlers := NewOrgHandlers(service)

	updateReq := orgs.UpdateOrgRequest{}
	body, err := json.Marshal(updateReq)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/orgs/123", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.UpdateOrganization(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestDeleteOrganization_Success tests successful organization deletion
func TestDeleteOrganization_Success(t *testing.T) {
	service := &mockOrgService{
		deleteOrganizationFunc: func(id int64) error {
			assert.Equal(t, int64(123), id)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.DeleteOrganization(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestDeleteOrganization_Error tests deletion error
func TestDeleteOrganization_Error(t *testing.T) {
	service := &mockOrgService{
		deleteOrganizationFunc: func(id int64) error {
			return errors.New("cannot delete")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/123", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.DeleteOrganization(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestGetQuotas_Success tests successful quota retrieval
func TestGetQuotas_Success(t *testing.T) {
	service := &mockOrgService{
		getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
			return &orgs.OrgQuotas{
				MaxModules: 100,
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123/quotas", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetQuotas(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetQuotas_NotFound tests quota not found
func TestGetQuotas_NotFound(t *testing.T) {
	service := &mockOrgService{
		getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
			return nil, errors.New("not found")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/999/quotas", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "999"})
	w := httptest.NewRecorder()

	handlers.GetQuotas(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestUpdateQuotas_Success tests successful quota update
func TestUpdateQuotas_Success(t *testing.T) {
	service := &mockOrgService{
		updateQuotasFunc: func(orgID int64, quotas *orgs.OrgQuotas) error {
			return nil
		},
		getQuotasFunc: func(orgID int64) (*orgs.OrgQuotas, error) {
			return &orgs.OrgQuotas{
				MaxModules: 200,
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	quotas := orgs.OrgQuotas{MaxModules: 200}
	body, err := json.Marshal(quotas)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/orgs/123/quotas", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.UpdateQuotas(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetUsage_Success tests successful usage retrieval
func TestGetUsage_Success(t *testing.T) {
	service := &mockOrgService{
		getUsageFunc: func(orgID int64) (*orgs.OrgUsage, error) {
			return &orgs.OrgUsage{}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123/usage", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetUsage(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetUsageHistory_Success tests successful usage history retrieval
func TestGetUsageHistory_Success(t *testing.T) {
	service := &mockOrgService{
		getUsageHistoryFunc: func(orgID int64, limit int) ([]*orgs.OrgUsage, error) {
			assert.Equal(t, 12, limit)
			return []*orgs.OrgUsage{
				{},
				{},
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123/usage/history", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetUsageHistory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetUsageHistory_WithLimit tests usage history with custom limit
func TestGetUsageHistory_WithLimit(t *testing.T) {
	service := &mockOrgService{
		getUsageHistoryFunc: func(orgID int64, limit int) ([]*orgs.OrgUsage, error) {
			assert.Equal(t, 24, limit)
			return []*orgs.OrgUsage{}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123/usage/history?limit=24", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetUsageHistory(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetUsageHistory_InvalidLimit tests invalid limit parameter
func TestGetUsageHistory_InvalidLimit(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/123/usage/history?limit=invalid", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	w := httptest.NewRecorder()

	handlers.GetUsageHistory(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

// TestAddMember_Success tests successfully adding a member
func TestAddMember_Success(t *testing.T) {
	service := &mockOrgService{
		addMemberFunc: func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
			assert.Equal(t, int64(1), orgID)
			assert.Equal(t, int64(2), userID)
			assert.Equal(t, auth.RoleDeveloper, role)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": 2,
		"role":    "developer",
	})
	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/1/members", reqBody, authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// TestAddMember_Unauthorized tests adding member without auth
func TestAddMember_Unauthorized(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": 2,
		"role":    "developer",
	})
	req := httptest.NewRequest("POST", "/orgs/1/members", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAddMember_InvalidID tests with invalid organization ID
func TestAddMember_InvalidID(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": 2,
		"role":    "developer",
	})
	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/invalid/members", reqBody, authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestAddMember_ServiceError tests service error
func TestAddMember_ServiceError(t *testing.T) {
	service := &mockOrgService{
		addMemberFunc: func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
			return errors.New("service error")
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(map[string]interface{}{
		"user_id": 2,
		"role":    "developer",
	})
	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/1/members", reqBody, authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.AddMember(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestCreateInvitation_Success tests successfully creating an invitation
func TestCreateInvitation_Success(t *testing.T) {
	service := &mockOrgService{
		createInvitationFunc: func(inv *orgs.OrgInvitation) error {
			assert.Equal(t, int64(1), inv.OrgID)
			assert.Equal(t, "newuser@example.com", inv.Email)
			assert.Equal(t, auth.RoleDeveloper, inv.Role)
			inv.ID = 1
			inv.Token = "test-token-123"
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(orgs.InviteMemberRequest{
		Email: "newuser@example.com",
		Role:  auth.RoleDeveloper,
	})
	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/1/invitations", reqBody, authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateInvitation(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response orgs.OrgInvitation
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "test-token-123", response.Token)
}

// TestCreateInvitation_Unauthorized tests creating invitation without auth
func TestCreateInvitation_Unauthorized(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(orgs.InviteMemberRequest{
		Email: "newuser@example.com",
		Role:  auth.RoleDeveloper,
	})
	req := httptest.NewRequest("POST", "/orgs/1/invitations", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateInvitation(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestCreateInvitation_InvalidJSON tests with invalid JSON
func TestCreateInvitation_InvalidJSON(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/1/invitations", []byte("invalid json"), authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateInvitation(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateInvitation_ServiceError tests service error
func TestCreateInvitation_ServiceError(t *testing.T) {
	service := &mockOrgService{
		createInvitationFunc: func(inv *orgs.OrgInvitation) error {
			return errors.New("service error")
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(orgs.InviteMemberRequest{
		Email: "newuser@example.com",
		Role:  auth.RoleDeveloper,
	})
	authCtx := createAuthContext(1, "testuser")
	req := createAuthRequest("POST", "/orgs/1/invitations", reqBody, authCtx)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.CreateInvitation(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestAcceptInvitation_Success tests successfully accepting an invitation
func TestAcceptInvitation_Success(t *testing.T) {
	service := &mockOrgService{
		acceptInvitationFunc: func(token string, userID int64) (*orgs.Organization, error) {
			assert.Equal(t, "test-token-123", token)
			assert.Equal(t, int64(2), userID)
			return &orgs.Organization{ID: 1, Name: "test-org"}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(2, "newuser")
	req := createAuthRequest("POST", "/invitations/test-token-123/accept", nil, authCtx)
	req = mux.SetURLVars(req, map[string]string{"token": "test-token-123"})
	w := httptest.NewRecorder()

	handlers.AcceptInvitation(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestAcceptInvitation_Unauthorized tests accepting without auth
func TestAcceptInvitation_Unauthorized(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("POST", "/invitations/test-token-123/accept", nil)
	req = mux.SetURLVars(req, map[string]string{"token": "test-token-123"})
	w := httptest.NewRecorder()

	handlers.AcceptInvitation(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// TestAcceptInvitation_InvalidToken tests with invalid token
func TestAcceptInvitation_InvalidToken(t *testing.T) {
	service := &mockOrgService{
		acceptInvitationFunc: func(token string, userID int64) (*orgs.Organization, error) {
			return nil, errors.New("invalid or expired token")
		},
	}
	handlers := NewOrgHandlers(service)

	authCtx := createAuthContext(2, "newuser")
	req := createAuthRequest("POST", "/invitations/invalid/accept", nil, authCtx)
	req = mux.SetURLVars(req, map[string]string{"token": "invalid"})
	w := httptest.NewRecorder()

	handlers.AcceptInvitation(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Benchmark tests
func BenchmarkCreateOrganization(b *testing.B) {
	service := &mockOrgService{
		createOrganizationFunc: func(org *orgs.Organization) error {
			org.ID = 123
			return nil
		},
		addMemberFunc: func(orgID int64, userID int64, role auth.Role, invitedBy *int64) error {
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	orgReq := orgs.CreateOrgRequest{Name: "bench-org"}
	body, _ := json.Marshal(orgReq)
	authCtx := createAuthContext(1, "testuser")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := createAuthRequest("POST", "/orgs", body, authCtx)
		w := httptest.NewRecorder()
		handlers.CreateOrganization(w, req)
	}
}

// TestListMembers_Success tests successful member listing
func TestListMembers_Success(t *testing.T) {
	service := &mockOrgService{
		listMembersFunc: func(orgID int64) ([]*orgs.OrgMember, error) {
			return []*orgs.OrgMember{
				{UserID: 1, Role: auth.RoleAdmin},
				{UserID: 2, Role: auth.RoleDeveloper},
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/1/members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListMembers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var members []*orgs.OrgMember
	err := json.Unmarshal(w.Body.Bytes(), &members)
	require.NoError(t, err)
	assert.Len(t, members, 2)
}

// TestListMembers_Error tests error from service
func TestListMembers_Error(t *testing.T) {
	service := &mockOrgService{
		listMembersFunc: func(orgID int64) ([]*orgs.OrgMember, error) {
			return nil, errors.New("database error")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/1/members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListMembers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListMembers_InvalidID tests invalid organization ID
func TestListMembers_InvalidID(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/invalid/members", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.ListMembers(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestUpdateMember_Success tests successful member update
func TestUpdateMember_Success(t *testing.T) {
	service := &mockOrgService{
		updateMemberFunc: func(orgID int64, userID int64, role auth.Role) error {
			assert.Equal(t, int64(1), orgID)
			assert.Equal(t, int64(2), userID)
			assert.Equal(t, auth.RoleAdmin, role)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(orgs.UpdateMemberRequest{
		Role: auth.RoleAdmin,
	})
	req := httptest.NewRequest("PATCH", "/orgs/1/members/2", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1", "user_id": "2"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestUpdateMember_Error tests error from service
func TestUpdateMember_Error(t *testing.T) {
	service := &mockOrgService{
		updateMemberFunc: func(orgID int64, userID int64, role auth.Role) error {
			return errors.New("update failed")
		},
	}
	handlers := NewOrgHandlers(service)

	reqBody, _ := json.Marshal(orgs.UpdateMemberRequest{
		Role: auth.RoleAdmin,
	})
	req := httptest.NewRequest("PATCH", "/orgs/1/members/2", bytes.NewBuffer(reqBody))
	req = mux.SetURLVars(req, map[string]string{"id": "1", "user_id": "2"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestUpdateMember_InvalidJSON tests invalid JSON
func TestUpdateMember_InvalidJSON(t *testing.T) {
	service := &mockOrgService{}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("PATCH", "/orgs/1/members/2", bytes.NewBufferString("invalid"))
	req = mux.SetURLVars(req, map[string]string{"id": "1", "user_id": "2"})
	w := httptest.NewRecorder()

	handlers.UpdateMember(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestRemoveMember_Success tests successful member removal
func TestRemoveMember_Success(t *testing.T) {
	service := &mockOrgService{
		removeMemberFunc: func(orgID int64, userID int64) error {
			assert.Equal(t, int64(1), orgID)
			assert.Equal(t, int64(2), userID)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/1/members/2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "user_id": "2"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestRemoveMember_Error tests error from service
func TestRemoveMember_Error(t *testing.T) {
	service := &mockOrgService{
		removeMemberFunc: func(orgID int64, userID int64) error {
			return errors.New("removal failed")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/1/members/2", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1", "user_id": "2"})
	w := httptest.NewRecorder()

	handlers.RemoveMember(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestListInvitations_Success tests successful invitation listing
func TestListInvitations_Success(t *testing.T) {
	service := &mockOrgService{
		listInvitationsFunc: func(orgID int64) ([]*orgs.OrgInvitation, error) {
			return []*orgs.OrgInvitation{
				{Email: "user1@example.com", Role: auth.RoleDeveloper},
				{Email: "user2@example.com", Role: auth.RoleAdmin},
			}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/1/invitations", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListInvitations(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var invitations []*orgs.OrgInvitation
	err := json.Unmarshal(w.Body.Bytes(), &invitations)
	require.NoError(t, err)
	assert.Len(t, invitations, 2)
}

// TestListInvitations_Error tests error from service
func TestListInvitations_Error(t *testing.T) {
	service := &mockOrgService{
		listInvitationsFunc: func(orgID int64) ([]*orgs.OrgInvitation, error) {
			return nil, errors.New("database error")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("GET", "/orgs/1/invitations", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()

	handlers.ListInvitations(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestRevokeInvitation_Success tests successful invitation revocation
func TestRevokeInvitation_Success(t *testing.T) {
	service := &mockOrgService{
		revokeInvitationFunc: func(invitationID int64) error {
			assert.Equal(t, int64(123), invitationID)
			return nil
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/invitations/123", nil)
	req = mux.SetURLVars(req, map[string]string{"invitation_id": "123"})
	w := httptest.NewRecorder()

	handlers.RevokeInvitation(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// TestRevokeInvitation_Error tests error from service
func TestRevokeInvitation_Error(t *testing.T) {
	service := &mockOrgService{
		revokeInvitationFunc: func(invitationID int64) error {
			return errors.New("revocation failed")
		},
	}
	handlers := NewOrgHandlers(service)

	req := httptest.NewRequest("DELETE", "/orgs/invitations/123", nil)
	req = mux.SetURLVars(req, map[string]string{"invitation_id": "123"})
	w := httptest.NewRecorder()

	handlers.RevokeInvitation(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func BenchmarkGetOrganization(b *testing.B) {
	service := &mockOrgService{
		getOrganizationFunc: func(id int64) (*orgs.Organization, error) {
			return &orgs.Organization{ID: id}, nil
		},
	}
	handlers := NewOrgHandlers(service)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/orgs/123", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "123"})
		w := httptest.NewRecorder()
		handlers.GetOrganization(w, req)
	}
}
