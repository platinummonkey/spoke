# Plugin Marketplace API Documentation

This document describes the REST API endpoints for the Spoke Plugin Marketplace.

## Base URL

```
https://spoke.example.com/api/v1
```

## Authentication

Most endpoints are public (read-only). Write operations require authentication via:
- Header: `Authorization: Bearer <token>`
- Header: `X-User-ID: <user_id>` (for development)

## Endpoints

### Plugin Discovery

#### List Plugins

List all available plugins with optional filters.

**Endpoint:** `GET /plugins`

**Query Parameters:**
- `type` (string, optional) - Filter by plugin type: `language`, `validator`, `generator`, `runner`, `transform`
- `security_level` (string, optional) - Filter by security level: `official`, `verified`, `community`
- `q` (string, optional) - Search query (searches name and description)
- `tags` (string[], optional) - Filter by tags
- `sort_by` (string, optional) - Sort field: `downloads`, `rating`, `created_at` (default: `created_at`)
- `sort_order` (string, optional) - Sort order: `asc`, `desc` (default: `desc`)
- `limit` (integer, optional) - Results per page (default: 20, max: 100)
- `offset` (integer, optional) - Pagination offset (default: 0)

**Response:** `200 OK`

```json
{
  "plugins": [
    {
      "id": "rust-language",
      "name": "Rust Language Plugin",
      "description": "Protocol Buffers code generation for Rust",
      "author": "Spoke Community",
      "license": "MIT",
      "homepage": "https://plugins.spoke.dev/rust",
      "repository": "https://github.com/spoke-plugins/rust-language",
      "type": "language",
      "security_level": "verified",
      "enabled": true,
      "created_at": "2026-01-20T10:00:00Z",
      "updated_at": "2026-01-25T15:30:00Z",
      "download_count": 1523,
      "latest_version": "1.2.0",
      "avg_rating": 4.7,
      "review_count": 23
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins?type=language&limit=10"
```

---

#### Search Plugins

Search plugins by keyword.

**Endpoint:** `GET /plugins/search`

**Query Parameters:**
- `q` (string, required) - Search query
- Same pagination parameters as List Plugins

