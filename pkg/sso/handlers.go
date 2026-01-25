package sso

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/platinummonkey/spoke/pkg/auth"
)

// Handlers handles SSO-related HTTP requests
type Handlers struct {
	db              *sql.DB
	storage         *Storage
	factory         *ProviderFactory
	provisioner     *UserProvisioner
	sessionManager  *SessionManager
	baseURL         string
}

// NewHandlers creates a new SSO handlers instance
func NewHandlers(db *sql.DB, baseURL string) *Handlers {
	return &Handlers{
		db:             db,
		storage:        NewStorage(db),
		factory:        NewProviderFactory(baseURL),
		provisioner:    NewUserProvisioner(db),
		sessionManager: NewSessionManager(db),
		baseURL:        baseURL,
	}
}

// RegisterRoutes registers SSO routes
func (h *Handlers) RegisterRoutes(router *mux.Router) {
	// Provider configuration routes
	router.HandleFunc("/sso/providers", h.listProviders).Methods("GET")
	router.HandleFunc("/sso/providers", h.createProvider).Methods("POST")
	router.HandleFunc("/sso/providers/{name}", h.getProvider).Methods("GET")
	router.HandleFunc("/sso/providers/{name}", h.updateProvider).Methods("PUT")
	router.HandleFunc("/sso/providers/{name}", h.deleteProvider).Methods("DELETE")

	// SSO authentication routes
	router.HandleFunc("/auth/sso/{provider}/login", h.initiateLogin).Methods("GET")
	router.HandleFunc("/auth/sso/{provider}/callback", h.handleCallback).Methods("GET", "POST")
	router.HandleFunc("/auth/sso/logout", h.logout).Methods("GET", "POST")

	// SAML metadata endpoint
	router.HandleFunc("/sso/metadata/{provider}", h.getSAMLMetadata).Methods("GET")
}

