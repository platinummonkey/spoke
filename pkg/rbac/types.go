package rbac

import (
	"time"
)

// Resource represents a resource type in the system
type Resource string

const (
	ResourceModule         Resource = "module"
	ResourceVersion        Resource = "version"
	ResourceDocumentation  Resource = "documentation"
	ResourceSettings       Resource = "settings"
	ResourceUser           Resource = "user"
	ResourceRole           Resource = "role"
	ResourceTeam           Resource = "team"
	ResourceOrganization   Resource = "organization"
)

// Action represents an action that can be performed on a resource
type Action string

const (
	ActionCreate     Action = "create"
	ActionRead       Action = "read"
	ActionUpdate     Action = "update"
	ActionDelete     Action = "delete"
	ActionPublish    Action = "publish"
	ActionDeprecate  Action = "deprecate"
	ActionInvite     Action = "invite"
	ActionRemove     Action = "remove"
	ActionUpdateRole Action = "update_role"
)

// Permission represents a specific permission (resource + action)
type Permission struct {
	Resource Resource `json:"resource"`
	Action   Action   `json:"action"`
}

// String returns a string representation of the permission
func (p Permission) String() string {
	return string(p.Resource) + ":" + string(p.Action)
}

// PermissionScope represents the scope at which a permission applies
type PermissionScope string

const (
	ScopeOrganization PermissionScope = "organization" // Organization-wide
	ScopeModule       PermissionScope = "module"       // Specific module
	ScopeGlobal       PermissionScope = "global"       // System-wide (superadmin)
)

// Role represents a role with a set of permissions
type Role struct {
	ID              int64              `json:"id"`
	Name            string             `json:"name"`
	DisplayName     string             `json:"display_name"`
	Description     string             `json:"description"`
	OrganizationID  *int64             `json:"organization_id,omitempty"` // nil for built-in roles
	Permissions     []Permission       `json:"permissions"`
	ParentRoleID    *int64             `json:"parent_role_id,omitempty"` // For role inheritance
	IsBuiltIn       bool               `json:"is_built_in"`
	IsCustom        bool               `json:"is_custom"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	CreatedBy       *int64             `json:"created_by,omitempty"`
}

// Built-in role names
const (
	RoleOrgAdmin           = "org:admin"
	RoleOrgDeveloper       = "org:developer"
	RoleOrgViewer          = "org:viewer"
	RoleModuleOwner        = "module:owner"
	RoleModuleContributor  = "module:contributor"
	RoleModuleViewer       = "module:viewer"
	RoleSuperAdmin         = "system:superadmin"
)

// UserRole represents a role assignment to a user
type UserRole struct {
	ID             int64           `json:"id"`
	UserID         int64           `json:"user_id"`
	RoleID         int64           `json:"role_id"`
	Scope          PermissionScope `json:"scope"`
	ResourceID     *string         `json:"resource_id,omitempty"` // Module name for module scope
	OrganizationID *int64          `json:"organization_id,omitempty"`
	GrantedBy      *int64          `json:"granted_by,omitempty"`
	GrantedAt      time.Time       `json:"granted_at"`
	ExpiresAt      *time.Time      `json:"expires_at,omitempty"`
}

// Team represents a team of users
type Team struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	Name           string    `json:"name"`
	DisplayName    string    `json:"display_name"`
	Description    string    `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      *int64    `json:"created_by,omitempty"`
}

// TeamMember represents a user's membership in a team
type TeamMember struct {
	ID        int64     `json:"id"`
	TeamID    int64     `json:"team_id"`
	UserID    int64     `json:"user_id"`
	RoleID    *int64    `json:"role_id,omitempty"` // Team-specific role
	AddedAt   time.Time `json:"added_at"`
	AddedBy   *int64    `json:"added_by,omitempty"`
}

// TeamRole represents a role assignment to a team
type TeamRole struct {
	ID             int64           `json:"id"`
	TeamID         int64           `json:"team_id"`
	RoleID         int64           `json:"role_id"`
	Scope          PermissionScope `json:"scope"`
	ResourceID     *string         `json:"resource_id,omitempty"`
	OrganizationID *int64          `json:"organization_id,omitempty"`
	GrantedBy      *int64          `json:"granted_by,omitempty"`
	GrantedAt      time.Time       `json:"granted_at"`
}

// PermissionCheck represents a permission check request
type PermissionCheck struct {
	UserID         int64           `json:"user_id"`
	Permission     Permission      `json:"permission"`
	Scope          PermissionScope `json:"scope"`
	ResourceID     *string         `json:"resource_id,omitempty"`
	OrganizationID *int64          `json:"organization_id,omitempty"`
}

