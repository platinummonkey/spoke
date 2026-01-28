package sso

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHandlers(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")
	assert.NotNil(t, handlers)
	assert.NotNil(t, handlers.db)
	assert.NotNil(t, handlers.storage)
	assert.NotNil(t, handlers.factory)
	assert.NotNil(t, handlers.provisioner)
	assert.NotNil(t, handlers.sessionManager)
	assert.Equal(t, "https://spoke.example.com", handlers.baseURL)
}

func TestRegisterRoutes(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")
	router := mux.NewRouter()
	handlers.RegisterRoutes(router)

	// Verify routes are registered
	err = router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		t.Logf("Route: %s %v", path, methods)
		return nil
	})
	assert.NoError(t, err)
}

func TestListProviders_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	// Mock database response
	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true,
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers ORDER BY name").WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/sso/providers", nil)
	w := httptest.NewRecorder()

	handlers.listProviders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var providers []*ProviderConfig
	err = json.Unmarshal(w.Body.Bytes(), &providers)
	require.NoError(t, err)
	assert.Len(t, providers, 1)
	assert.Equal(t, "test-provider", providers[0].Name)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListProviders_EnabledOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	})

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE enabled = true ORDER BY name").WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/sso/providers?enabled=true", nil)
	w := httptest.NewRecorder()

	handlers.listProviders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListProviders_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers").WillReturnError(errors.New("database error"))

	req := httptest.NewRequest("GET", "/sso/providers", nil)
	w := httptest.NewRecorder()

	handlers.listProviders(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateProvider_ValidationFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	// Create a provider config that will fail validation (missing required fields)
	jsonBody := `{
		"name": "test-provider",
		"provider_type": "oauth2",
		"provider_name": "generic_oauth2",
		"enabled": true,
		"oauth2_config": {
			"client_id": "test-client-id"
		},
		"attribute_mapping": {
			"user_id": "sub",
			"email": "email"
		}
	}`

	// Mock provider existence check
	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader([]byte(jsonBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	// Should fail validation due to missing required fields
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid provider config")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateProvider_InvalidJSON(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestCreateProvider_MissingName(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	config := &ProviderConfig{
		ProviderType: ProviderTypeOIDC,
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name is required")
}

func TestCreateProvider_MissingProviderType(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	config := &ProviderConfig{
		Name: "test-provider",
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "provider_type is required")
}

func TestCreateProvider_AlreadyExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	config := &ProviderConfig{
		Name:         "test-provider",
		ProviderType: ProviderTypeOIDC,
	}
	body, _ := json.Marshal(config)

	// Mock provider existence check - returns true
	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "provider with this name already exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProvider_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true,
		"viewer", nil, nil, []byte(`{"client_id":"test","client_secret":"secret","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/sso/providers/test-provider", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-provider"})
	w := httptest.NewRecorder()

	handlers.getProvider(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var provider ProviderConfig
	err = json.Unmarshal(w.Body.Bytes(), &provider)
	require.NoError(t, err)
	assert.Equal(t, "test-provider", provider.Name)
	// Client secret should be sanitized
	assert.Empty(t, provider.OIDCConfig.ClientSecret)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetProvider_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/sso/providers/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.getProvider(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "provider not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProvider_ValidationFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	// Mock getting existing provider
	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oauth2", "generic_oauth2", true, true,
		"viewer", nil, []byte(`{"client_id":"test","client_secret":"secret","auth_url":"https://example.com/auth","token_url":"https://example.com/token","redirect_url":"https://spoke.example.com/callback","scopes":["openid","email"]}`), nil,
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	// Update with invalid config (missing required fields)
	jsonBody := `{
		"enabled": true,
		"oauth2_config": {
			"client_id": "test-client-id"
		},
		"attribute_mapping": {
			"user_id": "sub",
			"email": "email"
		}
	}`

	req := httptest.NewRequest("PUT", "/sso/providers/test-provider", bytes.NewReader([]byte(jsonBody)))
	req = mux.SetURLVars(req, map[string]string{"name": "test-provider"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.updateProvider(w, req)

	// Should fail validation
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid provider config")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProvider_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	config := &ProviderConfig{
		Enabled: false,
	}
	body, _ := json.Marshal(config)

	req := httptest.NewRequest("PUT", "/sso/providers/nonexistent", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "nonexistent"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.updateProvider(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "provider not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteProvider_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectExec("DELETE FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest("DELETE", "/sso/providers/test-provider", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-provider"})
	w := httptest.NewRecorder()

	handlers.deleteProvider(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteProvider_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectExec("DELETE FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnError(errors.New("database error"))

	req := httptest.NewRequest("DELETE", "/sso/providers/test-provider", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-provider"})
	w := httptest.NewRecorder()

	handlers.deleteProvider(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitiateLogin_ProviderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/auth/sso/nonexistent/login", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.initiateLogin(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "provider not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInitiateLogin_ProviderDisabled(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", false, true, // enabled = false
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/auth/sso/test-provider/login", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	w := httptest.NewRecorder()

	handlers.initiateLogin(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "provider is disabled")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCallback_MissingStateCookie(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("GET", "/auth/sso/test-provider/callback?state=test-state", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	w := httptest.NewRecorder()

	handlers.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "missing state cookie")
}

func TestHandleCallback_InvalidState(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("GET", "/auth/sso/test-provider/callback?state=wrong-state", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	req.AddCookie(&http.Cookie{Name: "sso_state", Value: "correct-state"})
	w := httptest.NewRecorder()

	handlers.handleCallback(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid state parameter")
}

func TestHandleCallback_ProviderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/auth/sso/nonexistent/callback?state=test-state", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "nonexistent"})
	req.AddCookie(&http.Cookie{Name: "sso_state", Value: "test-state"})
	w := httptest.NewRecorder()

	handlers.handleCallback(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "provider not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogout_NoSessionCookie(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("GET", "/auth/sso/logout", nil)
	w := httptest.NewRecorder()

	handlers.logout(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
}

func TestLogout_SessionNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_sessions WHERE id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs("test-session").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/auth/sso/logout", nil)
	req.AddCookie(&http.Cookie{Name: "sso_session", Value: "test-session"})
	w := httptest.NewRecorder()

	handlers.logout(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))

	// Check that cookie is cleared
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "sso_session" {
			sessionCookie = c
			break
		}
	}
	assert.NotNil(t, sessionCookie)
	assert.Equal(t, -1, sessionCookie.MaxAge)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSAMLMetadata_ProviderNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("nonexistent").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/sso/metadata/nonexistent", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "nonexistent"})
	w := httptest.NewRecorder()

	handlers.getSAMLMetadata(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "provider not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSAMLMetadata_NotSAMLProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true, // Not SAML
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/sso/metadata/test-provider", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	w := httptest.NewRecorder()

	handlers.getSAMLMetadata(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "provider is not SAML")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSanitizeProvider(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	tests := []struct {
		name   string
		config *ProviderConfig
	}{
		{
			name: "SAML config sanitization",
			config: &ProviderConfig{
				SAMLConfig: &SAMLConfig{
					PrivateKey: "secret-key",
				},
			},
		},
		{
			name: "OAuth2 config sanitization",
			config: &ProviderConfig{
				OAuth2Config: &OAuth2Config{
					ClientSecret: "secret",
				},
			},
		},
		{
			name: "OIDC config sanitization",
			config: &ProviderConfig{
				OIDCConfig: &OIDCConfig{
					ClientSecret: "secret",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers.sanitizeProvider(tt.config)

			if tt.config.SAMLConfig != nil {
				assert.Empty(t, tt.config.SAMLConfig.PrivateKey)
			}
			if tt.config.OAuth2Config != nil {
				assert.Empty(t, tt.config.OAuth2Config.ClientSecret)
			}
			if tt.config.OIDCConfig != nil {
				assert.Empty(t, tt.config.OIDCConfig.ClientSecret)
			}
		})
	}
}

func TestGetAuthContext_NoSession(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("GET", "/test", nil)

	authCtx, err := handlers.GetAuthContext(req)

	assert.Error(t, err)
	assert.Nil(t, authCtx)
	assert.Contains(t, err.Error(), "no SSO session")
}

func TestGetAuthContext_InvalidSession(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_sessions WHERE id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs("invalid-session").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "sso_session", Value: "invalid-session"})

	authCtx, err := handlers.GetAuthContext(req)

	assert.Error(t, err)
	assert.Nil(t, authCtx)
	assert.Contains(t, err.Error(), "invalid session")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAuthContext_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	now := time.Now()
	userID := int64(123)

	// Mock session query
	sessionRows := sqlmock.NewRows([]string{
		"id", "provider_id", "user_id", "external_user_id", "saml_session_index", "created_at", "expires_at",
	}).AddRow(
		"test-session", 1, userID, "ext-user-123", "", now, now.Add(24*time.Hour),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_sessions WHERE id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs("test-session").
		WillReturnRows(sessionRows)

	// Mock user query
	userRows := sqlmock.NewRows([]string{
		"id", "username", "email", "full_name", "is_bot", "is_active", "created_at", "updated_at", "last_login_at",
	}).AddRow(
		userID, "testuser", "test@example.com", "Test User", false, true, now, now, &now,
	)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").
		WithArgs(userID).
		WillReturnRows(userRows)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "sso_session", Value: "test-session"})

	authCtx, err := handlers.GetAuthContext(req)

	require.NoError(t, err)
	assert.NotNil(t, authCtx)
	assert.NotNil(t, authCtx.User)
	assert.Equal(t, userID, authCtx.User.ID)
	assert.Equal(t, "testuser", authCtx.User.Username)
	assert.Equal(t, "test@example.com", authCtx.User.Email)
	assert.Contains(t, authCtx.Scopes, auth.ScopeAll)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestHandleCallback_SAMLRelayState(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	req := httptest.NewRequest("POST", "/auth/sso/test-provider/callback", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	req.AddCookie(&http.Cookie{Name: "sso_state", Value: "test-state"})
	req.Form = map[string][]string{
		"RelayState": {"test-state"},
	}
	w := httptest.NewRecorder()

	// This will fail due to provider not found, but we're testing state validation
	handlers.handleCallback(w, req)

	// Should not fail on state validation
	assert.NotContains(t, w.Body.String(), "invalid state parameter")
}

func TestInitiateLogin_WithReturnURL(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true,
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/auth/sso/test-provider/login?return_url=/dashboard", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	w := httptest.NewRecorder()

	handlers.initiateLogin(w, req)

	// Check that return_url cookie is set
	cookies := w.Result().Cookies()
	var returnURLCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "sso_return_url" {
			returnURLCookie = c
			break
		}
	}
	assert.NotNil(t, returnURLCookie)
	assert.Equal(t, "/dashboard", returnURLCookie.Value)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateProvider_InvalidProviderConfig(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	config := &ProviderConfig{
		Name:         "test-provider",
		ProviderType: ProviderTypeOIDC,
		Enabled:      true,
		// Missing required OIDCConfig
	}
	body, _ := json.Marshal(config)

	// Mock provider existence check
	mock.ExpectQuery("SELECT EXISTS").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	req := httptest.NewRequest("POST", "/sso/providers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.createProvider(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid provider config")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProvider_InvalidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	// Mock getting existing provider
	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true,
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnRows(rows)

	req := httptest.NewRequest("PUT", "/sso/providers/test-provider", bytes.NewReader([]byte("invalid json")))
	req = mux.SetURLVars(req, map[string]string{"name": "test-provider"})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handlers.updateProvider(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAuthContext_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	now := time.Now()

	// Mock session query
	sessionRows := sqlmock.NewRows([]string{
		"id", "provider_id", "user_id", "external_user_id", "saml_session_index", "created_at", "expires_at",
	}).AddRow(
		"test-session", 1, 123, "ext-user-123", "", now, now.Add(24*time.Hour),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_sessions WHERE id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs("test-session").
		WillReturnRows(sessionRows)

	// Mock user query - user not found
	mock.ExpectQuery("SELECT (.+) FROM users WHERE id = \\$1").
		WithArgs(int64(123)).
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "sso_session", Value: "test-session"})

	authCtx, err := handlers.GetAuthContext(req)

	assert.Error(t, err)
	assert.Nil(t, authCtx)
	assert.Contains(t, err.Error(), "failed to fetch user")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestLogout_WithProvider(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	now := time.Now()

	// Mock session query
	sessionRows := sqlmock.NewRows([]string{
		"id", "provider_id", "user_id", "external_user_id", "saml_session_index", "created_at", "expires_at",
	}).AddRow(
		"test-session", 1, 123, "ext-user-123", "", now, now.Add(24*time.Hour),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_sessions WHERE id = \\$1 AND expires_at > NOW\\(\\)").
		WithArgs("test-session").
		WillReturnRows(sessionRows)

	// Mock delete session
	mock.ExpectExec("DELETE FROM sso_sessions WHERE id = \\$1").
		WithArgs("test-session").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Mock provider query - provider disabled
	providerRows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", false, true, // disabled
		"viewer", nil, nil, []byte(`{"client_id":"test","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), now, now,
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE id = \\$1").
		WithArgs(int64(1)).
		WillReturnRows(providerRows)

	req := httptest.NewRequest("GET", "/auth/sso/logout", nil)
	req.AddCookie(&http.Cookie{Name: "sso_session", Value: "test-session"})
	w := httptest.NewRecorder()

	handlers.logout(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/", w.Header().Get("Location"))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSAMLMetadata_DatabaseError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	mock.ExpectQuery("SELECT (.+) FROM sso_providers WHERE name = \\$1").
		WithArgs("test-provider").
		WillReturnError(fmt.Errorf("database error"))

	req := httptest.NewRequest("GET", "/sso/metadata/test-provider", nil)
	req = mux.SetURLVars(req, map[string]string{"provider": "test-provider"})
	w := httptest.NewRecorder()

	handlers.getSAMLMetadata(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database error")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListProviders_Sanitization(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	handlers := NewHandlers(db, "https://spoke.example.com")

	// Mock database response with sensitive data
	rows := sqlmock.NewRows([]string{
		"id", "name", "provider_type", "provider_name", "enabled", "auto_provision",
		"default_role", "saml_config", "oauth2_config", "oidc_config",
		"group_mapping", "attribute_mapping", "created_at", "updated_at",
	}).AddRow(
		1, "test-provider", "oidc", "google", true, true,
		"viewer", nil, nil, []byte(`{"client_id":"test","client_secret":"should-be-removed","issuer_url":"https://accounts.google.com"}`),
		nil, []byte(`{"user_id":"sub","email":"email"}`), time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT (.+) FROM sso_providers ORDER BY name").WillReturnRows(rows)

	req := httptest.NewRequest("GET", "/sso/providers", nil)
	w := httptest.NewRecorder()

	handlers.listProviders(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var providers []*ProviderConfig
	err = json.Unmarshal(w.Body.Bytes(), &providers)
	require.NoError(t, err)
	assert.Len(t, providers, 1)

	// Verify client secret is sanitized
	assert.Empty(t, providers[0].OIDCConfig.ClientSecret)

	assert.NoError(t, mock.ExpectationsWereMet())
}
