# RBAC (Role-Based Access Control)

Advanced role-based access control system for Spoke with fine-grained permissions, team management, role inheritance, and permission caching.

## Features

- **Fine-Grained Permissions**: Resource-level permissions (module, version, documentation, etc.)
- **Role Management**: Built-in and custom roles with inheritance
- **Team Support**: Organize users into teams with shared permissions
- **Permission Caching**: High-performance permission checks with caching
- **Scope Control**: Organization, module, and global-level permissions
- **Audit Integration**: Full audit logging for all RBAC operations
- **Permission Templates**: Pre-defined role templates for common use cases

## Architecture

### Core Components

1. **Types** (`types.go`): Core RBAC data structures
   - Resources, Actions, Permissions
   - Roles, UserRoles, TeamRoles
   - Permission scopes

2. **Store** (`store.go`): Database persistence layer
   - Role CRUD operations
   - User/Team role assignments
   - Permission queries

3. **Checker** (`checker.go`): Permission evaluation engine
   - Permission checking with caching
   - Role inheritance resolution
   - Team-based permissions

4. **Handlers** (`handlers.go`): HTTP API endpoints
   - Role management
   - User/Team role assignments
   - Permission checking

5. **Middleware** (`middleware.go`): HTTP middleware
   - Permission-based request filtering
   - Role-based access control

## Built-in Roles

### Organization-Level Roles

- **org:admin**: Full organization access
  - All module operations
  - User management
  - Settings management
  - Role management

- **org:developer**: Development access
  - Create/update modules
  - Publish versions
  - Update documentation

- **org:viewer**: Read-only access
  - View modules and versions
  - View documentation

### Module-Level Roles

- **module:owner**: Full module access
  - All operations on specific module
  - Manage module permissions

- **module:contributor**: Contributor access
  - Publish versions to module
  - View module details

- **module:viewer**: Read-only module access
  - View module and versions

## Usage

### Database Setup

```go
import "github.com/platinummonkey/spoke/pkg/rbac"

// Run migrations
ctx := context.Background()
if err := rbac.RunMigrations(ctx, db); err != nil {
    log.Fatal(err)
}

// Initialize built-in roles
store := rbac.NewStore(db)
if err := rbac.InitializeBuiltInRoles(ctx, store); err != nil {
    log.Fatal(err)
}
```

### Creating Custom Roles

```go
// Create a custom role
role := &rbac.Role{
    Name:        "ci-bot",
    DisplayName: "CI/CD Bot",
    Description: "Automated deployment bot",
    Permissions: []rbac.Permission{
        {Resource: rbac.ResourceModule, Action: rbac.ActionRead},
        {Resource: rbac.ResourceVersion, Action: rbac.ActionPublish},
    },
    IsCustom: true,
}

if err := store.CreateRole(ctx, role); err != nil {
    log.Fatal(err)
}
```

### Assigning Roles to Users

```go
// Assign organization-level role
userRole := &rbac.UserRole{
    UserID:         userID,
    RoleID:         roleID,
    Scope:          rbac.ScopeOrganization,
    OrganizationID: &orgID,
}

if err := store.AssignRoleToUser(ctx, userRole); err != nil {
    log.Fatal(err)
}

// Assign module-specific role
moduleName := "my-module"
moduleRole := &rbac.UserRole{
    UserID:     userID,
    RoleID:     moduleOwnerRoleID,
    Scope:      rbac.ScopeModule,
    ResourceID: &moduleName,
}

if err := store.AssignRoleToUser(ctx, moduleRole); err != nil {
    log.Fatal(err)
}
```

### Checking Permissions

```go
checker := rbac.NewPermissionChecker(db, 5*time.Minute)

// Check if user can create a module
check := rbac.PermissionCheck{
    UserID: userID,
    Permission: rbac.Permission{
        Resource: rbac.ResourceModule,
        Action:   rbac.ActionCreate,
    },
    Scope:          rbac.ScopeOrganization,
    OrganizationID: &orgID,
}

result, err := checker.CheckPermission(ctx, check)
if err != nil {
    log.Fatal(err)
}

if result.Allowed {
    fmt.Println("Permission granted:", result.Reason)
} else {
    fmt.Println("Permission denied:", result.Reason)
}
```

### Using Middleware

```go
import (
    "github.com/gorilla/mux"
    "github.com/platinummonkey/spoke/pkg/rbac"
)

router := mux.NewRouter()
checker := rbac.NewPermissionChecker(db, 5*time.Minute)
permMiddleware := rbac.NewPermissionMiddleware(checker)

// Require organization admin role
router.Handle("/admin/settings",
    permMiddleware.RequireRole(rbac.RoleOrgAdmin)(handler),
).Methods("GET")

// Require specific permission
router.Handle("/modules",
    permMiddleware.RequirePermission(
        rbac.ResourceModule,
        rbac.ActionCreate,
        rbac.ScopeOrganization,
    )(createModuleHandler),
).Methods("POST")

// Require module-specific permission
router.Handle("/modules/{name}/versions",
    permMiddleware.RequireModulePermission(rbac.ActionPublish)(publishHandler),
).Methods("POST")
```

### Team Management

