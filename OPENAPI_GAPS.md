# OpenAPI Specification Gaps Analysis

## Overview

The existing `openapi.yaml` specification (3,270 lines, 59 endpoints) is comprehensive but missing several API groups that are implemented in the codebase.

## Current Coverage

✅ **Fully Documented:**
- Modules API - Module CRUD operations
- Versions API - Version management
- Compilation API (v1) - Code generation for 15+ languages
- Validation API - Proto validation and normalization
- Compatibility API - Breaking change detection
- Authentication API - Users, tokens, permissions
- Organizations API - Organization management with members
- Billing API - Subscriptions, invoices, payment methods
- Search API (v2) - Advanced search with filters
- User Features API (v2) - Bookmarks and saved searches
- Analytics API (v2) - Usage analytics and health scores
- Plugin Verification API (v1) - Plugin security verification

## Missing APIs

### 1. Dependencies API (HIGH PRIORITY)
**7 endpoints in `pkg/dependencies/handlers.go`:**

```
GET /modules/{name}/versions/{version}/dependencies
GET /modules/{name}/versions/{version}/dependencies/transitive
GET /modules/{name}/versions/{version}/dependents
GET /modules/{name}/versions/{version}/impact
GET /modules/{name}/versions/{version}/lockfile
POST /modules/{name}/versions/{version}/lockfile/validate
GET /modules/{name}/versions/{version}/graph
GET /api/v2/graph/cytoscape (visualization)
```

**Response Types:**
- `Dependency`: module, version, type (direct/transitive)
- `ImpactAnalysis`: direct_dependents, transitive_dependents, total_impact
- `Lockfile`: module, version, dependencies[]
- `DependencyGraph`: nodes[], edges[], has_circular_dependency

### 2. Documentation API (HIGH PRIORITY)
**4 endpoints in `pkg/docs/handlers.go`:**

```
GET /docs/{module}/{version}           - HTML documentation
GET /docs/{module}/{version}/markdown  - Markdown export
GET /docs/{module}/{version}/json      - JSON structured docs
GET /docs/{module}/compare?old={v}&new={v} - Version comparison
```

**Response Types:**
- `Documentation`: messages[], services[], enums[], package, syntax
- `MessageDoc`: name, description, fields[], oneofs[]
- `FieldDoc`: name, type, number, description, deprecated
- `ServiceDoc`: name, description, methods[]

### 3. Basic Search API (MEDIUM PRIORITY)
**5 endpoints in `pkg/search/handlers.go`:**

```
GET /search                   - Full-text search across all entities
GET /search/modules           - Search modules by name/description
GET /search/messages          - Search message definitions
GET /search/fields            - Search field definitions
GET /search/services          - Search service definitions
```

**Query Parameters:**
- q: search query (required)
- limit: results limit (default 50)
- offset: pagination offset

**Response Type:**
- `SearchResults`: results[], total, query, took_ms

### 4. RBAC API (MEDIUM PRIORITY)
**9 endpoints in `pkg/rbac/handlers.go`:**

```
POST   /rbac/roles                      - Create role
GET    /rbac/roles                      - List roles
GET    /rbac/roles/{id}                 - Get role details
PUT    /rbac/roles/{id}                 - Update role
DELETE /rbac/roles/{id}                 - Delete role
POST   /rbac/users/{id}/roles           - Assign role to user
GET    /rbac/users/{id}/roles           - Get user's roles
DELETE /rbac/users/{id}/roles/{role_id} - Revoke role from user
POST   /rbac/check                      - Check permission
GET    /rbac/templates                  - Get role templates
```

**Response Types:**
- `Role`: id, name, description, permissions[], created_at
- `Permission`: resource, action, constraints
- `PermissionCheck`: allowed, reason

### 5. Audit API (MEDIUM PRIORITY)
**4 endpoints in `pkg/audit/handlers.go`:**

```
GET /audit/events           - List audit events
GET /audit/events/{id}      - Get event details
GET /audit/export           - Export events (CSV/JSON)
GET /audit/stats            - Get audit statistics
```

**Response Types:**
- `AuditEvent`: id, actor, action, resource, timestamp, metadata
- `AuditStats`: total_events, events_by_action, events_by_actor

### 6. SSO API (LOW PRIORITY)
**Endpoints in `pkg/sso/handlers.go`:**

```
GET  /sso/providers                  - List SSO providers
POST /sso/providers                  - Configure SSO provider
GET  /sso/providers/{id}             - Get provider config
PUT  /sso/providers/{id}             - Update provider
DELETE /sso/providers/{id}           - Delete provider
GET  /sso/login/{provider}           - Initiate SSO login
GET  /sso/callback/{provider}        - SSO callback handler
POST /sso/link                       - Link SSO account
```

### 7. Marketplace API (LOW PRIORITY)
**Endpoints in `pkg/marketplace/handlers.go`:**

