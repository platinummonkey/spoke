// Package rbac provides role-based access control (RBAC) for the Spoke protobuf schema registry.
//
// # Overview
//
// This package implements a flexible, multi-tenant RBAC system for controlling access to
// modules, versions, documentation, and administrative functions. It supports organization-level,
// module-level, and system-level permissions with role inheritance and team-based access control.
//
// # Architecture
//
// The RBAC system consists of five key components:
//
//   1. Resources: Things that can be accessed (modules, versions, documentation, users, etc.)
//   2. Actions: Operations that can be performed (create, read, update, delete, publish, etc.)
//   3. Permissions: Combinations of resource + action (e.g., "module:create")
//   4. Roles: Named collections of permissions (e.g., "org:admin", "module:contributor")
//   5. Assignments: Bindings of roles to users or teams with specific scopes
//
// # Resources and Actions
//
// Resources define what can be controlled:
//
//	ResourceModule         - Protobuf module
//	ResourceVersion        - Specific version of a module
//	ResourceDocumentation  - Module documentation
//	ResourceSettings       - Organization or system settings
//	ResourceUser           - User management
//	ResourceRole           - Role management
//	ResourceTeam           - Team management
//	ResourceOrganization   - Organization management
//
// Actions define what can be done:
//
//	ActionCreate      - Create new resource
//	ActionRead        - View resource
//	ActionUpdate      - Modify resource
//	ActionDelete      - Remove resource
//	ActionPublish     - Publish a version
//	ActionDeprecate   - Mark version as deprecated
//	ActionInvite      - Invite user to organization
//	ActionRemove      - Remove user from organization
//	ActionUpdateRole  - Change user's role
//
// Permissions combine these:
//
//	permission := rbac.Permission{
//		Resource: rbac.ResourceModule,
//		Action:   rbac.ActionCreate,
//	}
//	// Represents "module:create"
//
// # Permission Scopes
//
// Permissions can be granted at three levels:
//
// Global Scope: System-wide access (superadmin only)
//
//	// Superadmin can perform action on ANY resource
//	rbac.ScopeGlobal
//
// Organization Scope: Access to all resources in an organization
//
//	// User can create modules in organization 123
//	rbac.ScopeOrganization + OrganizationID: 123
//
// Module Scope: Access to a specific module
//
//	// User can publish versions for "user-service" module only
//	rbac.ScopeModule + ResourceID: "user-service"
//
// # Built-In Roles
//
// The system provides seven built-in roles that cover common access patterns:
//
// Organization Roles:
//
//	org:admin      - Full access to organization (manage users, modules, settings)
//	org:developer  - Create and update modules, publish versions
//	org:viewer     - Read-only access to all organization resources
//
// Module Roles:
//
//	module:owner       - Full control over specific module (update, delete, publish)
//	module:contributor - Publish versions to specific module (read + publish)
//	module:viewer      - Read-only access to specific module
//
// System Roles:
//
//	system:superadmin - Global system access (manage all organizations)
//
// Built-in roles are immutable and defined in types.go:
//
//	roles := rbac.BuiltInRoles()
//	for _, role := range roles {
//		fmt.Printf("%s: %s\n", role.Name, role.Description)
//		for _, perm := range role.Permissions {
//			fmt.Printf("  - %s\n", perm.String())
//		}
//	}
//
// # Custom Roles
//
// Organizations can create custom roles tailored to their needs:
//
//	// Create a CI/CD bot role
//	customRole := &rbac.Role{
//		Name:           "ci-bot",
//		DisplayName:    "CI/CD Bot",
//		Description:    "Automated publishing from CI pipeline",
//		OrganizationID: &orgID,
//		IsCustom:       true,
//		Permissions: []rbac.Permission{
//			{Resource: rbac.ResourceModule, Action: rbac.ActionRead},
//			{Resource: rbac.ResourceVersion, Action: rbac.ActionPublish},
//		},
//	}
//	err := store.CreateRole(ctx, customRole)
//
// Common role templates are provided:
//
//	templates := rbac.CommonRoleTemplates()
//	// Returns: ci-bot, auditor, docs-manager
//
// # Role Assignment
//
// Roles are assigned to users with a specific scope:
//
//	// Assign org admin role to user
//	assignment := &rbac.UserRole{
//		UserID:         user.ID,
//		RoleID:         orgAdminRole.ID,
//		Scope:          rbac.ScopeOrganization,
//		OrganizationID: &org.ID,
//		GrantedBy:      &currentUser.ID,
//	}
//	err := store.AssignRole(ctx, assignment)
//
//	// Assign module contributor role for specific module
//	assignment := &rbac.UserRole{
//		UserID:         user.ID,
//		RoleID:         contributorRole.ID,
//		Scope:          rbac.ScopeModule,
//		ResourceID:     &moduleName,  // "user-service"
//		OrganizationID: &org.ID,
//		GrantedBy:      &currentUser.ID,
//	}
//	err := store.AssignRole(ctx, assignment)
//
// # Permission Checking
//
// The Checker interface provides permission evaluation:
//
//	checker := rbac.NewPermissionChecker(db, 5*time.Minute)
//
//	// Check if user can publish version
//	check := rbac.PermissionCheck{
//		UserID: user.ID,
//		Permission: rbac.Permission{
//			Resource: rbac.ResourceVersion,
//			Action:   rbac.ActionPublish,
//		},
//		Scope:          rbac.ScopeModule,
//		ResourceID:     &moduleName,
//		OrganizationID: &org.ID,
//	}
//
//	result, err := checker.CheckPermission(ctx, check)
//	if result.Allowed {
//		// User can publish
//		fmt.Printf("Granted by: %v\n", result.MatchedRoles)
//	} else {
//		fmt.Printf("Denied: %s\n", result.Reason)
//	}
//
// The checker evaluates permissions by:
//   1. Loading all roles assigned to the user
//   2. Checking if any role grants the requested permission
//   3. Verifying the role's scope matches the request
//   4. Caching the result (if caching enabled)
//
// # HTTP Middleware
//
// The PermissionMiddleware integrates RBAC with HTTP handlers:
//
//	// Create middleware
//	middleware := rbac.NewPermissionMiddleware(checker)
//
//	// Protect endpoint: require org admin
//	router.Handle("/modules",
//		middleware.RequirePermission(
//			rbac.ResourceModule,
//			rbac.ActionCreate,
//			rbac.ScopeOrganization,
//		)(createModuleHandler),
//	).Methods("POST")
//
//	// Protect endpoint: require module-specific permission
//	router.Handle("/modules/{name}/versions",
//		middleware.RequireModulePermission(rbac.ActionPublish)(publishVersionHandler),
//	).Methods("POST")
//
// The middleware:
//   1. Extracts authenticated user from request context
//   2. Performs permission check
//   3. Returns 401 Unauthorized if not authenticated
//   4. Returns 403 Forbidden if permission denied
//   5. Calls next handler if permission granted
//
// # Teams
//
// Teams enable group-based permission management:
//
//	// Create a team
//	team := &rbac.Team{
//		OrganizationID: org.ID,
//		Name:           "backend-team",
//		DisplayName:    "Backend Engineering",
//		Description:    "Backend service developers",
//	}
//	err := store.CreateTeam(ctx, team)
//
//	// Add user to team
//	member := &rbac.TeamMember{
//		TeamID: team.ID,
//		UserID: user.ID,
//		AddedBy: &currentUser.ID,
//	}
//	err := store.AddTeamMember(ctx, member)
//
//	// Grant team role for all backend modules
//	teamRole := &rbac.TeamRole{
//		TeamID:         team.ID,
//		RoleID:         contributorRole.ID,
//		Scope:          rbac.ScopeModule,
//		ResourceID:     &backendModulePattern,  // "backend-*"
//		OrganizationID: &org.ID,
//	}
//	err := store.AssignTeamRole(ctx, teamRole)
//
// Users inherit permissions from their team memberships. This simplifies management when
// multiple users need the same access pattern.
//
// # Role Inheritance
//
// Roles can inherit permissions from parent roles:
//
//	// Create a specialized admin role
//	securityAdmin := &rbac.Role{
//		Name:         "org:security-admin",
//		DisplayName:  "Security Administrator",
//		Description:  "Admin with additional security permissions",
//		ParentRoleID: &orgAdminRole.ID,  // Inherits org:admin permissions
//		Permissions: []rbac.Permission{
//			// Additional security-specific permissions
//			{Resource: rbac.ResourceAuditLog, Action: rbac.ActionRead},
//			{Resource: rbac.ResourceSecuritySettings, Action: rbac.ActionUpdate},
//		},
//	}
//
// Permission evaluation walks the inheritance chain, collecting permissions from all ancestors.
//
// # Permission Caching
//
// The permission checker caches results to reduce database load:
//
//	// Enable caching with 5-minute TTL
//	checker := rbac.NewPermissionChecker(db, 5*time.Minute)
//
//	// First check: queries database
//	result, _ := checker.CheckPermission(ctx, check)
//
//	// Second check: returns cached result (within TTL)
//	result, _ := checker.CheckPermission(ctx, check)
//
//	// Invalidate cache when roles change
//	err := checker.InvalidateCache(ctx, user.ID)
//
// Cache entries are scoped to (userID, permission, scope, resourceID, organizationID),
// ensuring correctness across different contexts.
//
// # Database Schema
//
// The RBAC system uses six database tables:
//
//   - roles: Role definitions with permissions JSON
//   - user_roles: Role assignments to users
//   - teams: Team definitions
//   - team_members: Team membership
//   - team_roles: Role assignments to teams
//   - permission_cache: Cached permission check results
//
// Schema migrations are provided in migrations.go:
//
//	err := rbac.RunMigrations(db)
//
// # Usage Examples
//
// Complete workflow for multi-tenant access control:
//
//	// 1. Initialize RBAC system
//	store := rbac.NewStore(db)
//	checker := rbac.NewPermissionChecker(db, 5*time.Minute)
//	middleware := rbac.NewPermissionMiddleware(checker)
//
//	// 2. Create organization
//	org := createOrganization("Acme Corp")
//
//	// 3. Invite admin user
//	admin := inviteUser("admin@acme.com", org.ID)
//	orgAdminRole := findRole("org:admin")
//	store.AssignRole(ctx, &rbac.UserRole{
//		UserID:         admin.ID,
//		RoleID:         orgAdminRole.ID,
//		Scope:          rbac.ScopeOrganization,
//		OrganizationID: &org.ID,
//	})
//
//	// 4. Create developer team
//	team, _ := store.CreateTeam(ctx, &rbac.Team{
//		OrganizationID: org.ID,
//		Name:           "developers",
//		DisplayName:    "Developers",
//	})
//
//	// 5. Add developers to team
//	for _, email := range []string{"alice@acme.com", "bob@acme.com"} {
//		user := inviteUser(email, org.ID)
//		store.AddTeamMember(ctx, &rbac.TeamMember{
//			TeamID: team.ID,
//			UserID: user.ID,
//		})
//	}
//
//	// 6. Grant team developer role
//	devRole := findRole("org:developer")
//	store.AssignTeamRole(ctx, &rbac.TeamRole{
//		TeamID:         team.ID,
//		RoleID:         devRole.ID,
//		Scope:          rbac.ScopeOrganization,
//		OrganizationID: &org.ID,
//	})
//
//	// 7. Protect API endpoints
//	router.Handle("/modules", middleware.RequirePermission(
//		rbac.ResourceModule, rbac.ActionCreate, rbac.ScopeOrganization,
//	)(createModuleHandler))
//
// # Design Decisions
//
// Scope-Based Access: Permissions are scoped to organization or module, enabling fine-grained
// control without complex ACLs. A user can be an admin in one org and viewer in another.
//
// Role Composition: Permissions are grouped into roles rather than granted individually.
// This reduces complexity and makes it easier to understand what access a user has.
//
// Built-In + Custom: Providing built-in roles covers 80% of use cases, while custom roles
// enable specialized access patterns without bloating the core role set.
//
// Team-Based Assignment: Teams simplify permission management for large organizations.
// Instead of assigning roles to 50 developers individually, assign once to the team.
//
// Caching with TTL: Permission checks are cached to reduce database load on hot paths
// (every API request). Cache invalidation on role changes prevents stale permissions.
//
// Permission JSON Storage: Role permissions are stored as JSON arrays in the roles table.
// This trades normalization for query simplicity - fetching a role loads all permissions
// in one query instead of joining a permission table.
//
// # Performance Considerations
//
// Permission checks can be expensive in high-traffic APIs. Optimize by:
//
// 1. Enable caching with appropriate TTL:
//
//	checker := rbac.NewPermissionChecker(db, 5*time.Minute)
//
// 2. Minimize database round trips by loading roles once per request:
//
//	roles, _ := checker.GetUserRoles(ctx, userID, &orgID)
//	// Use roles for multiple checks
//
// 3. Use database indexes on permission_cache:
//
//	CREATE INDEX idx_permission_cache_lookup
//	ON permission_cache(user_id, permission, scope, resource_id, organization_id)
//	WHERE expires_at > NOW();
//
// 4. Denormalize permissions for read-heavy workloads (store computed permissions on user).
//
// 5. Consider read replicas for permission checks (they don't need strong consistency).
//
// # Testing
//
// Mock the Checker interface for unit tests:
//
//	type mockChecker struct {
//		rbac.Checker
//		checkFunc func(context.Context, rbac.PermissionCheck) (*rbac.PermissionCheckResult, error)
//	}
//
//	func (m *mockChecker) CheckPermission(ctx context.Context, check rbac.PermissionCheck) (*rbac.PermissionCheckResult, error) {
//		return m.checkFunc(ctx, check)
//	}
//
//	// In test
//	mock := &mockChecker{
//		checkFunc: func(ctx context.Context, check rbac.PermissionCheck) (*rbac.PermissionCheckResult, error) {
//			return &rbac.PermissionCheckResult{Allowed: true}, nil
//		},
//	}
//
// For integration tests, use test database with migrations:
//
//	db := setupTestDB()
//	rbac.RunMigrations(db)
//	store := rbac.NewStore(db)
//	checker := rbac.NewPermissionChecker(db, 0) // Disable cache in tests
//
// # Related Packages
//
//   - pkg/auth: User authentication (login, JWT tokens)
//   - pkg/sso: Single sign-on integration (SAML, OAuth)
//   - pkg/orgs: Organization management
//   - pkg/audit: Audit logging of permission checks and role changes
//   - pkg/middleware: HTTP middleware for request authentication
package rbac