```go
// Create a team
team := &rbac.Team{
    OrganizationID: orgID,
    Name:           "backend-team",
    DisplayName:    "Backend Engineering Team",
    Description:    "Backend development team",
}

if err := store.CreateTeam(ctx, team); err != nil {
    log.Fatal(err)
}

// Add members to team
member := &rbac.TeamMember{
    TeamID: team.ID,
    UserID: userID,
}

if err := store.AddTeamMember(ctx, member); err != nil {
    log.Fatal(err)
}

// Assign role to team (all members inherit)
teamRole := &rbac.TeamRole{
    TeamID:         team.ID,
    RoleID:         developerRoleID,
    Scope:          rbac.ScopeOrganization,
    OrganizationID: &orgID,
}

if err := store.AssignRoleToTeam(ctx, teamRole); err != nil {
    log.Fatal(err)
}
```

### Role Inheritance

```go
// Create parent role
parentRole := &rbac.Role{
    Name:        "base-developer",
    DisplayName: "Base Developer",
    Permissions: []rbac.Permission{
        {Resource: rbac.ResourceModule, Action: rbac.ActionRead},
    },
}

store.CreateRole(ctx, parentRole)

// Create child role that inherits from parent
childRole := &rbac.Role{
    Name:         "senior-developer",
    DisplayName:  "Senior Developer",
    ParentRoleID: &parentRole.ID,
    Permissions: []rbac.Permission{
        {Resource: rbac.ResourceModule, Action: rbac.ActionCreate},
        {Resource: rbac.ResourceVersion, Action: rbac.ActionPublish},
    },
}

store.CreateRole(ctx, childRole)

// Users with "senior-developer" role will have:
// - module:read (from parent)
// - module:create (from child)
// - version:publish (from child)
```

## API Endpoints

### Role Management

- `POST /rbac/roles` - Create custom role
- `GET /rbac/roles` - List all roles
- `GET /rbac/roles/{id}` - Get role details
- `PUT /rbac/roles/{id}` - Update role
- `DELETE /rbac/roles/{id}` - Delete role

### User Role Assignment

- `POST /rbac/users/{id}/roles` - Assign role to user
- `GET /rbac/users/{id}/roles` - Get user's roles
- `DELETE /rbac/users/{id}/roles/{role_id}` - Revoke role from user
- `GET /rbac/users/{id}/permissions` - Get effective permissions

### Permission Checking

- `POST /rbac/check` - Check if user has permission

### Team Management

- `POST /rbac/teams` - Create team
- `GET /rbac/teams` - List teams
- `GET /rbac/teams/{id}` - Get team details
- `PUT /rbac/teams/{id}` - Update team
- `DELETE /rbac/teams/{id}` - Delete team

### Team Members

- `POST /rbac/teams/{id}/members` - Add team member
- `GET /rbac/teams/{id}/members` - Get team members
- `DELETE /rbac/teams/{id}/members/{user_id}` - Remove team member

### Team Roles

- `POST /rbac/teams/{id}/roles` - Assign role to team
- `DELETE /rbac/teams/{id}/roles/{role_id}` - Revoke role from team

### Role Templates

- `GET /rbac/templates` - Get common role templates

## Database Schema

### Tables

- `roles`: Role definitions with permissions
- `user_roles`: Role assignments to users
- `teams`: Team definitions
- `team_members`: Team membership
- `team_roles`: Role assignments to teams
- `permission_cache`: Cached permission check results

See `migrations.go` for complete schema.

## Performance

### Permission Caching

- Permission checks are cached for 5 minutes by default
- Cache is automatically invalidated when roles change
- Reduces database queries for frequent permission checks

### Cache Invalidation

```go
// Invalidate cache for a user
checker.InvalidateCache(ctx, userID)

// Cache is automatically invalidated when:
// - User roles are assigned/revoked
// - Team roles are changed
// - User's team membership changes
```

## Testing

```bash
# Run RBAC tests
go test ./pkg/rbac/...

# Run with coverage
go test ./pkg/rbac/... -cover

# Run specific test
go test ./pkg/rbac/ -run TestPermissionChecker_CheckPermission
```

## Integration with Audit Logging

All RBAC operations are automatically logged for audit purposes:

- Role creation/modification/deletion
- Permission grants/revocations
- Team management operations
- Permission check failures

Audit events include:
- User who performed the action
- Resource affected
- Timestamp
- Success/failure status

## Security Considerations

1. **Built-in Role Protection**: Built-in roles cannot be modified or deleted
2. **Permission Validation**: All permissions are validated before assignment
3. **Scope Enforcement**: Permissions are enforced at the appropriate scope
4. **Audit Trail**: All operations are logged for security review
5. **Token Scopes**: API tokens can have RBAC permissions for automation

## Best Practices

1. **Use Built-in Roles**: Start with built-in roles before creating custom ones
2. **Team-Based Access**: Use teams for group permissions instead of individual assignments
3. **Least Privilege**: Grant minimal permissions required for each role
4. **Role Inheritance**: Use role inheritance to build role hierarchies
5. **Regular Audits**: Review permission assignments periodically
6. **Temporary Access**: Use `ExpiresAt` for time-limited permissions

## Examples

See `checker_test.go` and `store_test.go` for comprehensive usage examples.