```
GET  /marketplace/plugins            - List marketplace plugins
GET  /marketplace/plugins/{id}       - Get plugin details
POST /marketplace/plugins/{id}/install - Install plugin
GET  /marketplace/categories         - List categories
GET  /marketplace/search             - Search marketplace
```

### 8. Webhooks API (LOW PRIORITY)
**Endpoints in `pkg/webhooks/handlers.go`:**

```
POST   /webhooks                     - Create webhook
GET    /webhooks                     - List webhooks
GET    /webhooks/{id}                - Get webhook
PUT    /webhooks/{id}                - Update webhook
DELETE /webhooks/{id}                - Delete webhook
GET    /webhooks/{id}/deliveries     - List webhook deliveries
POST   /webhooks/{id}/test           - Test webhook
```

### 9. Meta/Swagger API (LOW PRIORITY)
**Endpoints in `pkg/swagger/handlers.go`:**

```
GET /openapi.yaml          - OpenAPI spec (YAML) - EXISTS
GET /openapi.json          - OpenAPI spec (JSON) - NOT IMPLEMENTED
GET /swagger-ui            - Swagger UI HTML
GET /api-docs              - Swagger UI alias
```

## Implementation Priority

### Phase 1: Critical APIs (This PR)
1. ✅ Dependencies API - Core functionality for dependency management
2. ✅ Documentation API - Essential for API consumers
3. ✅ Basic Search API - Overlaps with v2 but different scope

### Phase 2: Enterprise Features
4. RBAC API - Access control
5. Audit API - Compliance and security

### Phase 3: Integration Features
6. SSO API - Enterprise authentication
7. Webhooks API - Event notifications
8. Marketplace API - Plugin ecosystem

## Schema Additions Needed

### New Component Schemas

```yaml
Dependency:
  type: object
  properties:
    module: string
    version: string
    type: string (direct/transitive)

DependencyGraph:
  type: object
  properties:
    nodes: array of GraphNode
    edges: array of GraphEdge
    has_circular_dependency: boolean
    circular_path: array of strings

Documentation:
  type: object
  properties:
    package: string
    syntax: string
    messages: array of MessageDoc
    services: array of ServiceDoc
    enums: array of EnumDoc

AuditEvent:
  type: object
  properties:
    id: string
    actor: string
    action: string
    resource: string
    timestamp: string (date-time)
    metadata: object

Role:
  type: object
  properties:
    id: string
    name: string
    description: string
    permissions: array of Permission
```

## CI/CD Integration

### Validation Steps to Add

1. **Spec Validation**
   ```bash
   npm install -g @apidevtools/swagger-cli
   swagger-cli validate openapi.yaml
   ```

2. **Breaking Change Detection**
   ```bash
   npm install -g oasdiff
   oasdiff breaking openapi.yaml openapi-previous.yaml
   ```

3. **Spec Generation from Code**
   Consider using `swaggo/swag` annotations or `oapi-codegen` to generate from code

4. **Mock Server Testing**
   ```bash
   npm install -g @stoplight/prism-cli
   prism mock openapi.yaml
   ```

## Tools Evaluation

### Option 1: swaggo/swag (Annotations)
**Pros:** Generate from code comments, keeps docs close to code
**Cons:** Requires annotating all handlers, verbose

### Option 2: oapi-codegen (Spec-first)
**Pros:** Strong typing, generates handlers from spec
**Cons:** Reverse of current approach, requires refactoring

### Option 3: go-swagger (Bidirectional)
**Pros:** Can generate from code or from spec
**Cons:** More complex, larger dependency

**Recommendation:** Manual spec maintenance + CI validation (current approach) is working well. Add validation steps to prevent drift.

## Completed Steps

1. [x] Document all gaps
2. [x] Add Dependencies API to openapi.yaml (7 endpoints + 4 schemas)
3. [x] Add Documentation API to openapi.yaml (4 endpoints + 6 schemas)
4. [x] Add Basic Search API to openapi.yaml (5 endpoints + 1 schema)
5. [x] Add RBAC API to openapi.yaml (9 endpoints + 2 schemas)
6. [x] Add Audit API to openapi.yaml (4 endpoints + 1 schema)
7. [x] Implement /openapi.json endpoint (YAML to JSON conversion)
8. [x] Spec validation in CI (Spectral linting)
9. [x] Breaking change detection in CI (oasdiff)
10. [x] Client SDK generation (Go and Python via scripts/generate-sdks.sh)

## Remaining Steps (Lower Priority)

11. [ ] Add SSO API to openapi.yaml (8 endpoints) - when SSO is implemented
12. [ ] Add Webhooks API to openapi.yaml (7 endpoints) - when webhooks are implemented
13. [ ] Add Marketplace API to openapi.yaml (5 endpoints) - when marketplace is implemented

## Conditional Features Documentation

Some endpoints only register if database is available:
- All /auth/* endpoints require database
- All /api/v2/* endpoints require database
- /rbac/* endpoints require database
- /audit/* endpoints require database

This should be documented in the OpenAPI spec using conditional logic or clearly marked in descriptions.
