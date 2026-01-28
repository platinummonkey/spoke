// Package auth provides user authentication and API token management for the Spoke registry.
//
// # Overview
//
// This package implements core authentication infrastructure including user management,
// API token generation with cryptographic security, scope-based permissions, and security
// audit logging. It supports both human users and bot accounts with fine-grained access control.
//
// # Key Components
//
// User Management: Registration, login, bot accounts
//
//	user := &auth.User{
//		Username: "alice",
//		Email:    "alice@example.com",
//		IsBot:    false,
//	}
//
// API Tokens: Secure token generation with prefix display, scopes, and expiration
//
//	token := &auth.APIToken{
//		Name:      "CI/CD Pipeline",
//		UserID:    user.ID,
//		Scopes:    []auth.Scope{auth.ScopeModuleRead, auth.ScopeVersionPublish},
//		ExpiresAt: time.Now().Add(90 * 24 * time.Hour),
//	}
//	// Token format: spoke_[base64url(32 random bytes)]
//	// Stored as SHA256 hash for security
//
// Scopes: Fine-grained API permissions
//
//	ScopeModuleRead     - Read module metadata
//	ScopeModuleWrite    - Create/update modules
//	ScopeVersionPublish - Publish new versions
//	ScopeTokenCreate    - Create API tokens
//	ScopeOrgWrite       - Manage organization
//	ScopeAll            - Full access
//
// Role-Based Permissions: Organization-level roles
//
//	RoleAdmin     - Full organization access
//	RoleDeveloper - Create/update modules
//	RoleViewer    - Read-only access
//
// # Authentication Flow
//
// Token Generation:
//
//	generator := auth.NewTokenGenerator()
//	tokenString, hash, err := generator.Generate()
//	// tokenString: spoke_xxx (give to user, display once)
//	// hash: SHA256(tokenString) (store in database)
//
// Token Validation:
//
//	manager := auth.NewTokenManager(db)
//	token, err := manager.Validate(ctx, tokenString)
//	if err != nil {
//		return errors.New("invalid token")
//	}
//	// Check scopes
//	if !token.HasScope(auth.ScopeModuleWrite) {
//		return errors.New("insufficient permissions")
//	}
//
// # Authorization Context
//
// AuthContext: Request-scoped authentication state
//
//	type AuthContext struct {
//		User         *User
//		Organization *orgs.Organization
//		Token        *APIToken
//		Permissions  PermissionChecker
//	}
//
// Middleware injects AuthContext into request:
//
//	authCtx := auth.GetAuthContext(r)
//	if authCtx == nil || authCtx.User == nil {
//		http.Error(w, "Unauthorized", http.StatusUnauthorized)
//		return
//	}
//
// # Module-Level Permissions
//
// Beyond org-level roles, users can have per-module permissions:
//
//	permission := &auth.ModulePermission{
//		UserID:     user.ID,
//		ModuleName: "user-service",
//		Permission: auth.PermissionWrite,
//	}
//	// Permission levels: Read, Write, Delete, Admin
//
// Check module access:
//
//	canWrite, err := manager.CheckModulePermission(ctx, userID, moduleName, auth.PermissionWrite)
//
// # Security Audit Logging
//
// The package integrates with pkg/audit for security events:
//
//	auditLogger.LogAuthentication(ctx, &audit.AuthEvent{
//		UserID:    user.ID,
//		TokenID:   token.ID,
//		IPAddress: r.RemoteAddr,
//		UserAgent: r.UserAgent(),
//		Success:   true,
//	})
//
// # Token Lifecycle Management
//
// Create token:
//
//	tokenString, err := manager.CreateToken(ctx, &auth.CreateTokenRequest{
//		UserID:    user.ID,
//		Name:      "Deploy Bot",
//		Scopes:    []auth.Scope{auth.ScopeVersionPublish},
//		ExpiresAt: time.Now().Add(365 * 24 * time.Hour),
//	})
//
// Revoke token:
//
//	err := manager.RevokeToken(ctx, tokenID)
//
// List user's tokens:
//
//	tokens, err := manager.ListTokens(ctx, userID)
//	for _, token := range tokens {
//		fmt.Printf("%s: %s (expires %v)\n",
//			token.DisplayPrefix, token.Name, token.ExpiresAt)
//	}
//
// Cleanup expired tokens:
//
//	deleted, err := manager.CleanupExpired(ctx)
//	fmt.Printf("Deleted %d expired tokens\n", deleted)
//
// # Bot Accounts
//
// Bot accounts are users with IsBot=true, typically for automation:
//
//	bot := &auth.User{
//		Username: "github-actions",
//		Email:    "bot@example.com",
//		IsBot:    true,
//	}
//	// Bots can't login via password
//	// Bots use API tokens only
//
// # Related Packages
//
//   - pkg/rbac: Role-based access control (organization permissions)
//   - pkg/sso: Single sign-on integration
//   - pkg/orgs: Organization management
//   - pkg/audit: Security audit logging
//   - pkg/middleware: HTTP authentication middleware
package auth
