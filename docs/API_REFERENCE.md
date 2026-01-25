# Spoke API Reference

Complete API reference for Spoke's RESTful HTTP API.

## Base URL

```
http://localhost:8080
```

## Authentication

Currently, the API does not require authentication. Future versions may add API key or OAuth support.

## API Versions

- **v1** (Current): `/api/v1/...`
- **Legacy**: Direct endpoints without version prefix

## Endpoints

### Languages

#### List All Languages

Returns a list of all supported programming languages.

```http
GET /api/v1/languages
```

**Response:** `200 OK`

```json
[
  {
    "id": "go",
    "name": "Go",
    "display_name": "Go (Protocol Buffers)",
    "supports_grpc": true,
    "file_extensions": [".pb.go"],
    "enabled": true,
    "stable": true,
    "description": "Go language support with protoc-gen-go",
    "documentation_url": "https://protobuf.dev/reference/go/go-generated/",
    "plugin_version": "v1.31.0",
    "package_manager": {
      "name": "go-modules",
      "config_files": ["go.mod"]
    }
  }
]
```

**Fields:**
- `id` (string): Unique language identifier
- `name` (string): Human-readable language name
- `display_name` (string): Full display name with context
- `supports_grpc` (boolean): Whether gRPC is supported
- `file_extensions` (string[]): Generated file extensions
- `enabled` (boolean): Whether language is currently enabled
- `stable` (boolean): Whether language support is production-ready
- `description` (string): Brief description of language support
- `documentation_url` (string): Link to official documentation
- `plugin_version` (string): Version of the protoc plugin
- `package_manager` (object): Package manager information (optional)

#### Get Language Details

Returns detailed information about a specific language.

```http
GET /api/v1/languages/{id}
```

**Parameters:**
- `id` (path): Language ID (e.g., `go`, `python`, `rust`)

**Response:** `200 OK`

```json
{
  "id": "rust",
  "name": "Rust",
  "display_name": "Rust (Protocol Buffers)",
  "supports_grpc": true,
  "file_extensions": [".rs"],
  "enabled": true,
  "stable": true,
  "description": "Rust language support with prost and tonic",
  "documentation_url": "https://github.com/tokio-rs/prost",
  "plugin_version": "3.2.0",
  "package_manager": {
    "name": "cargo",
    "config_files": ["Cargo.toml"]
  }
}
```

**Errors:**
- `404 Not Found`: Language not found

### Compilation

#### Trigger Compilation

Triggers compilation of a module version for one or more languages.

```http
POST /api/v1/modules/{name}/versions/{version}/compile
```

**Parameters:**
- `name` (path): Module name
- `version` (path): Module version

**Request Body:**

```json
{
  "languages": ["go", "python", "rust"],
  "include_grpc": true,
  "options": {
    "go_package": "github.com/company/service",
    "java_package": "com.company.service"
  }
}
```

**Fields:**
- `languages` (string[]): List of language IDs to compile for (required)
- `include_grpc` (boolean): Include gRPC code generation (optional, default: false)
- `options` (object): Language-specific options (optional)

**Response:** `200 OK`

```json
{
  "job_id": "user-service-v1.0.0",
  "results": [
    {
      "id": "user-service-v1.0.0-go",
      "language": "go",
      "status": "completed",
      "started_at": "2026-01-25T10:00:00Z",
      "completed_at": "2026-01-25T10:00:02Z",
      "duration_ms": 1850,
      "cache_hit": false,
      "s3_key": "compiled/user-service/v1.0.0/go.tar.gz",
      "s3_bucket": "spoke-artifacts"
    }
  ]
}
```

**Status Values:**
- `pending`: Compilation queued
- `running`: Currently compiling
- `completed`: Successfully completed
- `failed`: Compilation failed

**Errors:**
- `400 Bad Request`: Invalid request (e.g., empty languages array)
- `404 Not Found`: Module version not found
- `503 Service Unavailable`: Compilation service unavailable

#### Get Compilation Job Status

Returns the status of a specific compilation job.

```http
GET /api/v1/modules/{name}/versions/{version}/compile/{jobId}
```

**Parameters:**
- `name` (path): Module name
- `version` (path): Module version
- `jobId` (path): Job ID

**Response:** `200 OK`

