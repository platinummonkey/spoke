---
title: "Using Spoke with gRPC Services"
weight: 3
---

# Using Spoke with gRPC Services

Learn how to use Spoke to manage protobuf definitions for gRPC microservices.

## Overview

In this tutorial, you'll:
1. Define a gRPC service in protobuf
2. Push it to Spoke
3. Create a Go gRPC server
4. Create a Python gRPC client
5. Communicate between services

## Prerequisites

- Spoke server running
- `spoke-cli` installed
- Go 1.16+
- Python 3.7+
- Basic understanding of gRPC

## Step 1: Define the gRPC Service

### Create Directory Structure

```bash
mkdir -p grpc-tutorial/proto
cd grpc-tutorial/proto
```

### Create user_service.proto

```proto
syntax = "proto3";

package user.v1;

option go_package = "github.com/example/user/v1;userv1";

// User represents a user account
message User {
  string id = 1;
  string email = 2;
  string first_name = 3;
  string last_name = 4;
  int64 created_at = 5;
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
  string id = 1;
}

// GetUserResponse is the response containing a user
message GetUserResponse {
  User user = 1;
  bool found = 2;
}

// ListUsersRequest is the request to list users
message ListUsersRequest {
  int32 page = 1;
  int32 page_size = 2;
}

// ListUsersResponse is the response containing multiple users
message ListUsersResponse {
  repeated User users = 1;
  int32 total_count = 2;
}

// UserService manages user accounts
service UserService {
  // CreateUser creates a new user account
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);

  // GetUser retrieves a user by ID
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // ListUsers lists all users with pagination
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
}
```

## Step 2: Push to Spoke

```bash
cd ..  # Back to grpc-tutorial directory

spoke-cli push \
  -module user-service \
  -version v1.0.0 \
  -dir ./proto \
  -registry http://localhost:8080 \
  -description "User service gRPC API"
```

## Step 3: Create Go gRPC Server

### Set Up Go Project

```bash
mkdir -p server
cd server

go mod init github.com/example/user-server
```

### Pull and Compile Schemas

```bash
mkdir proto
spoke-cli pull \
  -module user-service \
  -version v1.0.0 \
  -dir ./proto \
  -registry http://localhost:8080

# Compile to Go
spoke-cli compile \
  -dir ./proto \
  -out ./pb \
  -lang go
```

### Install Dependencies

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf/reflect/protoreflect
```

### Create Server Implementation

Create `server/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"

    pb "github.com/example/user-server/pb"
)

type userServer struct {
    pb.UnimplementedUserServiceServer
    users map[string]*pb.User
}

func newUserServer() *userServer {
    return &userServer{
        users: make(map[string]*pb.User),
    }
}

func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
    id := fmt.Sprintf("user-%d", time.Now().Unix())

    user := &pb.User{
        Id:        id,
        Email:     req.Email,
        FirstName: req.FirstName,
        LastName:  req.LastName,
        CreatedAt: time.Now().Unix(),
    }

    s.users[id] = user

    log.Printf("Created user: %s (%s)", user.Email, user.Id)

    return &pb.CreateUserResponse{
        User: user,
    }, nil
}

func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, found := s.users[req.Id]

    if !found {
        return &pb.GetUserResponse{
            Found: false,
        }, nil
    }

    return &pb.GetUserResponse{
        User:  user,
        Found: true,
    }, nil
}

func (s *userServer) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
    users := make([]*pb.User, 0, len(s.users))

    for _, user := range s.users {
        users = append(users, user)
    }

    return &pb.ListUsersResponse{
        Users:      users,
        TotalCount: int32(len(users)),
    }, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    grpcServer := grpc.NewServer()
    pb.RegisterUserServiceServer(grpcServer, newUserServer())

    // Enable reflection for grpcurl
    reflection.Register(grpcServer)

    log.Println("gRPC server listening on :50051")

    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Run the Server

```bash
go run main.go
```

Output:
```
gRPC server listening on :50051
```

## Step 4: Create Python gRPC Client

### Set Up Python Project

```bash
cd ..  # Back to grpc-tutorial
mkdir client
cd client

python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

pip install grpcio grpcio-tools
```

### Pull and Compile Schemas

```bash
mkdir proto
spoke-cli pull \
  -module user-service \
  -version v1.0.0 \
  -dir ./proto \
  -registry http://localhost:8080

# Compile to Python
spoke-cli compile \
  -dir ./proto \
  -out ./pb \
  -lang python
```

### Create Client

Create `client/client.py`:

