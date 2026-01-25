package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/orgs"
)

type mockOrgService struct {
	orgs            map[int64]*orgs.Organization
	slugToID        map[string]int64
	quotaExceeded   bool
	checkModuleCalls int
}

func (m *mockOrgService) GetOrganization(id int64) (*orgs.Organization, error) {
	org, ok := m.orgs[id]
	if !ok {
		return nil, nil
	}
	return org, nil
}

func (m *mockOrgService) GetOrganizationBySlug(slug string) (*orgs.Organization, error) {
	id, ok := m.slugToID[slug]
	if !ok {
		return nil, nil
	}
	return m.GetOrganization(id)
}

func (m *mockOrgService) CheckModuleQuota(orgID int64) error {
	m.checkModuleCalls++
	if m.quotaExceeded {
		return &orgs.QuotaExceededError{
			Resource: "modules",
			Current:  10,
			Limit:    5,
		}
	}
	return nil
}

func (m *mockOrgService) CheckVersionQuota(orgID int64, moduleName string) error {
	return nil
}

func (m *mockOrgService) CheckStorageQuota(orgID int64, additionalBytes int64) error {
	return nil
}

func (m *mockOrgService) CheckCompileJobQuota(orgID int64) error {
	return nil
}

func (m *mockOrgService) CheckAPIRateLimit(orgID int64) error {
	return nil
}

func (m *mockOrgService) IncrementAPIRequests(orgID int64) error {
	return nil
}

// Implement other required methods as stubs
func (m *mockOrgService) CreateOrganization(org *orgs.Organization) error { return nil }
func (m *mockOrgService) ListOrganizations(userID int64) ([]*orgs.Organization, error) { return nil, nil }
func (m *mockOrgService) UpdateOrganization(id int64, updates *orgs.UpdateOrgRequest) error { return nil }
func (m *mockOrgService) DeleteOrganization(id int64) error { return nil }
func (m *mockOrgService) GetQuotas(orgID int64) (*orgs.OrgQuotas, error) { return nil, nil }
func (m *mockOrgService) UpdateQuotas(orgID int64, quotas *orgs.OrgQuotas) error { return nil }
func (m *mockOrgService) GetDefaultQuotas(quotaTier orgs.QuotaTier) *orgs.OrgQuotas { return nil }
func (m *mockOrgService) GetUsage(orgID int64) (*orgs.OrgUsage, error) { return nil, nil }
func (m *mockOrgService) GetUsageHistory(orgID int64, limit int) ([]*orgs.OrgUsage, error) { return nil, nil }
func (m *mockOrgService) ResetUsagePeriod(orgID int64) error { return nil }
func (m *mockOrgService) ListMembers(orgID int64) ([]*orgs.OrgMember, error) { return nil, nil }
func (m *mockOrgService) GetMember(orgID, userID int64) (*orgs.OrgMember, error) { return nil, nil }
func (m *mockOrgService) AddMember(orgID, userID int64, role auth.Role, invitedBy *int64) error {
	return nil
}
func (m *mockOrgService) UpdateMemberRole(orgID, userID int64, role auth.Role) error {
	return nil
}
func (m *mockOrgService) RemoveMember(orgID, userID int64) error { return nil }
func (m *mockOrgService) CreateInvitation(invitation *orgs.OrgInvitation) error { return nil }
func (m *mockOrgService) GetInvitation(token string) (*orgs.OrgInvitation, error) { return nil, nil }
func (m *mockOrgService) ListInvitations(orgID int64) ([]*orgs.OrgInvitation, error) { return nil, nil }
func (m *mockOrgService) AcceptInvitation(token string, userID int64) error { return nil }
func (m *mockOrgService) RevokeInvitation(id int64) error { return nil }
func (m *mockOrgService) CleanupExpiredInvitations() error { return nil }
func (m *mockOrgService) IncrementModules(orgID int64) error { return nil }
func (m *mockOrgService) IncrementVersions(orgID int64) error { return nil }
func (m *mockOrgService) IncrementStorage(orgID int64, bytes int64) error { return nil }
func (m *mockOrgService) IncrementCompileJobs(orgID int64) error { return nil }
func (m *mockOrgService) DecrementModules(orgID int64) error { return nil }
func (m *mockOrgService) DecrementVersions(orgID int64) error { return nil }
func (m *mockOrgService) DecrementStorage(orgID int64, bytes int64) error { return nil }

func TestOrgContextMiddleware(t *testing.T) {
	mockSvc := &mockOrgService{
		orgs: map[int64]*orgs.Organization{
			1: {ID: 1, Name: "test-org", Slug: "test-org"},
		},
		slugToID: map[string]int64{
			"test-org": 1,
		},
	}

	t.Run("adds organization to context by ID", func(t *testing.T) {
		middleware := OrgContextMiddleware(mockSvc)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			org, ok := r.Context().Value(OrgKey).(*orgs.Organization)
			if !ok || org == nil {
				t.Fatal("organization not found in context")
			}
			if org.ID != 1 {
				t.Errorf("expected org ID 1, got %d", org.ID)
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/orgs/1", nil)
		req = mux.SetURLVars(req, map[string]string{"org_id": "1"})
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("adds organization to context by slug", func(t *testing.T) {
		middleware := OrgContextMiddleware(mockSvc)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			org, ok := r.Context().Value(OrgKey).(*orgs.Organization)
			if !ok || org == nil {
				t.Fatal("organization not found in context")
			}
			if org.Slug != "test-org" {
				t.Errorf("expected org slug test-org, got %s", org.Slug)
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/orgs/test-org", nil)
		req = mux.SetURLVars(req, map[string]string{"org_slug": "test-org"})
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestQuotaCheckMiddleware(t *testing.T) {
	t.Run("allows request when quota not exceeded", func(t *testing.T) {
		mockSvc := &mockOrgService{
			quotaExceeded: false,
		}

		middleware := QuotaCheckMiddleware(mockSvc, "module")
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		org := &orgs.Organization{ID: 1}
		ctx := context.WithValue(context.Background(), OrgKey, org)
		req := httptest.NewRequest("POST", "/modules", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if mockSvc.checkModuleCalls != 1 {
			t.Errorf("expected 1 quota check call, got %d", mockSvc.checkModuleCalls)
		}
	})

	t.Run("blocks request when quota exceeded", func(t *testing.T) {
		mockSvc := &mockOrgService{
			quotaExceeded: true,
		}

		middleware := QuotaCheckMiddleware(mockSvc, "module")
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		org := &orgs.Organization{ID: 1}
		ctx := context.WithValue(context.Background(), OrgKey, org)
		req := httptest.NewRequest("POST", "/modules", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("expected status 429, got %d", w.Code)
		}
	})

	t.Run("skips quota check for GET requests", func(t *testing.T) {
		mockSvc := &mockOrgService{
			quotaExceeded: true,
		}

		middleware := QuotaCheckMiddleware(mockSvc, "module")
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		org := &orgs.Organization{ID: 1}
		ctx := context.WithValue(context.Background(), OrgKey, org)
		req := httptest.NewRequest("GET", "/modules", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if mockSvc.checkModuleCalls != 0 {
			t.Errorf("expected 0 quota check calls for GET, got %d", mockSvc.checkModuleCalls)
		}
	})
}