// PermissionCheckResult represents the result of a permission check
type PermissionCheckResult struct {
	Allowed      bool      `json:"allowed"`
	Reason       string    `json:"reason,omitempty"`
	MatchedRoles []string  `json:"matched_roles,omitempty"`
	CheckedAt    time.Time `json:"checked_at"`
}

// PermissionCacheEntry represents a cached permission check
type PermissionCacheEntry struct {
	UserID         int64           `json:"user_id"`
	Permission     string          `json:"permission"` // Resource:Action
	Scope          PermissionScope `json:"scope"`
	ResourceID     *string         `json:"resource_id,omitempty"`
	OrganizationID *int64          `json:"organization_id,omitempty"`
	Allowed        bool            `json:"allowed"`
	ExpiresAt      time.Time       `json:"expires_at"`
	CreatedAt      time.Time       `json:"created_at"`
}

// RoleTemplate represents a template for creating custom roles
type RoleTemplate struct {
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// BuiltInRoles returns all built-in role definitions
func BuiltInRoles() []Role {
	return []Role{
		{
			Name:        RoleOrgAdmin,
			DisplayName: "Organization Admin",
			Description: "Full access to organization resources",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionCreate},
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceModule, Action: ActionUpdate},
				{Resource: ResourceModule, Action: ActionDelete},
				{Resource: ResourceVersion, Action: ActionPublish},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionDeprecate},
				{Resource: ResourceDocumentation, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionUpdate},
				{Resource: ResourceSettings, Action: ActionRead},
				{Resource: ResourceSettings, Action: ActionUpdate},
				{Resource: ResourceUser, Action: ActionInvite},
				{Resource: ResourceUser, Action: ActionRemove},
				{Resource: ResourceUser, Action: ActionUpdateRole},
				{Resource: ResourceRole, Action: ActionCreate},
				{Resource: ResourceRole, Action: ActionRead},
				{Resource: ResourceRole, Action: ActionUpdate},
				{Resource: ResourceRole, Action: ActionDelete},
				{Resource: ResourceTeam, Action: ActionCreate},
				{Resource: ResourceTeam, Action: ActionRead},
				{Resource: ResourceTeam, Action: ActionUpdate},
				{Resource: ResourceTeam, Action: ActionDelete},
			},
		},
		{
			Name:        RoleOrgDeveloper,
			DisplayName: "Organization Developer",
			Description: "Can create and update modules",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionCreate},
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceModule, Action: ActionUpdate},
				{Resource: ResourceVersion, Action: ActionPublish},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionUpdate},
			},
		},
		{
			Name:        RoleOrgViewer,
			DisplayName: "Organization Viewer",
			Description: "Read-only access to organization resources",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
			},
		},
		{
			Name:        RoleModuleOwner,
			DisplayName: "Module Owner",
			Description: "Full access to a specific module",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceModule, Action: ActionUpdate},
				{Resource: ResourceModule, Action: ActionDelete},
				{Resource: ResourceVersion, Action: ActionPublish},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionDeprecate},
				{Resource: ResourceDocumentation, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionUpdate},
			},
		},
		{
			Name:        RoleModuleContributor,
			DisplayName: "Module Contributor",
			Description: "Can push versions to a specific module",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionPublish},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
			},
		},
		{
			Name:        RoleModuleViewer,
			DisplayName: "Module Viewer",
			Description: "Read-only access to a specific module",
			IsBuiltIn:   true,
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
			},
		},
	}
}

// CommonRoleTemplates returns common role templates
func CommonRoleTemplates() []RoleTemplate {
	return []RoleTemplate{
		{
			Name:        "ci-bot",
			DisplayName: "CI/CD Bot",
			Description: "Automated CI/CD pipeline access",
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionPublish},
				{Resource: ResourceVersion, Action: ActionRead},
			},
		},
		{
			Name:        "auditor",
			DisplayName: "Auditor",
			Description: "Read-only access for auditing purposes",
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
				{Resource: ResourceSettings, Action: ActionRead},
			},
		},
		{
			Name:        "docs-manager",
			DisplayName: "Documentation Manager",
			Description: "Can manage documentation for modules",
			Permissions: []Permission{
				{Resource: ResourceModule, Action: ActionRead},
				{Resource: ResourceVersion, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionRead},
				{Resource: ResourceDocumentation, Action: ActionUpdate},
			},
		},
	}
}
