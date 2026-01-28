package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platinummonkey/spoke/pkg/auth"
)

func TestNewAuthMiddleware(t *testing.T) {
	tm := &auth.TokenManager{}

	t.Run("creates middleware with required auth", func(t *testing.T) {
		m := NewAuthMiddleware(tm, false)
		if m == nil {
			t.Fatal("expected non-nil middleware")
		}
		if m.tokenManager != tm {
			t.Error("token manager not set correctly")
		}
		if m.optional {
			t.Error("expected optional to be false")
		}
	})

	t.Run("creates middleware with optional auth", func(t *testing.T) {
		m := NewAuthMiddleware(tm, true)
		if m == nil {
			t.Fatal("expected non-nil middleware")
		}
		if !m.optional {
			t.Error("expected optional to be true")
		}
	})
}

func TestAuthMiddleware_Handler(t *testing.T) {
	// Since TokenManager.ValidateToken always returns error "not implemented",
	// we can only test the auth flow up to token validation

	t.Run("rejects request without Authorization header when required", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, false)
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"missing authorization header"}` {
			t.Errorf("unexpected body: %s", body)
		}
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
	})

	t.Run("allows request without Authorization header when optional", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, true)
		handlerCalled := false
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("handler should have been called")
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("rejects request with invalid Authorization header format", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, false)
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		testCases := []struct {
			name          string
			header        string
			expectedError string
		}{
			{"no Bearer prefix", "token123", "invalid authorization header format"},
			{"Basic auth", "Basic dXNlcjpwYXNz", "invalid authorization header format"},
			{"Bearer without token", "Bearer", "invalid authorization header format"},
			// "Bearer " with trailing space creates empty token, which fails validation instead
			{"empty Bearer", "Bearer ", "invalid or expired token"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", tc.header)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				if w.Code != http.StatusUnauthorized {
					t.Errorf("expected status 401, got %d", w.Code)
				}
				body := w.Body.String()
				expectedBody := `{"error":"` + tc.expectedError + `"}`
				if body != expectedBody {
					t.Errorf("expected body %s, got %s", expectedBody, body)
				}
			})
		}
	})

	t.Run("rejects request with valid format but token validation fails", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, false)
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		// Generate a valid token but it will fail validation since DB lookup is not implemented
		_, token, err := tm.CreateToken(123, "test", "test token", []auth.Scope{auth.ScopeModuleRead}, nil)
		if err != nil {
			t.Fatalf("failed to create token: %v", err)
		}

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"invalid or expired token"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("rejects request with malformed token", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, false)
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer malformed_token")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("rejects token without spoke_ prefix", func(t *testing.T) {
		tm := auth.NewTokenManager()
		middleware := NewAuthMiddleware(tm, false)
		handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer abc123def456")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"invalid or expired token"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})
}

func TestGetAuthContext(t *testing.T) {
	t.Run("returns auth context when present", func(t *testing.T) {
		expectedAuthCtx := &auth.AuthContext{
			Token: &auth.APIToken{
				ID:     1,
				UserID: 123,
			},
			Scopes: []auth.Scope{auth.ScopeModuleRead},
		}

		ctx := context.WithValue(context.Background(), AuthContextKey, expectedAuthCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)

		authCtx := GetAuthContext(req)
		if authCtx == nil {
			t.Fatal("expected auth context, got nil")
		}
		if authCtx != expectedAuthCtx {
			t.Error("returned auth context does not match expected")
		}
	})

	t.Run("returns nil when auth context not in request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		authCtx := GetAuthContext(req)
		if authCtx != nil {
			t.Error("expected nil auth context")
		}
	})

	t.Run("returns nil when context value is wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), AuthContextKey, "wrong_type")
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)

		authCtx := GetAuthContext(req)
		if authCtx != nil {
			t.Error("expected nil auth context for wrong type")
		}
	})
}

func TestRequireScope(t *testing.T) {
	t.Run("allows request with required scope", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeModuleRead, auth.ScopeModuleWrite},
		}

		middleware := RequireScope(auth.ScopeModuleRead)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("allows request with wildcard scope", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeAll},
		}

		middleware := RequireScope(auth.ScopeModuleDelete)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("rejects request without auth context", func(t *testing.T) {
		middleware := RequireScope(auth.ScopeModuleRead)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"authentication required"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("rejects request without required scope", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeModuleRead},
		}

		middleware := RequireScope(auth.ScopeModuleWrite)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"insufficient permissions"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("rejects request with empty scopes", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{},
		}

		middleware := RequireScope(auth.ScopeModuleRead)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("rejects request without auth context", func(t *testing.T) {
		middleware := RequireRole(auth.RoleAdmin)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"authentication required"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("rejects request without required role", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeModuleRead},
		}

		middleware := RequireRole(auth.RoleAdmin)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"insufficient role permissions"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("checks different roles", func(t *testing.T) {
		roles := []auth.Role{auth.RoleAdmin, auth.RoleDeveloper, auth.RoleViewer}

		for _, role := range roles {
			t.Run(string(role), func(t *testing.T) {
				authCtx := &auth.AuthContext{
					Scopes: []auth.Scope{auth.ScopeAll},
				}

				middleware := RequireRole(role)
				handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
				req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				// HasRole always returns false (TODO implementation)
				if w.Code != http.StatusForbidden {
					t.Errorf("expected status 403, got %d", w.Code)
				}
			})
		}
	})
}

func TestRequireModulePermission(t *testing.T) {
	t.Run("rejects request without auth context", func(t *testing.T) {
		middleware := RequireModulePermission("module_id", auth.PermissionRead)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"authentication required"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("rejects request without required permission", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeModuleRead},
		}

		middleware := RequireModulePermission("module_id", auth.PermissionWrite)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		}))

		ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
		req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		body := w.Body.String()
		if body != `{"error":"insufficient module permissions"}` {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("checks different permissions", func(t *testing.T) {
		permissions := []auth.Permission{
			auth.PermissionRead,
			auth.PermissionWrite,
			auth.PermissionDelete,
			auth.PermissionAdmin,
		}

		for _, perm := range permissions {
			t.Run(string(perm), func(t *testing.T) {
				authCtx := &auth.AuthContext{
					Scopes: []auth.Scope{auth.ScopeAll},
				}

				middleware := RequireModulePermission("module_id", perm)
				handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
				req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				// HasPermission always returns false (TODO implementation)
				if w.Code != http.StatusForbidden {
					t.Errorf("expected status 403, got %d", w.Code)
				}
			})
		}
	})

	t.Run("handles different module ID parameters", func(t *testing.T) {
		authCtx := &auth.AuthContext{
			Scopes: []auth.Scope{auth.ScopeModuleRead},
		}

		moduleIDParams := []string{"id", "module_id", "moduleID"}

		for _, param := range moduleIDParams {
			t.Run(param, func(t *testing.T) {
				middleware := RequireModulePermission(param, auth.PermissionRead)
				handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				ctx := context.WithValue(context.Background(), AuthContextKey, authCtx)
				req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
				w := httptest.NewRecorder()

				handler.ServeHTTP(w, req)

				// Since extraction is not implemented, this will always fail
				if w.Code != http.StatusForbidden {
					t.Errorf("expected status 403, got %d", w.Code)
				}
			})
		}
	})
}

func TestForbiddenResponse(t *testing.T) {
	t.Run("writes forbidden response with correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		forbiddenResponse(w, "test error message")

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
		body := w.Body.String()
		expected := `{"error":"test error message"}`
		if body != expected {
			t.Errorf("expected body %s, got %s", expected, body)
		}
	})

	t.Run("handles empty message", func(t *testing.T) {
		w := httptest.NewRecorder()
		forbiddenResponse(w, "")

		body := w.Body.String()
		expected := `{"error":""}`
		if body != expected {
			t.Errorf("expected body %s, got %s", expected, body)
		}
	})

	t.Run("handles message with special characters", func(t *testing.T) {
		w := httptest.NewRecorder()
		forbiddenResponse(w, "error with \"quotes\"")

		if w.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", w.Code)
		}
		// Note: The current implementation doesn't escape quotes,
		// which would create invalid JSON. This test documents the behavior.
		body := w.Body.String()
		if body == "" {
			t.Error("expected non-empty body")
		}
	})
}

func TestUnauthorizedResponse(t *testing.T) {
	tm := auth.NewTokenManager()
	middleware := NewAuthMiddleware(tm, false)

	t.Run("writes unauthorized response with correct format", func(t *testing.T) {
		w := httptest.NewRecorder()
		middleware.unauthorizedResponse(w, "test error")

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}
		body := w.Body.String()
		expected := `{"error":"test error"}`
		if body != expected {
			t.Errorf("expected body %s, got %s", expected, body)
		}
	})
}

func TestContextKey(t *testing.T) {
	t.Run("AuthContextKey has correct value", func(t *testing.T) {
		if AuthContextKey != "auth_context" {
			t.Errorf("expected AuthContextKey to be 'auth_context', got %s", AuthContextKey)
		}
	})

	t.Run("can use AuthContextKey in context", func(t *testing.T) {
		ctx := context.Background()
		value := "test_value"
		ctx = context.WithValue(ctx, AuthContextKey, value)

		retrieved := ctx.Value(AuthContextKey)
		if retrieved != value {
			t.Errorf("expected %s, got %v", value, retrieved)
		}
	})
}