```json
{
  "id": "user-service-v1.0.0-go",
  "language": "go",
  "status": "completed",
  "started_at": "2026-01-25T10:00:00Z",
  "completed_at": "2026-01-25T10:00:02Z",
  "duration_ms": 1850,
  "cache_hit": false,
  "error": "",
  "s3_key": "compiled/user-service/v1.0.0/go.tar.gz",
  "s3_bucket": "spoke-artifacts"
}
```

**Errors:**
- `404 Not Found`: Job not found
- `503 Service Unavailable`: Compilation service unavailable

### Modules (Legacy Endpoints)

#### Create Module

```http
POST /modules
```

**Request Body:**

```json
{
  "name": "user-service",
  "description": "User management service"
}
```

#### List Modules

```http
GET /modules
```

#### Get Module

```http
GET /modules/{name}
```

### Versions (Legacy Endpoints)

#### Create Version

```http
POST /modules/{name}/versions
```

**Request Body:**

```json
{
  "version": "v1.0.0",
  "files": [
    {
      "path": "user.proto",
      "content": "syntax = \"proto3\";\n..."
    }
  ],
  "dependencies": ["common:v1.0.0"],
  "source_info": {
    "repository": "github.com/company/protos",
    "commit_sha": "abc123",
    "branch": "main"
  }
}
```

#### List Versions

```http
GET /modules/{name}/versions
```

#### Get Version

```http
GET /modules/{name}/versions/{version}
```

### Files (Legacy Endpoints)

#### Get File

```http
GET /modules/{name}/versions/{version}/files/{path}
```

### Downloads (Legacy Endpoints)

#### Download Compiled Artifacts

Downloads pre-compiled artifacts for a specific language.

```http
GET /modules/{name}/versions/{version}/download/{language}
```

**Parameters:**
- `name` (path): Module name
- `version` (path): Module version
- `language` (path): Language ID (e.g., `go`, `python`)

**Response:** `200 OK`

Returns a tar.gz archive containing:
- Compiled code files
- Package manager configuration (go.mod, setup.py, etc.)
- README with usage instructions

**Content-Type:** `application/gzip`

### Documentation Features

#### Get Code Examples

Returns auto-generated code examples for a specific module version and language.

```http
GET /api/v1/modules/{name}/versions/{version}/examples/{language}
```

**Parameters:**
- `name` (path): Module name
- `version` (path): Module version
- `language` (path): Language ID (e.g., `go`, `python`, `rust`)

**Response:** `200 OK`

Returns language-specific code example as plain text showing how to use the module's services and methods.

**Content-Type:** `text/plain`

**Example Response (Go):**

```go
package main

import (
    "context"
    "log"

    pb "github.com/company/user-service"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to gRPC server
    conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // Create client
    client := pb.NewUserServiceClient(conn)
    ctx := context.Background()

    // Call CreateUser
    resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
        Email: "user@example.com",
        Name: "John Doe",
    })
    if err != nil {
        log.Printf("Error calling CreateUser: %v", err)
    } else {
        log.Printf("CreateUser response: %+v", resp)
    }
}
```

**Errors:**
- `404 Not Found`: Module, version, or language not found
- `500 Internal Server Error`: Example generation failed

**Supported Languages:**
- go, python, java, cpp, csharp, rust, typescript, javascript
- dart, swift, kotlin, objc, ruby, php, scala

#### Compare Schema Versions

Compares two versions of a module to detect schema changes and breaking changes.

```http
POST /api/v1/modules/{name}/diff
```

**Parameters:**
- `name` (path): Module name

**Request Body:**

```json
{
  "from_version": "v1.0.0",
  "to_version": "v1.1.0"
}
```

**Response:** `200 OK`

```json
{
  "changes": [
    {
      "type": "field_removed",
      "severity": "breaking",
      "location": "user.proto:message User:field phone_number",
      "old_value": "string phone_number = 3",
      "new_value": null,
      "description": "Field 'phone_number' was removed from message 'User'",
      "migration_tip": "Update all code that references User.phone_number. Consider using ContactInfo message instead."
    },
    {
      "type": "field_added",
      "severity": "non_breaking",
      "location": "user.proto:message User:field contact_info",
      "old_value": null,
      "new_value": "ContactInfo contact_info = 4",
      "description": "Field 'contact_info' was added to message 'User'",
      "migration_tip": "New field is optional and backward compatible."
    }
  ]
}
```

