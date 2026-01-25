---
title: "Your First Module"
weight: 4
---

# Creating Your First Protobuf Module

This tutorial walks you through creating, versioning, and managing your first protobuf module in Spoke.

## What You'll Build

We'll create a simple user management service with:
- User data model
- gRPC service definition
- Common types module
- Multiple versions

## Prerequisites

- Spoke server running (see [Quick Start](/getting-started/quick-start/))
- `spoke-cli` installed
- Basic understanding of Protocol Buffers

## Step 1: Create Common Types Module

First, let's create a module for common types that can be shared across services.

### Create Directory Structure

```bash
mkdir -p tutorial/common/proto
cd tutorial/common/proto
```

### Create types.proto

```proto
// types.proto
syntax = "proto3";

package common.v1;

option go_package = "github.com/example/common/v1;commonv1";

// UUID represents a universally unique identifier
message UUID {
  string value = 1;  // UUID in string format (e.g., "550e8400-e29b-41d4-a716-446655440000")
}

// Timestamp represents a point in time
message Timestamp {
  int64 seconds = 1;  // Seconds since Unix epoch
  int32 nanos = 2;    // Nanoseconds component
}

// Status represents a generic status
enum Status {
  STATUS_UNSPECIFIED = 0;
  STATUS_ACTIVE = 1;
  STATUS_INACTIVE = 2;
  STATUS_DELETED = 3;
}

// Pagination for list requests
message PaginationRequest {
  int32 page = 1;      // Page number (1-indexed)
  int32 page_size = 2; // Items per page
}

// Pagination metadata for responses
message PaginationResponse {
  int32 page = 1;         // Current page
  int32 page_size = 2;    // Items per page
  int32 total_pages = 3;  // Total number of pages
  int64 total_items = 4;  // Total number of items
}
```

### Push to Spoke

```bash
cd ../..  # Back to tutorial directory

./spoke-cli push \
  -module common \
  -version v1.0.0 \
  -dir common/proto \
  -registry http://localhost:8080 \
  -description "Common types shared across services"
```

Output:
```
Successfully pushed module 'common' version 'v1.0.0'
Files uploaded: 1
Module URL: http://localhost:8080/modules/common/versions/v1.0.0
```

## Step 2: Create User Service Module

Now let's create a user service that imports our common types.

### Create Directory Structure

```bash
mkdir -p user/proto
cd user/proto
```

### Create user.proto

```proto
// user.proto
syntax = "proto3";

package user.v1;

import "types.proto";

option go_package = "github.com/example/user/v1;userv1";

// User represents a user account
message User {
  common.v1.UUID id = 1;
  string email = 2;
  string first_name = 3;
  string last_name = 4;
  common.v1.Status status = 5;
  common.v1.Timestamp created_at = 6;
  common.v1.Timestamp updated_at = 7;
}

// CreateUserRequest is the request to create a new user
message CreateUserRequest {
  string email = 1;
  string first_name = 2;
  string last_name = 3;
  string password = 4;
}

// CreateUserResponse is the response after creating a user
message CreateUserResponse {
  User user = 1;
}

// GetUserRequest is the request to get a user by ID
message GetUserRequest {
  common.v1.UUID id = 1;
}

// GetUserResponse is the response containing a user
message GetUserResponse {
  User user = 1;
}

// ListUsersRequest is the request to list users
message ListUsersRequest {
  common.v1.PaginationRequest pagination = 1;
  common.v1.Status status = 2;  // Filter by status
}

// ListUsersResponse is the response containing multiple users
message ListUsersResponse {
  repeated User users = 1;
  common.v1.PaginationResponse pagination = 2;
}

// UpdateUserRequest is the request to update a user
message UpdateUserRequest {
  common.v1.UUID id = 1;
  string first_name = 2;
  string last_name = 3;
}

// UpdateUserResponse is the response after updating a user
message UpdateUserResponse {
  User user = 1;
}

// DeleteUserRequest is the request to delete a user
message DeleteUserRequest {
  common.v1.UUID id = 1;
}

// DeleteUserResponse is the response after deleting a user
message DeleteUserResponse {
  bool success = 1;
}

// UserService manages user accounts
service UserService {
  // CreateUser creates a new user account
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);

  // GetUser retrieves a user by ID
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // ListUsers lists all users with pagination
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);

  // UpdateUser updates an existing user
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);

  // DeleteUser soft-deletes a user
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
}
```

### Push to Spoke

```bash
cd ../..  # Back to tutorial directory

./spoke-cli push \
  -module user \
  -version v1.0.0 \
  -dir user/proto \
  -registry http://localhost:8080 \
  -description "User management service"
```

## Step 3: Pull and Compile

Now let's pull both modules and compile them.

### Pull with Dependencies

```bash
mkdir -p output

# Pull user module with recursive dependency resolution
./spoke-cli pull \
  -module user \
  -version v1.0.0 \
  -dir output \
  -registry http://localhost:8080 \
  -recursive
```

This will download both `user.proto` and `types.proto` (the dependency).

### Compile to Go

```bash
./spoke-cli compile \
  -dir output \
  -out output/generated/go \
  -lang go
```