**Response:** Same as List Plugins

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/search?q=rust"
```

---

#### Get Trending Plugins

Get trending plugins based on recent download growth.

**Endpoint:** `GET /plugins/trending`

**Query Parameters:**
- `limit` (integer, optional) - Number of results (default: 10)

**Response:** `200 OK`

```json
{
  "plugins": [
    {
      "id": "buf-connect-go",
      "name": "Buf Connect for Go",
      "growth_rate": 2.5,
      "weekly_downloads": 450,
      // ... other plugin fields
    }
  ]
}
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/trending"
```

---

#### Get Plugin Details

Get detailed information about a specific plugin.

**Endpoint:** `GET /plugins/{id}`

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Response:** `200 OK`

```json
{
  "id": "rust-language",
  "name": "Rust Language Plugin",
  "description": "Protocol Buffers code generation for Rust using prost",
  "author": "Spoke Community",
  "license": "MIT",
  "homepage": "https://plugins.spoke.dev/rust",
  "repository": "https://github.com/spoke-plugins/rust-language",
  "type": "language",
  "security_level": "verified",
  "enabled": true,
  "created_at": "2026-01-20T10:00:00Z",
  "updated_at": "2026-01-25T15:30:00Z",
  "verified_at": "2026-01-21T14:00:00Z",
  "verified_by": "spoke-admin",
  "download_count": 1523,
  "latest_version": "1.2.0",
  "avg_rating": 4.7,
  "review_count": 23
}
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/rust-language"
```

---

### Plugin Versions

#### List Plugin Versions

List all versions of a plugin.

**Endpoint:** `GET /plugins/{id}/versions`

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Response:** `200 OK`

```json
[
  {
    "id": 123,
    "plugin_id": "rust-language",
    "version": "1.2.0",
    "api_version": "1.0.0",
    "manifest_url": "https://storage.spoke.dev/plugins/rust-language/1.2.0/plugin.yaml",
    "download_url": "https://storage.spoke.dev/plugins/rust-language/1.2.0/rust-language-1.2.0.tar.gz",
    "checksum": "sha256:abc123...",
    "size_bytes": 2048576,
    "downloads": 523,
    "created_at": "2026-01-25T10:00:00Z"
  },
  {
    "id": 122,
    "plugin_id": "rust-language",
    "version": "1.1.0",
    // ...
  }
]
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/rust-language/versions"
```

---

#### Get Specific Version

Get details for a specific plugin version.

**Endpoint:** `GET /plugins/{id}/versions/{version}`

**Path Parameters:**
- `id` (string, required) - Plugin ID
- `version` (string, required) - Version string (e.g., "1.2.0")

**Response:** `200 OK`

```json
{
  "id": 123,
  "plugin_id": "rust-language",
  "version": "1.2.0",
  "api_version": "1.0.0",
  "manifest_url": "https://storage.spoke.dev/plugins/rust-language/1.2.0/plugin.yaml",
  "download_url": "https://storage.spoke.dev/plugins/rust-language/1.2.0/rust-language-1.2.0.tar.gz",
  "checksum": "sha256:abc123...",
  "size_bytes": 2048576,
  "downloads": 523,
  "created_at": "2026-01-25T10:00:00Z"
}
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/rust-language/versions/1.2.0"
```

---

#### Download Plugin

Download a specific plugin version.

**Endpoint:** `GET /plugins/{id}/versions/{version}/download`

**Path Parameters:**
- `id` (string, required) - Plugin ID
- `version` (string, required) - Version string

**Response:** `302 Found` - Redirects to download URL

**Headers:**
- `Location` - Download URL

**Example:**
```bash
curl -L "https://spoke.example.com/api/v1/plugins/rust-language/versions/1.2.0/download" -o rust-language.tar.gz
```

---

### Plugin Reviews

#### List Reviews

List reviews for a plugin.

**Endpoint:** `GET /plugins/{id}/reviews`

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Query Parameters:**
- `limit` (integer, optional) - Results per page (default: 20)
- `offset` (integer, optional) - Pagination offset (default: 0)

**Response:** `200 OK`

```json
[
  {
    "id": 456,
    "plugin_id": "rust-language",
    "user_id": "user123",
    "user_name": "John Doe",
    "rating": 5,
    "review": "Excellent plugin! Works flawlessly with our project.",
    "created_at": "2026-01-24T12:00:00Z",
    "updated_at": "2026-01-24T12:00:00Z"
  }
]
```

**Example:**
```bash
curl "https://spoke.example.com/api/v1/plugins/rust-language/reviews"
```

---

#### Create or Update Review

Submit a review for a plugin.

**Endpoint:** `POST /plugins/{id}/reviews`

**Authentication:** Required

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Request Body:**

```json
{
  "rating": 5,
  "review": "Excellent plugin! Works great."
}
```

**Response:** `201 Created`

```json
{
  "status": "success"
}
```

**Example:**
```bash
curl -X POST "https://spoke.example.com/api/v1/plugins/rust-language/reviews" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "review": "Great plugin!"}'
```

---

### Installation Tracking

#### Record Installation

Record that a plugin was installed.

**Endpoint:** `POST /plugins/{id}/install`

**Authentication:** Optional

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Request Body:**

```json
{
  "version": "1.2.0",
  "organization_id": "org-123"
}
```

**Response:** `201 Created`

```json
{
  "status": "success"
}
```

**Example:**
```bash
curl -X POST "https://spoke.example.com/api/v1/plugins/rust-language/install" \
  -H "Content-Type: application/json" \
  -d '{"version": "1.2.0"}'
```

---

#### Record Uninstallation

Record that a plugin was uninstalled.

**Endpoint:** `POST /plugins/{id}/uninstall`

**Authentication:** Optional

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Response:** `200 OK`

```json
{
  "status": "success"
}
```

**Example:**
```bash
curl -X POST "https://spoke.example.com/api/v1/plugins/rust-language/uninstall" \
  -H "Content-Type: application/json"
