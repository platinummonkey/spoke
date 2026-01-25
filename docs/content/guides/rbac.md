---
title: "RBAC & Permissions"
weight: 10
---

# Role-Based Access Control (RBAC)

Spoke includes a comprehensive RBAC system for fine-grained access control.

## Overview

Spoke's RBAC system provides:

- **Fine-grained permissions** at resource and action level
- **Built-in roles** for common use cases
- **Custom roles** with flexible permission sets
- **Team management** for group-based permissions
- **Permission caching** for performance
- **Audit logging** for all RBAC operations

## Permission Model

### Resources

| Resource | Description |
|----------|-------------|
| `module` | Protobuf modules |
| `version` | Module versions |
| `documentation` | Module documentation |
| `settings` | Organization settings |
| `user` | User management |
| `role` | Role management |
| `team` | Team management |
| `organization` | Organization management |

### Actions

| Action | Description |
|--------|-------------|
| `create` | Create new resources |
| `read` | View resources |
| `update` | Modify resources |
| `delete` | Remove resources |
| `publish` | Publish versions |
| `deprecate` | Deprecate versions |
| `invite` | Invite users |
| `remove` | Remove users/members |
| `update_role` | Change user roles |

### Scopes

| Scope | Description |
|-------|-------------|
| `global` | System-wide permissions |
| `organization` | Organization-level permissions |
| `module` | Module-specific permissions |

## Built-in Roles

### Organization Roles

#### org:admin

Full organization access.

**Permissions:**
- All module operations (create, read, update, delete)
- Version management (publish, read, deprecate)
- Documentation management
- Settings management
- User management (invite, remove, update roles)
- Role and team management

#### org:developer

Development access without administrative privileges.

**Permissions:**
- Create and update modules
- Publish and read versions
- Read and write documentation
- Read organization settings
- Cannot manage users, roles, or teams

#### org:viewer

Read-only access.

**Permissions:**
- Read modules and versions
- Read documentation
- Cannot create, update, or delete resources

### Module Roles

#### module:maintainer

Maintainer of specific modules.

**Permissions:**
- Full access to assigned modules
- Publish and deprecate versions
- Manage module documentation
- Cannot access other modules

#### module:contributor

Contributor to specific modules.

**Permissions:**
- Read and update assigned modules
- Create draft versions (requires approval)
- Cannot publish or deprecate

#### module:reader

Read-only access to specific modules.

**Permissions:**
- Read module contents and versions
- Download proto files
- Cannot modify anything

## Configuration

### Enable RBAC

```yaml
# config.yaml
rbac:
  enabled: true
  cache_ttl: 5m  # Permission cache duration
  audit_enabled: true
```

### Database Schema

RBAC requires PostgreSQL. Run migrations:

```bash
./spoke migrate -db-url "postgres://spoke:password@localhost:5432/spoke?sslmode=disable"
```

## Managing Roles

### Create Custom Role

```http
POST /roles
Authorization: Bearer <token>

{
  "name": "ci-publisher",
  "description": "CI/CD pipeline role for publishing modules",
  "organization_id": "org-123",
  "permissions": [
    {
      "resource": "module",
      "actions": ["read", "update"],
      "scope": "organization"
    },
    {
      "resource": "version",
      "actions": ["create", "publish", "read"],
      "scope": "organization"
    }
  ]
}
```

### List Roles

```http
GET /roles?organization_id=org-123
Authorization: Bearer <token>
```

**Response:**

```json
{
  "roles": [
    {
      "id": "role-123",
      "name": "org:admin",
      "description": "Organization administrator",
      "built_in": true,
      "permission_count": 25
    },
    {
      "id": "role-456",
      "name": "ci-publisher",
      "description": "CI/CD pipeline role",
      "built_in": false,
      "permission_count": 5
    }
  ]
}
```

### Assign Role to User

```http
POST /organizations/{org_id}/members/{user_id}/roles
Authorization: Bearer <token>

{
  "role_id": "role-456"
}
```

## Team Management

### Create Team

```http
POST /organizations/{org_id}/teams
Authorization: Bearer <token>

{
  "name": "Backend Team",
  "description": "Backend engineers"
}
```

### Add Members to Team

```http
POST /organizations/{org_id}/teams/{team_id}/members
Authorization: Bearer <token>

{
  "user_ids": ["user-1", "user-2", "user-3"]
}
```

### Assign Role to Team

```http
POST /organizations/{org_id}/teams/{team_id}/roles
Authorization: Bearer <token>

{
  "role_id": "role-456",
  "scope": "organization"
}
```

All team members inherit the team's permissions.

## Module-Specific Permissions

Grant access to specific modules:

```http
POST /roles
Authorization: Bearer <token>

{
  "name": "user-module-maintainer",
  "organization_id": "org-123",
  "permissions": [
    {
      "resource": "module",
      "actions": ["read", "update", "delete"],
      "scope": "module",
      "resource_id": "module-user"
    },
    {
      "resource": "version",
      "actions": ["create", "publish", "deprecate"],
      "scope": "module",
      "resource_id": "module-user"
    }
  ]
}
```

