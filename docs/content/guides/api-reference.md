---
title: "REST API Reference"
weight: 2
---

# REST API Reference

Complete reference for the Spoke HTTP REST API.

## Base URL

```
http://localhost:8080
```

## Authentication

Most endpoints require authentication using JWT tokens.

```http
Authorization: Bearer <token>
```

Get a token via the `/auth/login` endpoint.

## Common Headers

| Header | Description | Required |
|--------|-------------|----------|
| `Authorization` | Bearer token | Yes (most endpoints) |
| `Content-Type` | `application/json` | Yes (POST/PUT) |
| `X-Organization-ID` | Organization ID (multi-tenancy) | No |

## Response Format

All responses return JSON:

```json
{
  "data": { /* response data */ },
  "error": null
}
```

Error responses:

```json
{
  "data": null,
  "error": {
    "code": "MODULE_NOT_FOUND",
    "message": "Module 'user' not found"
  }
}
```

## Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 500 | Internal Server Error |

---

## Modules

### Create Module

Create a new protobuf module.

```http
POST /modules
```

**Request Body:**

```json
{
  "name": "user",
  "description": "User service protobuf definitions"
}
```

**Response:**

```json
{
  "name": "user",
  "description": "User service protobuf definitions",
  "created_at": "2025-01-24T10:00:00Z",
  "versions": []
}
```

---

### List Modules

List all modules.

```http
GET /modules
```

**Query Parameters:**

| Parameter | Type | Description | Default |
|-----------|------|-------------|---------|
| `page` | int | Page number | 1 |
| `page_size` | int | Items per page | 20 |
| `org` | string | Organization ID | "" |

**Response:**

```json
{
  "modules": [
    {
      "name": "user",
      "description": "User service",
      "created_at": "2025-01-24T10:00:00Z",
      "version_count": 3
    },
    {
      "name": "order",
      "description": "Order service",
      "created_at": "2025-01-24T11:00:00Z",
      "version_count": 2
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_pages": 1,
    "total_items": 2
  }
}
```

---

### Get Module

Get details for a specific module.

```http
GET /modules/{name}
```

**Response:**

```json
{
  "name": "user",
  "description": "User service protobuf definitions",
  "created_at": "2025-01-24T10:00:00Z",
  "updated_at": "2025-01-24T12:00:00Z",
  "versions": ["v1.0.0", "v1.1.0", "v2.0.0"],
  "latest_version": "v2.0.0"
}
```

---

## Versions

### Create Version

Create a new version of a module.

```http
POST /modules/{name}/versions
```

**Request Body (multipart/form-data):**

```
version: v1.0.0
description: Initial release
file1: user.proto (file upload)
file2: types.proto (file upload)
```

**Response:**

```json
{
  "module": "user",
  "version": "v1.0.0",
  "description": "Initial release",
  "created_at": "2025-01-24T10:00:00Z",
  "files": [
    "user.proto",
    "types.proto"
  ],
  "dependencies": []
}
```

---

### List Versions

List all versions of a module.

```http
GET /modules/{name}/versions
```

**Response:**

```json
{
  "module": "user",
  "versions": [
    {
      "version": "v2.0.0",
      "description": "Major update",
      "created_at": "2025-01-24T14:00:00Z",
      "file_count": 3
    },
    {
      "version": "v1.1.0",
      "description": "Added phone field",
      "created_at": "2025-01-24T12:00:00Z",
      "file_count": 3
    },
    {
      "version": "v1.0.0",
      "description": "Initial release",
      "created_at": "2025-01-24T10:00:00Z",
      "file_count": 2
    }
  ]
}
```

---

### Get Version

Get details for a specific version.

```http
GET /modules/{name}/versions/{version}
```

**Response:**

```json
{
  "module": "user",
  "version": "v1.0.0",
  "description": "Initial release",
  "created_at": "2025-01-24T10:00:00Z",
  "files": [
    {
      "path": "user.proto",
      "size": 1024,
      "checksum": "sha256:abc123..."
    },
    {
      "path": "types.proto",
      "size": 512,
      "checksum": "sha256:def456..."
    }
  ],
  "dependencies": [
    {
      "module": "common",
      "version": "v1.0.0"
    }
  ],
  "compiled_languages": ["go", "python"]
}
```

---

## Files

### Get File

Download a specific proto file.

```http
GET /modules/{name}/versions/{version}/files/{path}
```

**Example:**

```http
GET /modules/user/versions/v1.0.0/files/user.proto
```

**Response:**

```proto
syntax = "proto3";
package user.v1;

message User {
  string id = 1;
  string email = 2;
}
```

---

### Download Compiled Files

Download pre-compiled code for a specific language.

```http
GET /modules/{name}/versions/{version}/download/{language}
```

**Supported Languages:**
- `go`
- `python`
- `java`
- `cpp`
- `csharp`

**Example:**

```http
GET /modules/user/versions/v1.0.0/download/go
```

**Response:**

Returns a ZIP file containing compiled protobuf files.

---

## Compatibility

### Check Compatibility

Check if two versions are compatible.

```http
POST /modules/{name}/compatibility/check
```

**Request Body:**

```json
{
  "version1": "v1.0.0",
  "version2": "v1.1.0"
}
```

**Response:**

```json
{
  "compatible": true,
  "breaking_changes": [],
  "warnings": [
    "Field 'phone_number' added to User message"
  ]
}
```

---

## Validation

### Validate Proto Files