You should now have:
```
output/generated/go/
├── user.pb.go
├── user_grpc.pb.go
└── types.pb.go
```

### Compile to Python

```bash
./spoke-cli compile \
  -dir output \
  -out output/generated/python \
  -lang python
```

You should now have:
```
output/generated/python/
├── user_pb2.py
├── user_pb2_grpc.py
└── types_pb2.py
```

## Step 4: Validate Your Schemas

Before pushing, it's good practice to validate your proto files:

```bash
./spoke-cli validate -dir user/proto
```

Output:
```
Validating proto files in user/proto...
✓ user.proto is valid
All files validated successfully
```

## Step 5: Version Your Module

Let's make a change and create a new version.

### Update user.proto

Add a new field to the `User` message:

```proto
message User {
  common.v1.UUID id = 1;
  string email = 2;
  string first_name = 3;
  string last_name = 4;
  common.v1.Status status = 5;
  common.v1.Timestamp created_at = 6;
  common.v1.Timestamp updated_at = 7;
  string phone_number = 8;  // NEW FIELD
}
```

### Push New Version

```bash
./spoke-cli push \
  -module user \
  -version v1.1.0 \
  -dir user/proto \
  -registry http://localhost:8080 \
  -description "Added phone number field"
```

### Check Version History

```bash
# Using CLI (if implemented)
./spoke-cli versions -module user -registry http://localhost:8080

# Or using curl
curl http://localhost:8080/modules/user/versions
```

Output:
```json
{
  "module": "user",
  "versions": [
    {
      "version": "v1.0.0",
      "created_at": "2025-01-24T10:00:00Z",
      "description": "User management service"
    },
    {
      "version": "v1.1.0",
      "created_at": "2025-01-24T10:15:00Z",
      "description": "Added phone number field"
    }
  ]
}
```

## Step 6: Use in Your Application

### Go Example

```go
package main

import (
    "context"
    "log"

    "google.golang.org/grpc"
    userv1 "github.com/example/user/v1"
)

func main() {
    // Connect to user service
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // Create client
    client := userv1.NewUserServiceClient(conn)

    // Create a user
    resp, err := client.CreateUser(context.Background(), &userv1.CreateUserRequest{
        Email:     "john@example.com",
        FirstName: "John",
        LastName:  "Doe",
        Password:  "secret123",
    })
    if err != nil {
        log.Fatalf("CreateUser failed: %v", err)
    }

    log.Printf("Created user: %s", resp.User.Email)
}
```

### Python Example

```python
import grpc
import user_pb2
import user_pb2_grpc

def main():
    # Connect to user service
    channel = grpc.insecure_channel('localhost:50051')
    stub = user_pb2_grpc.UserServiceStub(channel)

    # Create a user
    response = stub.CreateUser(user_pb2.CreateUserRequest(
        email='john@example.com',
        first_name='John',
        last_name='Doe',
        password='secret123'
    ))

    print(f"Created user: {response.user.email}")

if __name__ == '__main__':
    main()
```

## Best Practices Learned

### 1. Module Organization

```
common/          # Shared types
user/            # User service
order/           # Order service
payment/         # Payment service
```

### 2. Versioning

- Use semantic versioning: `v1.0.0`, `v1.1.0`, `v2.0.0`
- `v1.0.0` → `v1.1.0`: Backward compatible changes (add fields)
- `v1.0.0` → `v2.0.0`: Breaking changes (remove fields, change types)

### 3. Proto File Structure

```proto
syntax = "proto3";

package service.version;

import "common/types.proto";

option go_package = "github.com/org/service/version;version";

// Data models
message Entity { ... }

// Request/Response messages
message CreateEntityRequest { ... }
message CreateEntityResponse { ... }

// Service definition
service EntityService { ... }
```

### 4. Field Naming

- Use `snake_case` for field names: `first_name`, `created_at`
- Use `PascalCase` for message names: `CreateUserRequest`
- Use `UPPER_CASE` for enum values: `STATUS_ACTIVE`

### 5. Reserved Fields

When deprecating fields, reserve them:

```proto
message User {
  reserved 9;  // Old field
  reserved "deprecated_field";

  UUID id = 1;
  string email = 2;
  // ...
}
```

## Next Steps

Now that you've created your first module, explore:

- [CLI Reference](/guides/cli-reference/) - Complete CLI commands
- [Module Management](/guides/module-management/) - Advanced module operations
- [Version Control](/guides/version-control/) - Versioning strategies
- [Tutorials](/tutorials/) - More advanced tutorials
- [Examples](/examples/) - Real-world examples

## Troubleshooting

### Import Not Found

**Problem**: `types.proto: File not found`

**Solution**: Pull with `-recursive` flag:

```bash
./spoke-cli pull -module user -version v1.0.0 -dir output -recursive
```

### Compilation Fails

**Problem**: `protoc-gen-go: program not found`

**Solution**: Install the Go plugin:

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

### Version Already Exists

**Problem**: `Version v1.0.0 already exists`

**Solution**: Use a new version number or delete the old version (not recommended).

```bash
# Push with new version
./spoke-cli push -module user -version v1.0.1 -dir user/proto
```
