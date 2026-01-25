package auth

import "time"

// User represents a user or bot account
type User struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email,omitempty"`
	FullName    string    `json:"full_name,omitempty"`
	IsBot       bool      `json:"is_bot"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

// Organization represents an organization
type Organization struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Role represents organization-level roles
type Role string

const (
	RoleAdmin     Role = "admin"     // Full access to organization
	RoleDeveloper Role = "developer" // Can push and manage schemas
	RoleViewer    Role = "viewer"    // Read-only access
)

// Permission represents module-level permissions
type Permission string

const (
	PermissionRead   Permission = "read"   // Can view module
	PermissionWrite  Permission = "write"  // Can push versions
	PermissionDelete Permission = "delete" // Can delete module/versions
	PermissionAdmin  Permission = "admin"  // Full control over module
)

// Scope represents API token scopes
type Scope string

const (
	ScopeModuleRead     Scope = "module:read"
	ScopeModuleWrite    Scope = "module:write"
	ScopeModuleDelete   Scope = "module:delete"
	ScopeVersionRead    Scope = "version:read"
	ScopeVersionWrite   Scope = "version:write"
	ScopeVersionDelete  Scope = "version:delete"
	ScopeTokenCreate    Scope = "token:create"
	ScopeTokenRevoke    Scope = "token:revoke"
	ScopeOrgRead        Scope = "org:read"
	ScopeOrgWrite       Scope = "org:write"
	ScopeUserRead       Scope = "user:read"
	ScopeUserWrite      Scope = "user:write"
	ScopeAuditRead      Scope = "audit:read"
	ScopeAll            Scope = "*" // All permissions (for admin)
)

// APIToken represents an API token
type APIToken struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	TokenHash   string     `json:"-"` // Never expose hash
	TokenPrefix string     `json:"token_prefix"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Scopes      []Scope    `json:"scopes"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	RevokedBy   *int64     `json:"revoked_by,omitempty"`
	RevokeReason string    `json:"revoke_reason,omitempty"`
}

// ModulePermission represents module-level access control
type ModulePermission struct {
	ID             int64      `json:"id"`
	ModuleID       int64      `json:"module_id"`
	UserID         *int64     `json:"user_id,omitempty"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	Permission     Permission `json:"permission"`
	GrantedAt      time.Time  `json:"granted_at"`
	GrantedBy      *int64     `json:"granted_by,omitempty"`
}

// AuditLog represents a security audit log entry
type AuditLog struct {
	ID             int64     `json:"id"`
	UserID         *int64    `json:"user_id,omitempty"`
	OrganizationID *int64    `json:"organization_id,omitempty"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id,omitempty"`
	IPAddress      string    `json:"ip_address,omitempty"`
	UserAgent      string    `json:"user_agent,omitempty"`
	Status         string    `json:"status"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// RateLimitBucket represents a rate limit tracking entry
type RateLimitBucket struct {
	ID                   int64     `json:"id"`
	UserID               *int64    `json:"user_id,omitempty"`
	OrganizationID       *int64    `json:"organization_id,omitempty"`
	Endpoint             string    `json:"endpoint"`
	RequestsCount        int       `json:"requests_count"`
	WindowStart          time.Time `json:"window_start"`
	WindowDurationSeconds int      `json:"window_duration_seconds"`
}

// AuthContext holds authenticated user information
type AuthContext struct {
	User         *User
	Organization *Organization
	Token        *APIToken
	Scopes       []Scope
}

// HasScope checks if the context has a specific scope
func (ac *AuthContext) HasScope(scope Scope) bool {
	// Check for wildcard
	for _, s := range ac.Scopes {
		if s == ScopeAll {
			return true
		}
		if s == scope {
			return true
		}
	}
	return false
}

// HasRole checks if user has a specific role in organization
func (ac *AuthContext) HasRole(role Role) bool {
	// TODO: Query organization_members table
	return false
}

// HasPermission checks if user has permission on a module
func (ac *AuthContext) HasPermission(moduleID int64, perm Permission) bool {
	// TODO: Query module_permissions table
	return false
}