Validate protobuf files before pushing.

```http
POST /validate
```

**Request Body (multipart/form-data):**

```
file1: user.proto (file upload)
file2: types.proto (file upload)
```

**Response:**

```json
{
  "valid": true,
  "errors": [],
  "warnings": []
}
```

---

## Authentication

### Login

Authenticate and get a JWT token.

```http
POST /auth/login
```

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2025-01-25T10:00:00Z",
  "user": {
    "id": "user-123",
    "email": "user@example.com",
    "role": "developer"
  }
}
```

---

### Register

Create a new user account.

```http
POST /auth/register
```

**Request Body:**

```json
{
  "email": "user@example.com",
  "password": "secret123",
  "first_name": "John",
  "last_name": "Doe"
}
```

**Response:**

```json
{
  "user": {
    "id": "user-123",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  }
}
```

---

### Logout

Invalidate current token.

```http
POST /auth/logout
```

**Headers:**

```
Authorization: Bearer <token>
```

**Response:**

```json
{
  "success": true
}
```

---

## Organizations (Multi-Tenancy)

### Create Organization

```http
POST /organizations
```

**Request Body:**

```json
{
  "name": "Acme Corp",
  "slug": "acme"
}
```

**Response:**

```json
{
  "id": "org-123",
  "name": "Acme Corp",
  "slug": "acme",
  "plan": "free",
  "created_at": "2025-01-24T10:00:00Z"
}
```

---

### List Organizations

```http
GET /organizations
```

**Response:**

```json
{
  "organizations": [
    {
      "id": "org-123",
      "name": "Acme Corp",
      "slug": "acme",
      "plan": "professional",
      "member_count": 5
    }
  ]
}
```

---

## Health & Metrics

### Health Check

```http
GET /health
```

**Response:**

```json
{
  "status": "ok",
  "timestamp": "2025-01-24T10:00:00Z",
  "version": "1.0.0",
  "uptime": "48h30m15s"
}
```

---

### Metrics

```http
GET /metrics
```

**Response:**

Prometheus-formatted metrics.

```
# HELP spoke_modules_total Total number of modules
# TYPE spoke_modules_total gauge
spoke_modules_total 42

# HELP spoke_versions_total Total number of versions
# TYPE spoke_versions_total gauge
spoke_versions_total 156
```

---

## Webhooks

### Register Webhook

```http
POST /webhooks
```

**Request Body:**

```json
{
  "url": "https://example.com/webhook",
  "events": ["module.created", "version.published"],
  "secret": "webhook-secret"
}
```

**Response:**

```json
{
  "id": "webhook-123",
  "url": "https://example.com/webhook",
  "events": ["module.created", "version.published"],
  "created_at": "2025-01-24T10:00:00Z"
}
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `MODULE_NOT_FOUND` | Module does not exist |
| `VERSION_NOT_FOUND` | Version does not exist |
| `FILE_NOT_FOUND` | File does not exist |
| `INVALID_VERSION` | Invalid version format |
| `VERSION_EXISTS` | Version already exists |
| `COMPILATION_FAILED` | Protobuf compilation failed |
| `UNAUTHORIZED` | Authentication required |
| `FORBIDDEN` | Insufficient permissions |
| `QUOTA_EXCEEDED` | Organization quota exceeded |

---

## Rate Limiting

API requests are rate-limited per organization:

| Plan | Requests per Hour |
|------|-------------------|
| Free | 100 |
| Professional | 1,000 |
| Enterprise | 10,000 |

Rate limit headers:

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 995
X-RateLimit-Reset: 1643040000
```

---

## Examples

### cURL Examples

#### Push a Module

```bash
# Create module
curl -X POST http://localhost:8080/modules \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user",
    "description": "User service"
  }'

# Upload version with files
curl -X POST http://localhost:8080/modules/user/versions \
  -H "Authorization: Bearer $TOKEN" \
  -F "version=v1.0.0" \
  -F "description=Initial release" \
  -F "file1=@proto/user.proto" \
  -F "file2=@proto/types.proto"
```

#### Pull a Module

```bash
# Get version details
curl http://localhost:8080/modules/user/versions/v1.0.0

# Download proto file
curl http://localhost:8080/modules/user/versions/v1.0.0/files/user.proto \
  -o user.proto

# Download compiled Go code
curl http://localhost:8080/modules/user/versions/v1.0.0/download/go \
  -o go-code.zip
```

### Python Example

```python
import requests

BASE_URL = "http://localhost:8080"
TOKEN = "your-token"

headers = {
    "Authorization": f"Bearer {TOKEN}",
    "Content-Type": "application/json"
}

# List modules
response = requests.get(f"{BASE_URL}/modules", headers=headers)
modules = response.json()

print(f"Found {len(modules['modules'])} modules")

# Get specific version
response = requests.get(
    f"{BASE_URL}/modules/user/versions/v1.0.0",
    headers=headers
)
version = response.json()
print(f"Version has {len(version['files'])} files")
```

### Go Example

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

const baseURL = "http://localhost:8080"
const token = "your-token"

func listModules() error {
    req, _ := http.NewRequest("GET", baseURL+"/modules", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    fmt.Printf("Modules: %+v\n", result)
    return nil
}
```

---

## Next Steps

- [CLI Reference](/guides/cli-reference/) - Command-line tool
- [Webhooks](/guides/webhooks/) - Webhook integration
- [Authentication](/guides/sso/) - SSO configuration