```

---

### Plugin Submission

#### Submit New Plugin

Submit a new plugin to the marketplace.

**Endpoint:** `POST /plugins`

**Authentication:** Required

**Request Body:**

```json
{
  "id": "my-plugin",
  "name": "My Plugin",
  "description": "A great plugin for Spoke",
  "author": "Your Name",
  "license": "MIT",
  "homepage": "https://example.com",
  "repository": "https://github.com/you/my-plugin",
  "type": "language",
  "version": "1.0.0",
  "api_version": "1.0.0",
  "archive_data": "base64_encoded_tar_gz",
  "tags": ["rust", "protobuf"]
}
```

**Response:** `201 Created`

```json
{
  "status": "success",
  "plugin_id": "my-plugin",
  "message": "Plugin submitted for review"
}
```

**Example:**
```bash
curl -X POST "https://spoke.example.com/api/v1/plugins" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d @plugin-submission.json
```

---

#### Submit Plugin Version

Submit a new version of an existing plugin.

**Endpoint:** `POST /plugins/{id}/versions`

**Authentication:** Required

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Request Body:**

```json
{
  "version": "1.1.0",
  "api_version": "1.0.0",
  "archive_data": "base64_encoded_tar_gz",
  "changelog": "Fixed bugs, added features"
}
```

**Response:** `201 Created`

```json
{
  "status": "success",
  "version": "1.1.0"
}
```

---

### Plugin Statistics

#### Get Plugin Statistics

Get aggregated statistics for a plugin.

**Endpoint:** `GET /plugins/{id}/stats`

**Path Parameters:**
- `id` (string, required) - Plugin ID

**Query Parameters:**
- `days` (integer, optional) - Number of days of history (default: 30)

**Response:** `200 OK`

```json
{
  "plugin_id": "rust-language",
  "total_downloads": 1523,
  "total_installations": 456,
  "active_installations": 420,
  "avg_rating": 4.7,
  "review_count": 23,
  "daily_stats": [
    {
      "date": "2026-01-25",
      "downloads": 45,
      "installations": 12,
      "uninstallations": 1,
      "active_installations": 420
    }
  ],
  "top_versions": [
    {
      "version": "1.2.0",
      "downloads": 523
    },
    {
      "version": "1.1.0",
      "downloads": 450
    }
  ]
}
```

---

## Error Responses

All endpoints return standard HTTP status codes and error responses:

**400 Bad Request:**
```json
{
  "error": "invalid request",
  "message": "rating must be between 1 and 5"
}
```

**401 Unauthorized:**
```json
{
  "error": "unauthorized",
  "message": "authentication required"
}
```

**404 Not Found:**
```json
{
  "error": "not found",
  "message": "plugin not found: unknown-plugin"
}
```

**500 Internal Server Error:**
```json
{
  "error": "internal server error",
  "message": "failed to query database"
}
```

---

## Rate Limiting

- **Anonymous:** 100 requests per hour per IP
- **Authenticated:** 1000 requests per hour per user

Rate limit headers:
- `X-RateLimit-Limit` - Request limit
- `X-RateLimit-Remaining` - Remaining requests
- `X-RateLimit-Reset` - Unix timestamp when limit resets

---

## Pagination

List endpoints support pagination via `limit` and `offset` parameters:

```bash
# Get first page
curl "https://spoke.example.com/api/v1/plugins?limit=20&offset=0"

# Get second page
curl "https://spoke.example.com/api/v1/plugins?limit=20&offset=20"
```

Response includes pagination metadata:
```json
{
  "plugins": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

---

## Webhooks

Subscribe to plugin events (Coming soon):

- `plugin.published` - New plugin published
- `plugin.updated` - Plugin updated
- `plugin.version.released` - New version released
- `plugin.review.created` - New review posted

---

## SDK Support

Official SDKs available:
- **Go:** `import "github.com/platinummonkey/spoke/pkg/marketplace"`
- **Python:** `pip install spoke-marketplace`
- **JavaScript:** `npm install @spoke/marketplace`
- **Rust:** `cargo add spoke-marketplace`

Example (Go):
```go
import "github.com/platinummonkey/spoke/pkg/marketplace"

client := marketplace.NewClient("https://spoke.example.com")
plugins, err := client.ListPlugins(ctx, &marketplace.PluginListRequest{
    Type: "language",
    Limit: 10,
})
```

---

## See Also

- [Plugin Development Guide](PLUGIN_DEVELOPMENT.md)
- [Plugin Manifest Specification](PLUGIN_MANIFEST.md)
- [Authentication Guide](AUTHENTICATION.md)
