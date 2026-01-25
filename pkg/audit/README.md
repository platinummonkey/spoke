# Audit Logging Package

Comprehensive audit logging system for Spoke that provides security event tracking, data mutation logging, and compliance support.

## Features

- **Multiple Storage Backends**: File-based and PostgreSQL database logging
- **Structured Events**: Well-defined event types for authentication, authorization, data mutations, configuration changes, and admin actions
- **HTTP Middleware**: Automatic logging of HTTP requests with configurable filtering
- **Search and Filter**: Query audit logs by time range, user, event type, resource, and more
- **Export Capabilities**: Export logs in JSON, CSV, or newline-delimited JSON formats
- **Statistics**: Generate audit statistics and reports
- **Retention Policies**: Automatic cleanup of old audit logs with optional archiving
- **SOC2 Compliance**: Designed to support SOC2 compliance requirements

## Installation

```go
import "github.com/platinummonkey/spoke/pkg/audit"
```

## Quick Start

### Basic Setup

```go
package main

import (
    "database/sql"
    "log"
    "net/http"

    "github.com/gorilla/mux"
    "github.com/platinummonkey/spoke/pkg/audit"
)

func main() {
    // Initialize database
    db, err := sql.Open("postgres", "postgres://localhost/spoke?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }

    // Setup audit logging
    config := audit.DefaultIntegrationConfig(db)
    auditMiddleware, auditHandlers, err := audit.SetupAuditLogging(config)
    if err != nil {
        log.Fatal(err)
    }

    // Create router
    router := mux.NewRouter()

    // Register audit API routes
    if auditHandlers != nil {
        auditHandlers.RegisterRoutes(router)
    }

    // Your application routes...
    router.HandleFunc("/api/resource", yourHandler)

    // Wrap with audit middleware
    handler := auditMiddleware.Handler(router)

    // Start server
    http.ListenAndServe(":8080", handler)
}
```

## Event Types

### Authentication Events
- `auth.login` - Successful login
- `auth.logout` - User logout
- `auth.login_failed` - Failed login attempt
- `auth.password_change` - Password changed
- `auth.token_create` - API token created
- `auth.token_revoke` - API token revoked

### Authorization Events
- `authz.permission_check` - Permission check performed
- `authz.permission_grant` - Permission granted to user/org
- `authz.permission_revoke` - Permission revoked
- `authz.role_change` - User role changed
- `authz.access_denied` - Access denied to resource

### Data Mutation Events
- `data.module_create` - Module created
- `data.module_update` - Module updated
- `data.module_delete` - Module deleted
- `data.version_create` - Version published
- `data.version_update` - Version updated
- `data.version_delete` - Version deleted

### Configuration Events
- `config.change` - Configuration setting changed
- `config.sso_update` - SSO configuration updated
- `config.webhook_create` - Webhook created
- `config.webhook_update` - Webhook updated
- `config.webhook_delete` - Webhook deleted

### Admin Events
- `admin.user_create` - User created
- `admin.user_update` - User updated
- `admin.user_delete` - User deleted
- `admin.org_create` - Organization created
- `admin.org_member_add` - Member added to organization
- `admin.org_member_remove` - Member removed from organization

## Usage Examples

### Logging from Application Code

```go
func (h *Handler) createModule(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // ... create module logic ...

    // Log successful creation
    audit.LogSuccess(ctx, audit.EventTypeDataModuleCreate,
        "Module created successfully",
        map[string]interface{}{
            "module_name": module.Name,
            "created_by": userID,
        },
    )

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(module)
}
```

### Logging Authentication

```go
func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    logger := audit.FromContext(ctx)

    // ... authentication logic ...

    if err != nil {
        logger.LogAuthentication(ctx,
            audit.EventTypeAuthLoginFailed,
            nil,
            username,
            audit.EventStatusFailure,
            "Invalid credentials",
        )
        http.Error(w, "authentication failed", http.StatusUnauthorized)
        return
    }

    logger.LogAuthentication(ctx,
        audit.EventTypeAuthLogin,
        &user.ID,
        user.Username,
        audit.EventStatusSuccess,
        "Login successful",
    )

    // ... return token ...
}
```

### Logging Authorization Failures

```go
func (h *Handler) deleteModule(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    if !hasDeletePermission {
        audit.LogDenied(ctx,
            audit.EventTypeAuthzAccessDenied,
            audit.ResourceTypeModule,
            moduleName,
            "User lacks delete permission",
        )
        http.Error(w, "forbidden", http.StatusForbidden)
        return
    }

    // ... delete module ...
}
```

### Logging Data Mutations with Change Tracking

```go
func (h *Handler) updateModule(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    logger := audit.FromContext(ctx)

    // Get original state
    original, _ := h.store.GetModule(moduleName)

    // ... update module ...

    // Log with before/after values
    changes := &audit.ChangeDetails{
        Before: map[string]interface{}{
            "description": original.Description,
            "owner": original.Owner,
        },
        After: map[string]interface{}{
            "description": updated.Description,
            "owner": updated.Owner,
        },
    }

    logger.LogDataMutation(ctx,
        audit.EventTypeDataModuleUpdate,
        &userID,
        audit.ResourceTypeModule,
        moduleName,
        changes,
        "Module updated",
    )
}
```