## Permission Checking

### Check User Permission

```http
POST /rbac/check
Authorization: Bearer <token>

{
  "user_id": "user-123",
  "organization_id": "org-123",
  "resource": "module",
  "action": "create"
}
```

**Response:**

```json
{
  "allowed": true,
  "reason": "User has org:developer role"
}
```

## API Token Permissions

Create API tokens with limited permissions:

```http
POST /api-tokens
Authorization: Bearer <token>

{
  "name": "CI Pipeline Token",
  "organization_id": "org-123",
  "role_id": "role-ci-publisher",
  "expires_at": "2026-01-24T00:00:00Z"
}
```

Use the token:

```bash
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8080/modules
```

## Examples

### CI/CD Role Setup

```bash
# Create CI/CD role
curl -X POST http://localhost:8080/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-publisher",
    "organization_id": "org-123",
    "permissions": [
      {
        "resource": "module",
        "actions": ["read", "update"],
        "scope": "organization"
      },
      {
        "resource": "version",
        "actions": ["create", "publish"],
        "scope": "organization"
      }
    ]
  }'

# Create API token with CI role
curl -X POST http://localhost:8080/api-tokens \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "GitHub Actions",
    "organization_id": "org-123",
    "role_id": "role-ci-publisher",
    "expires_at": "2026-12-31T23:59:59Z"
  }'

# Use in CI/CD
export SPOKE_TOKEN="<generated-token>"
spoke-cli push -module user -version v1.0.0 -dir ./proto
```

### Module-Specific Access

```bash
# Create module maintainer role
curl -X POST http://localhost:8080/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user-module-maintainer",
    "organization_id": "org-123",
    "permissions": [
      {
        "resource": "module",
        "actions": ["read", "update"],
        "scope": "module",
        "resource_id": "module-user"
      },
      {
        "resource": "version",
        "actions": ["create", "publish"],
        "scope": "module",
        "resource_id": "module-user"
      }
    ]
  }'

# Assign to user
curl -X POST http://localhost:8080/organizations/org-123/members/user-456/roles \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "role_id": "role-user-maintainer"
  }'
```

## Audit Logging

All RBAC operations are logged:

```http
GET /audit/logs?resource_type=role&action=assign
Authorization: Bearer <token>
```

**Response:**

```json
{
  "logs": [
    {
      "id": "log-123",
      "timestamp": "2025-01-24T10:00:00Z",
      "user_id": "user-admin",
      "action": "role.assign",
      "resource_type": "role",
      "resource_id": "role-456",
      "metadata": {
        "target_user": "user-789",
        "role_name": "ci-publisher"
      }
    }
  ]
}
```

## Best Practices

### 1. Principle of Least Privilege

Grant minimum necessary permissions:

```json
{
  "name": "readonly-ci",
  "permissions": [
    {
      "resource": "module",
      "actions": ["read"],
      "scope": "organization"
    }
  ]
}
```

### 2. Use Teams for Group Management

Instead of assigning roles individually, use teams:

```bash
# Create team
curl -X POST /organizations/org-123/teams \
  -d '{"name": "Backend Team"}'

# Assign role to team
curl -X POST /organizations/org-123/teams/team-123/roles \
  -d '{"role_id": "role-developer"}'

# Add members to team
curl -X POST /organizations/org-123/teams/team-123/members \
  -d '{"user_ids": ["user-1", "user-2"]}'
```

### 3. Regular Audits

Review permissions regularly:

```bash
# List all role assignments
curl /organizations/org-123/members?include_roles=true

# Check audit logs
curl /audit/logs?organization_id=org-123&days=30
```

### 4. API Token Expiration

Always set expiration for API tokens:

```json
{
  "name": "CI Token",
  "expires_at": "2026-12-31T23:59:59Z"
}
```

### 5. Module-Specific Permissions for Sensitive Data

Use module-scoped permissions for sensitive modules:

```json
{
  "permissions": [
    {
      "resource": "module",
      "actions": ["read"],
      "scope": "module",
      "resource_id": "module-payment"
    }
  ]
}
```

## Troubleshooting

### Permission Denied

Check user's roles and permissions:

```bash
# Get user's roles
curl /organizations/org-123/members/user-456

# Check specific permission
curl -X POST /rbac/check \
  -d '{
    "user_id": "user-456",
    "resource": "module",
    "action": "create"
  }'
```

### Performance Issues

Adjust cache TTL:

```yaml
rbac:
  cache_ttl: 10m  # Increase cache duration
```

## Next Steps

- [SSO Integration](/guides/sso/) - Single Sign-On setup
- [Multi-Tenancy](/guides/multi-tenancy/) - Organization management
- [Audit Logging](/guides/audit/) - Compliance and auditing
