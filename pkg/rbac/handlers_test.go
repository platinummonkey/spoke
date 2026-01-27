package rbac

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/audit"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockAuditLogger is a mock audit logger for testing
type mockAuditLogger struct {
	logs []*audit.AuditEvent
}

func (m *mockAuditLogger) Log(ctx context.Context, event *audit.AuditEvent) error {
	m.logs = append(m.logs, event)
	return nil
}

func (m *mockAuditLogger) LogAuthentication(ctx context.Context, eventType audit.EventType, userID *int64, username string, status audit.EventStatus, message string) error {
	return nil
}

func (m *mockAuditLogger) LogAuthorization(ctx context.Context, eventType audit.EventType, userID *int64, resourceType audit.ResourceType, resourceID string, status audit.EventStatus, message string) error {
	return nil
}

func (m *mockAuditLogger) LogDataMutation(ctx context.Context, eventType audit.EventType, userID *int64, resourceType audit.ResourceType, resourceID string, changes *audit.ChangeDetails, message string) error {
	return nil
}

func (m *mockAuditLogger) LogConfiguration(ctx context.Context, eventType audit.EventType, userID *int64, resourceID string, changes *audit.ChangeDetails, message string) error {
	return nil
}

func (m *mockAuditLogger) LogAdminAction(ctx context.Context, eventType audit.EventType, adminUserID *int64, targetUserID *int64, message string) error {
	return nil
}

func (m *mockAuditLogger) LogAccess(ctx context.Context, eventType audit.EventType, userID *int64, resourceType audit.ResourceType, resourceID string, message string) error {
	return nil
}

func (m *mockAuditLogger) LogHTTPRequest(ctx context.Context, r *http.Request, statusCode int, duration time.Duration, err error) error {
	return nil
}

func (m *mockAuditLogger) Close() error {
	return nil
}

// TestNewHandlers verifies handler initialization
func TestNewHandlers(t *testing.T) {
	db := &sql.DB{} // Mock DB
	auditLogger := &mockAuditLogger{}

	handlers := NewHandlers(db, auditLogger)

	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.store)
	assert.NotNil(t, handlers.checker)
	assert.NotNil(t, handlers.auditLogger)
}