```python
import grpc
import sys
sys.path.insert(0, './pb')

import user_service_pb2
import user_service_pb2_grpc


def create_user(stub, email, first_name, last_name):
    """Create a new user"""
    request = user_service_pb2.CreateUserRequest(
        email=email,
        first_name=first_name,
        last_name=last_name,
        password="secret123"
    )

    response = stub.CreateUser(request)
    print(f"Created user: {response.user.email} (ID: {response.user.id})")
    return response.user


def get_user(stub, user_id):
    """Get a user by ID"""
    request = user_service_pb2.GetUserRequest(id=user_id)

    response = stub.GetUser(request)

    if response.found:
        user = response.user
        print(f"Found user: {user.email} ({user.first_name} {user.last_name})")
        return user
    else:
        print(f"User {user_id} not found")
        return None


def list_users(stub):
    """List all users"""
    request = user_service_pb2.ListUsersRequest(
        page=1,
        page_size=10
    )

    response = stub.ListUsers(request)
    print(f"\nFound {response.total_count} users:")

    for user in response.users:
        print(f"  - {user.email} ({user.first_name} {user.last_name})")


def main():
    # Connect to server
    channel = grpc.insecure_channel('localhost:50051')
    stub = user_service_pb2_grpc.UserServiceStub(channel)

    try:
        # Create some users
        user1 = create_user(stub, "alice@example.com", "Alice", "Smith")
        user2 = create_user(stub, "bob@example.com", "Bob", "Jones")

        print()

        # Get a user
        get_user(stub, user1.id)

        print()

        # List all users
        list_users(stub)

    except grpc.RpcError as e:
        print(f"RPC failed: {e.code()}: {e.details()}")
    finally:
        channel.close()


if __name__ == '__main__':
    main()
```

### Run the Client

```bash
python client.py
```

Output:
```
Created user: alice@example.com (ID: user-1643040000)
Created user: bob@example.com (ID: user-1643040001)

Found user: alice@example.com (Alice Smith)

Found 2 users:
  - alice@example.com (Alice Smith)
  - bob@example.com (Bob Jones)
```

## Step 5: Update the Service

Let's add a new RPC method to demonstrate schema evolution.

### Update Proto File

Add to `user_service.proto`:

```proto
// DeleteUserRequest is the request to delete a user
message DeleteUserRequest {
  string id = 1;
}

// DeleteUserResponse is the response after deleting a user
message DeleteUserResponse {
  bool success = 1;
}

service UserService {
  // ... existing methods ...

  // DeleteUser deletes a user account
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);
}
```

### Push New Version

```bash
spoke-cli push \
  -module user-service \
  -version v1.1.0 \
  -dir ./proto \
  -registry http://localhost:8080 \
  -description "Added DeleteUser RPC"
```

### Update Server

Pull new version and recompile:

```bash
cd server

spoke-cli pull \
  -module user-service \
  -version v1.1.0 \
  -dir ./proto \
  -registry http://localhost:8080

spoke-cli compile \
  -dir ./proto \
  -out ./pb \
  -lang go
```

Add to `server/main.go`:

```go
func (s *userServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
    _, exists := s.users[req.Id]

    if exists {
        delete(s.users, req.Id)
        log.Printf("Deleted user: %s", req.Id)
        return &pb.DeleteUserResponse{Success: true}, nil
    }

    return &pb.DeleteUserResponse{Success: false}, nil
}
```

Restart the server.

### Update Client

Pull new version and recompile:

```bash
cd ../client

spoke-cli pull \
  -module user-service \
  -version v1.1.0 \
  -dir ./proto \
  -registry http://localhost:8080

spoke-cli compile \
  -dir ./proto \
  -out ./pb \
  -lang python
```

Add to `client.py`:

```python
def delete_user(stub, user_id):
    """Delete a user"""
    request = user_service_pb2.DeleteUserRequest(id=user_id)

    response = stub.DeleteUser(request)

    if response.success:
        print(f"Deleted user: {user_id}")
    else:
        print(f"Failed to delete user: {user_id}")
```

## Testing with grpcurl

You can also test the server using grpcurl:

```bash
# Install grpcurl
brew install grpcurl  # macOS
# or download from: https://github.com/fullstorydev/grpcurl

# List services
grpcurl -plaintext localhost:50051 list

# List methods
grpcurl -plaintext localhost:50051 list user.v1.UserService

# Create a user
grpcurl -plaintext -d '{
  "email": "test@example.com",
  "first_name": "Test",
  "last_name": "User",
  "password": "secret"
}' localhost:50051 user.v1.UserService/CreateUser

# List users
grpcurl -plaintext -d '{"page": 1, "page_size": 10}' \
  localhost:50051 user.v1.UserService/ListUsers
```

## Project Structure

Final directory structure:

```
grpc-tutorial/
├── proto/
│   └── user_service.proto
├── server/
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── proto/
│   │   └── user_service.proto
│   └── pb/
│       ├── user_service.pb.go
│       └── user_service_grpc.pb.go
└── client/
    ├── venv/
    ├── client.py
    ├── proto/
    │   └── user_service.proto
    └── pb/
        ├── user_service_pb2.py
        └── user_service_pb2_grpc.py
```

## Benefits of Using Spoke

1. **Single Source of Truth**: Proto files stored centrally in Spoke
2. **Version Management**: Track changes over time (v1.0.0 → v1.1.0)
3. **Multi-Language**: Same proto generates Go server and Python client
4. **Easy Updates**: Pull latest version and recompile
5. **No Manual Copying**: No need to manually copy proto files between services

## Next Steps

- [Event Streaming Tutorial](/tutorials/event-streaming/) - Use Spoke with Kafka
- [Polyglot Microservices](/tutorials/polyglot-services/) - More languages
- [Schema Evolution](/tutorials/schema-evolution/) - Managing breaking changes
- [CI/CD Integration](/tutorials/cicd-setup/) - Automate schema management