// listProviders handles GET /sso/providers
func (h *Handlers) listProviders(w http.ResponseWriter, r *http.Request) {
	enabledOnly := r.URL.Query().Get("enabled") == "true"

	providers, err := h.storage.ListProviders(enabledOnly)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove sensitive data
	for _, p := range providers {
		h.sanitizeProvider(p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// createProvider handles POST /sso/providers
func (h *Handlers) createProvider(w http.ResponseWriter, r *http.Request) {
	var config ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if config.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if config.ProviderType == "" {
		http.Error(w, "provider_type is required", http.StatusBadRequest)
		return
	}

	// Check if provider already exists
	exists, err := h.storage.ProviderExists(config.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "provider with this name already exists", http.StatusConflict)
		return
	}

	// Validate configuration
	provider, err := h.factory.CreateProvider(&config)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid provider config: %v", err), http.StatusBadRequest)
		return
	}

	if err := provider.ValidateConfig(); err != nil {
		http.Error(w, fmt.Sprintf("invalid provider config: %v", err), http.StatusBadRequest)
		return
	}

	// Save to database
	if err := h.storage.CreateProvider(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sanitize response
	h.sanitizeProvider(&config)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(config)
}

// getProvider handles GET /sso/providers/{name}
func (h *Handlers) getProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	config, err := h.storage.GetProvider(name)
	if err == sql.ErrNoRows {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sanitize response
	h.sanitizeProvider(config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// updateProvider handles PUT /sso/providers/{name}
func (h *Handlers) updateProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Get existing config
	existing, err := h.storage.GetProvider(name)
	if err == sql.ErrNoRows {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse update
	var config ProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Keep ID and name from existing config
	config.ID = existing.ID
	config.Name = existing.Name

	// Validate configuration
	provider, err := h.factory.CreateProvider(&config)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid provider config: %v", err), http.StatusBadRequest)
		return
	}

	if err := provider.ValidateConfig(); err != nil {
		http.Error(w, fmt.Sprintf("invalid provider config: %v", err), http.StatusBadRequest)
		return
	}

	// Update in database
	if err := h.storage.UpdateProvider(&config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Sanitize response
	h.sanitizeProvider(&config)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// deleteProvider handles DELETE /sso/providers/{name}
func (h *Handlers) deleteProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if err := h.storage.DeleteProvider(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// initiateLogin handles GET /auth/sso/{provider}/login
func (h *Handlers) initiateLogin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]

	// Get provider config
	config, err := h.storage.GetProvider(providerName)
	if err == sql.ErrNoRows {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !config.Enabled {
		http.Error(w, "provider is disabled", http.StatusForbidden)
		return
	}

	// Create provider instance
	provider, err := h.factory.CreateProvider(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Generate state token
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}
	state := base64.URLEncoding.EncodeToString(stateBytes)

	// Store state in session/cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sso_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes
	})

	// Store provider name
	http.SetCookie(w, &http.Cookie{
		Name:     "sso_provider",
		Value:    providerName,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})

	// Store return URL if provided
	returnURL := r.URL.Query().Get("return_url")
	if returnURL != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "sso_return_url",
			Value:    returnURL,
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   600,
		})
	}

	// Initiate login
	if err := provider.InitiateLogin(w, r, state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleCallback handles GET/POST /auth/sso/{provider}/callback
func (h *Handlers) handleCallback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]

	// Verify state parameter
	stateCookie, err := r.Cookie("sso_state")
	if err != nil {
		http.Error(w, "missing state cookie", http.StatusBadRequest)
		return
	}

	stateParam := r.URL.Query().Get("state")
	if r.Method == "POST" {
		stateParam = r.FormValue("RelayState") // SAML uses RelayState
	}

	if stateParam != stateCookie.Value {
		http.Error(w, "invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get provider config
	config, err := h.storage.GetProvider(providerName)
	if err == sql.ErrNoRows {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create provider instance
	provider, err := h.factory.CreateProvider(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle callback
	ssoUser, err := provider.HandleCallback(w, r)
	if err != nil {
		http.Error(w, fmt.Sprintf("authentication failed: %v", err), http.StatusUnauthorized)
		return
	}

	// Provision user
	user, err := h.provisioner.ProvisionUser(ssoUser, config)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to provision user: %v", err), http.StatusInternalServerError)
		return
	}

	// Create SSO session
	sessionID := base64.URLEncoding.EncodeToString(stateBytes[:16])
	session := &SSOSession{
		ID:             sessionID,
		ProviderID:     config.ID,
		UserID:         user.ID,
		ExternalUserID: ssoUser.ExternalID,
		CreatedAt:      time.Now(),
		ExpiresAt:      time.Now().Add(24 * time.Hour),
	}

	if err := h.sessionManager.CreateSession(session); err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "sso_session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400, // 24 hours
	})

	// Clear temporary cookies
	http.SetCookie(w, &http.Cookie{Name: "sso_state", MaxAge: -1, Path: "/"})
	http.SetCookie(w, &http.Cookie{Name: "sso_provider", MaxAge: -1, Path: "/"})

	// Get return URL
	returnURL := "/"
	if returnCookie, err := r.Cookie("sso_return_url"); err == nil {
		returnURL = returnCookie.Value
		http.SetCookie(w, &http.Cookie{Name: "sso_return_url", MaxAge: -1, Path: "/"})
	}

	// Redirect to return URL
	http.Redirect(w, r, returnURL, http.StatusFound)
}

// logout handles GET/POST /auth/sso/logout
func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	// Get session
	sessionCookie, err := r.Cookie("sso_session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	session, err := h.sessionManager.GetSession(sessionCookie.Value)
	if err != nil {
		// Session not found, just clear cookie
		http.SetCookie(w, &http.Cookie{Name: "sso_session", MaxAge: -1, Path: "/"})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Delete session
	h.sessionManager.DeleteSession(session.ID)

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{Name: "sso_session", MaxAge: -1, Path: "/"})

	// Get provider and initiate logout if supported
	config, err := h.storage.GetProviderByID(session.ProviderID)
	if err == nil && config.Enabled {
		provider, err := h.factory.CreateProvider(config)
		if err == nil {
			provider.Logout(w, r, session.SAMLSessionIndex)
			return
		}
	}

	// Fallback: redirect to home
	http.Redirect(w, r, "/", http.StatusFound)
}

// getSAMLMetadata handles GET /sso/metadata/{provider}
func (h *Handlers) getSAMLMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerName := vars["provider"]

	config, err := h.storage.GetProvider(providerName)
	if err == sql.ErrNoRows {
		http.Error(w, "provider not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if config.ProviderType != ProviderTypeSAML {
		http.Error(w, "provider is not SAML", http.StatusBadRequest)
		return
	}

	provider, err := h.factory.CreateProvider(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	samlProvider, ok := provider.(*SAMLProvider)
	if !ok {
		http.Error(w, "provider is not SAML", http.StatusInternalServerError)
		return
	}

	metadata, err := samlProvider.GetMetadata()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(metadata)
}

// sanitizeProvider removes sensitive information from provider config
func (h *Handlers) sanitizeProvider(config *ProviderConfig) {
	if config.SAMLConfig != nil {
		config.SAMLConfig.PrivateKey = ""
	}
	if config.OAuth2Config != nil {
		config.OAuth2Config.ClientSecret = ""
	}
	if config.OIDCConfig != nil {
		config.OIDCConfig.ClientSecret = ""
	}
}

// GetAuthContext extracts authenticated user from SSO session
func (h *Handlers) GetAuthContext(r *http.Request) (*auth.AuthContext, error) {
	sessionCookie, err := r.Cookie("sso_session")
	if err != nil {
		return nil, fmt.Errorf("no SSO session")
	}

	session, err := h.sessionManager.GetSession(sessionCookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}

	// Fetch user
	user := &auth.User{}
	err = h.db.QueryRow(`
		SELECT id, username, email, full_name, is_bot, is_active, created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, session.UserID).Scan(&user.ID, &user.Username, &user.Email, &user.FullName,
		&user.IsBot, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}

	return &auth.AuthContext{
		User:   user,
		Scopes: []auth.Scope{auth.ScopeAll}, // SSO users get all scopes
	}, nil
}

var stateBytes = make([]byte, 32)

func init() {
	rand.Read(stateBytes)
}