## API Endpoints

### List Audit Events

```
GET /audit/events?limit=100&offset=0&event_type=auth.login&start_time=2024-01-01T00:00:00Z
```

Query parameters:
- `limit` - Number of results to return (default: 100)
- `offset` - Pagination offset (default: 0)
- `start_time` - Filter events after this time (RFC3339)
- `end_time` - Filter events before this time (RFC3339)
- `user_id` - Filter by user ID
- `username` - Filter by username
- `organization_id` - Filter by organization ID
- `event_type` - Filter by event type (comma-separated)
- `status` - Filter by status (success, failure, denied)
- `resource_type` - Filter by resource type
- `resource_id` - Filter by resource ID
- `ip_address` - Filter by IP address
- `method` - Filter by HTTP method
- `path` - Filter by URL path (partial match)

### Get Specific Event

```
GET /audit/events/{id}
```

### Export Audit Logs

```
GET /audit/export?format=csv&start_time=2024-01-01T00:00:00Z&end_time=2024-01-31T23:59:59Z
```

Formats:
- `json` - JSON array
- `csv` - CSV file
- `ndjson` - Newline-delimited JSON

### Get Statistics

```
GET /audit/stats?start_time=2024-01-01T00:00:00Z&end_time=2024-01-31T23:59:59Z
```

Returns:
- Total events
- Events by type
- Events by status
- Unique users
- Unique IPs
- Failed authentication attempts
- Access denials

## Configuration

### File Logging

```go
config := audit.FileLoggerConfig{
    BasePath: "/var/log/spoke/audit",
    Rotate:   true,
    MaxSize:  100 * 1024 * 1024, // 100MB
    MaxFiles: 10,
}

fileLogger, err := audit.NewFileLogger(config)
```

### Database Logging

```go
dbLogger, err := audit.NewDBLogger(db)
```

### Multi-Logger (Both File and DB)

```go
fileLogger, _ := audit.NewFileLogger(fileConfig)
dbLogger, _ := audit.NewDBLogger(db)

multiLogger := audit.NewMultiLogger(fileLogger, dbLogger)
```

## Retention Policies

```go
policy := audit.RetentionPolicy{
    RetentionDays:   90,
    ArchiveEnabled:  true,
    ArchivePath:     "/var/spoke/audit-archive",
    CompressArchive: true,
}

store := audit.NewDBStore(dbLogger)
deletedCount, err := store.Cleanup(ctx, policy)
```

## SOC2 Compliance

This audit logging system supports SOC2 compliance by:

1. **Logging all security-relevant events** - Authentication, authorization, access denials
2. **Immutable audit trail** - Events are append-only, cannot be modified
3. **Detailed event tracking** - Who, what, when, where for all operations
4. **Data mutation tracking** - Before/after values for all changes
5. **Retention policies** - Configurable retention with archiving support
6. **Search and export** - Easy retrieval of audit logs for review
7. **Access logging** - Track who accessed what resources and when

## Database Schema

The audit logging system creates the following table:

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    user_id BIGINT,
    username VARCHAR(255),
    organization_id BIGINT,
    token_id BIGINT,
    resource_type VARCHAR(50),
    resource_id VARCHAR(255),
    resource_name VARCHAR(255),
    ip_address VARCHAR(45),
    user_agent TEXT,
    request_id VARCHAR(100),
    method VARCHAR(10),
    path TEXT,
    status_code INTEGER,
    message TEXT,
    error_message TEXT,
    metadata JSONB,
    changes JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

See `migrations/004_create_audit_schema.up.sql` for the complete schema with indexes.

## Testing

Run the test suite:

```bash
go test ./pkg/audit/...
```

Run with coverage:

```bash
go test -cover ./pkg/audit/...
```

## Performance Considerations

1. **Asynchronous Logging**: Use `MultiLogger` with async mode for high-throughput scenarios
2. **Index Strategy**: The database schema includes indexes optimized for common query patterns
3. **Partitioning**: For very high volumes, consider PostgreSQL table partitioning by timestamp
4. **Archiving**: Use the retention policy to archive old logs and keep the active table size manageable
5. **Selective Logging**: Configure middleware to only log mutations and sensitive operations (`logAllRequests: false`)

## Best Practices

1. **Always log security events** - Authentication, authorization, permission changes
2. **Log data mutations** - All creates, updates, and deletes
3. **Include context** - User ID, IP address, request ID for correlation
4. **Use structured metadata** - Add relevant details to the metadata field
5. **Track changes** - For updates, include before/after values
6. **Don't log sensitive data** - Avoid logging passwords, tokens, or PII in clear text
7. **Review regularly** - Monitor failed auth attempts and access denials
8. **Export for archival** - Regularly export old logs for long-term storage

## License

See LICENSE.md in the root of the repository.