**Change Types:**
- `field_added`: New field added to message
- `field_removed`: Field removed from message
- `field_renamed`: Field name changed
- `type_changed`: Field type changed
- `field_number_changed`: Field number changed
- `message_added`: New message type added
- `message_removed`: Message type removed
- `enum_added`: New enum type added
- `enum_removed`: Enum type removed
- `enum_value_added`: New enum value added
- `enum_value_removed`: Enum value removed
- `service_added`: New service added
- `service_removed`: Service removed
- `method_added`: New service method added
- `method_removed`: Service method removed
- `import_added`: New import added

**Severity Levels:**
- `breaking`: Change breaks backward compatibility
- `non_breaking`: Change is backward compatible
- `warning`: Potential compatibility issue

**Errors:**
- `400 Bad Request`: Invalid request (missing versions)
- `404 Not Found`: Module or version not found
- `500 Internal Server Error`: Diff comparison failed

## Error Responses

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong"
}
```

**Common HTTP Status Codes:**
- `200 OK`: Request successful
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

Currently, no rate limiting is enforced. Future versions may add rate limits.

## Examples

### Example 1: List Languages and Compile

```bash
# List all languages
curl http://localhost:8080/api/v1/languages

# Get details for Rust
curl http://localhost:8080/api/v1/languages/rust

# Trigger compilation
curl -X POST http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile \
  -H "Content-Type: application/json" \
  -d '{
    "languages": ["go", "rust"],
    "include_grpc": true
  }'

# Check job status
curl http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile/user-service-v1.0.0-go
```

### Example 2: Python Client

```python
import requests

# List languages
response = requests.get('http://localhost:8080/api/v1/languages')
languages = response.json()
print(f"Available languages: {len(languages)}")

# Trigger compilation
compile_request = {
    'languages': ['go', 'python'],
    'include_grpc': True,
    'options': {
        'go_package': 'github.com/company/service'
    }
}

response = requests.post(
    'http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile',
    json=compile_request
)

result = response.json()
print(f"Job ID: {result['job_id']}")

# Poll for completion
job_id = result['results'][0]['id']
while True:
    response = requests.get(
        f'http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile/{job_id}'
    )
    job = response.json()

    if job['status'] in ['completed', 'failed']:
        break

    time.sleep(1)

print(f"Status: {job['status']}, Duration: {job['duration_ms']}ms")
```

### Example 3: JavaScript/TypeScript Client

```typescript
// List languages
const response = await fetch('http://localhost:8080/api/v1/languages');
const languages = await response.json();
console.log(`Available languages: ${languages.length}`);

// Trigger compilation
const compileRequest = {
  languages: ['typescript', 'javascript'],
  include_grpc: true
};

const compileResponse = await fetch(
  'http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile',
  {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(compileRequest)
  }
);

const result = await compileResponse.json();
console.log(`Job ID: ${result.job_id}`);

// Wait for completion
const jobId = result.results[0].id;
let job;
do {
  const jobResponse = await fetch(
    `http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/compile/${jobId}`
  );
  job = await jobResponse.json();

  if (job.status === 'completed' || job.status === 'failed') {
    break;
  }

  await new Promise(resolve => setTimeout(resolve, 1000));
} while (true);

console.log(`Status: ${job.status}, Duration: ${job.duration_ms}ms`);
```

## Webhooks (Future)

Future versions may support webhooks for compilation events:

```json
{
  "event": "compilation.completed",
  "job_id": "user-service-v1.0.0-go",
  "module": "user-service",
  "version": "v1.0.0",
  "language": "go",
  "status": "completed",
  "timestamp": "2026-01-25T10:00:02Z"
}
```

## OpenAPI Specification

An OpenAPI 3.0 specification is available at:

```
http://localhost:8080/api/v1/openapi.json
```

(Future enhancement)

## Client Libraries

Official client libraries (planned):
- Go: `github.com/platinummonkey/spoke-go`
- Python: `spoke-client`
- TypeScript: `@spoke/client`
- Java: `com.platinummonkey.spoke:spoke-client`

## Support

For API support:
- GitHub Issues: https://github.com/platinummonkey/spoke/issues
- Documentation: https://spoke.dev/docs