// TestRegisterRoutes verifies all RBAC routes are registered
func TestRegisterRoutes(t *testing.T) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)
	router := mux.NewRouter()

	handlers.RegisterRoutes(router)

	tests := []struct {
		method string
		path   string
	}{
		// Role management
		{"POST", "/rbac/roles"},
		{"GET", "/rbac/roles"},
		{"GET", "/rbac/roles/123"},
		{"PUT", "/rbac/roles/123"},
		{"DELETE", "/rbac/roles/123"},
		// User role assignments
		{"POST", "/rbac/users/123/roles"},
		{"GET", "/rbac/users/123/roles"},
		{"DELETE", "/rbac/users/123/roles/456"},
		{"GET", "/rbac/users/123/permissions"},
		// Permission checking
		{"POST", "/rbac/check"},
		// Team management
		{"POST", "/rbac/teams"},
		{"GET", "/rbac/teams"},
		{"GET", "/rbac/teams/123"},
		{"PUT", "/rbac/teams/123"},
		{"DELETE", "/rbac/teams/123"},
		// Team members
		{"POST", "/rbac/teams/123/members"},
		{"GET", "/rbac/teams/123/members"},
		{"DELETE", "/rbac/teams/123/members/456"},
		// Team roles
		{"POST", "/rbac/teams/123/roles"},
		{"DELETE", "/rbac/teams/123/roles/456"},
		// Templates
		{"GET", "/rbac/templates"},
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

// TestCreateRole_Validation tests role creation validation
func TestCreateRole_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		skipWithMockDB bool
	}{
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"display_name": "Test Role",
				"description":  "A test role",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name and display_name are required",
		},
		{
			name: "missing display_name",
			requestBody: map[string]interface{}{
				"name":        "test-role",
				"description": "A test role",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name and display_name are required",
		},
		{
			name: "valid role",
			requestBody: map[string]interface{}{
				"name":         "test-role",
				"display_name": "Test Role",
				"description":  "A test role",
				"permissions":  []Permission{},
			},
			expectedStatus: http.StatusInternalServerError, // Would succeed with real DB
			skipWithMockDB: true,                           // Skip - requires real database
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipWithMockDB {
				t.Skip("Requires real database")
			}

			db := &sql.DB{}
			auditLogger := &mockAuditLogger{}
			handlers := NewHandlers(db, auditLogger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/rbac/roles", bytes.NewReader(body))
			// Add auth context
			authCtx := &auth.AuthContext{User: &auth.User{ID: 1}, Organization: &auth.Organization{ID: 1}}
			ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handlers.CreateRole(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

// TestCreateRole_InvalidJSON tests invalid JSON handling
func TestCreateRole_InvalidJSON(t *testing.T) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	req := httptest.NewRequest("POST", "/rbac/roles", bytes.NewReader([]byte("{invalid json")))
	authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

// TestAssignRoleToUser_Validation tests user role assignment validation
func TestAssignRoleToUser_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "missing role_id",
			userID: "123",
			requestBody: map[string]interface{}{
				"scope": "global",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "role_id is required",
		},
		{
			name:   "invalid user_id",
			userID: "invalid",
			requestBody: map[string]interface{}{
				"role_id": 456,
				"scope":   "global",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid user ID",
		},
		{
			name:   "zero role_id",
			userID: "123",
			requestBody: map[string]interface{}{
				"role_id": 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "role_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sql.DB{}
			auditLogger := &mockAuditLogger{}
			handlers := NewHandlers(db, auditLogger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/rbac/users/"+tt.userID+"/roles", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"id": tt.userID})
			authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
			ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handlers.AssignRoleToUser(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestCheckPermission_Validation tests permission check validation
func TestCheckPermission_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing user_id",
			requestBody: map[string]interface{}{
				"action":        "read",
				"resource_type": "module",
				"resource_id":   "test-module",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name: "missing action",
			requestBody: map[string]interface{}{
				"user_id":       123,
				"resource_type": "module",
				"resource_id":   "test-module",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "action is required",
		},
		{
			name: "missing resource_type",
			requestBody: map[string]interface{}{
				"user_id":     123,
				"action":      "read",
				"resource_id": "test-module",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "resource_type is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sql.DB{}
			auditLogger := &mockAuditLogger{}
			handlers := NewHandlers(db, auditLogger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/rbac/check", bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers.CheckPermission(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestCreateTeam_Validation tests team creation validation
func TestCreateTeam_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing name",
			requestBody: map[string]interface{}{
				"display_name": "Test Team",
				"description":  "A test team",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Name is required",
		},
		{
			name: "missing organization_id",
			requestBody: map[string]interface{}{
				"name":         "test-team",
				"display_name": "Test Team",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Organization ID is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sql.DB{}
			auditLogger := &mockAuditLogger{}
			handlers := NewHandlers(db, auditLogger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/rbac/teams", bytes.NewReader(body))
			authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
			ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handlers.CreateTeam(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestAddTeamMember_Validation tests team member addition validation
func TestAddTeamMember_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	tests := []struct {
		name           string
		teamID         string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:   "missing user_id",
			teamID: "123",
			requestBody: map[string]interface{}{
				"role": "member",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
		{
			name:   "invalid team_id",
			teamID: "invalid",
			requestBody: map[string]interface{}{
				"user_id": 456,
				"role":    "member",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid team ID",
		},
		{
			name:   "zero user_id",
			teamID: "123",
			requestBody: map[string]interface{}{
				"user_id": 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "user_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &sql.DB{}
			auditLogger := &mockAuditLogger{}
			handlers := NewHandlers(db, auditLogger)

			body, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/rbac/teams/"+tt.teamID+"/members", bytes.NewReader(body))
			req = mux.SetURLVars(req, map[string]string{"id": tt.teamID})
			authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
			ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			handlers.AddTeamMember(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// TestUpdateRole_Validation tests role update validation
func TestUpdateRole_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	// Test invalid JSON
	req := httptest.NewRequest("PUT", "/rbac/roles/123", bytes.NewReader([]byte("{bad json")))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

// TestUpdateTeam_Validation tests team update validation
func TestUpdateTeam_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database - validation logic needs database mock")

	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	// Test invalid JSON
	req := httptest.NewRequest("PUT", "/rbac/teams/123", bytes.NewReader([]byte("not json")))
	req = mux.SetURLVars(req, map[string]string{"id": "123"})
	authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.UpdateTeam(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

// TestGetRoleTemplates tests getting role templates
func TestGetRoleTemplates(t *testing.T) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	req := httptest.NewRequest("GET", "/rbac/templates", nil)
	w := httptest.NewRecorder()

	handlers.GetRoleTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var templates []RoleTemplate
	err := json.NewDecoder(w.Body).Decode(&templates)
	require.NoError(t, err)

	// Verify we get role templates back
	assert.NotEmpty(t, templates, "Should return role templates")

	// Verify templates have required fields
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.Name, "Template should have a name")
		assert.NotEmpty(t, tmpl.DisplayName, "Template should have a display name")
	}
}

// TestAuditLogging tests that audit logs are created
func TestAuditLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")

	db := &sql.DB{}
	mockLogger := &mockAuditLogger{logs: []*audit.AuditEvent{}}
	handlers := NewHandlers(db, mockLogger)

	// Attempt to create a role (will fail without real DB, but should still audit)
	body, _ := json.Marshal(map[string]interface{}{
		"name":         "test-role",
		"display_name": "Test Role",
		"permissions":  []Permission{},
	})

	req := httptest.NewRequest("POST", "/rbac/roles", bytes.NewReader(body))
	authCtx := &auth.AuthContext{User: &auth.User{ID: 123}, Organization: &auth.Organization{ID: 456}}
	ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	handlers.CreateRole(w, req)

	// Note: Audit logging happens even on errors
	// In this test, it will fail due to no real DB, but audit should still be called
	// The actual audit logging is tested in the audit package tests
}

// TestPermissionCheckResponse tests permission check response format
func TestPermissionCheckResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test that permission check returns proper response format
}

// TestRoleInheritance tests role inheritance
func TestRoleInheritance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test that child roles inherit parent permissions
}

// TestTeamRolePropagation tests that team roles propagate to members
func TestTeamRolePropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test that assigning a role to a team affects all members
}

// TestScopeHierarchy tests permission scope hierarchy
func TestScopeHierarchy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}
	t.Skip("Requires PostgreSQL test database")
	// Would test global > organization > resource scope hierarchy
}

// TestGetUser_InvalidID tests invalid user ID handling
func TestGetUser_InvalidID(t *testing.T) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	req := httptest.NewRequest("GET", "/rbac/users/invalid/roles", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "invalid"})
	w := httptest.NewRecorder()

	handlers.GetUserRoles(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid user ID")
}

// TestRemoveTeamMember_InvalidIDs tests invalid ID handling
func TestRemoveTeamMember_InvalidIDs(t *testing.T) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	tests := []struct {
		name           string
		teamID         string
		userID         string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid team ID",
			teamID:         "invalid",
			userID:         "123",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid team ID",
		},
		{
			name:           "invalid user ID",
			teamID:         "123",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/rbac/teams/"+tt.teamID+"/members/"+tt.userID, nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.teamID, "user_id": tt.userID})
			authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
			ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
			req = req.WithContext(ctx)
			w := httptest.NewRecorder()

			handlers.RemoveTeamMember(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedError)
		})
	}
}

// Benchmark role creation
func BenchmarkCreateRole(b *testing.B) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	body, _ := json.Marshal(map[string]interface{}{
		"name":         "test-role",
		"display_name": "Test Role",
		"permissions":  []Permission{},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/rbac/roles", bytes.NewReader(body))
		authCtx := &auth.AuthContext{User: &auth.User{ID: 1}}
		ctx := context.WithValue(req.Context(), middleware.AuthContextKey, authCtx)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handlers.CreateRole(w, req)
	}
}

// Benchmark permission check
func BenchmarkCheckPermission(b *testing.B) {
	db := &sql.DB{}
	auditLogger := &mockAuditLogger{}
	handlers := NewHandlers(db, auditLogger)

	body, _ := json.Marshal(map[string]interface{}{
		"user_id":       123,
		"action":        "read",
		"resource_type": "module",
		"resource_id":   "test-module",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/rbac/check", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.CheckPermission(w, req)
	}
}
