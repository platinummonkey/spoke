---
title: "Code Examples"
weight: 2
---

# Auto-Generated Code Examples

Spoke automatically generates working code examples in 15+ programming languages, showing how to use your protobuf services with gRPC clients.

## Overview

The **Usage Examples** tab provides:
- Ready-to-use client code
- gRPC connection setup
- Service instantiation
- Method calls with sample data
- Error handling
- Copy-to-clipboard functionality

## Accessing Code Examples

1. Navigate to any module
2. Select a version
3. Click the **Usage Examples** tab (4th tab)
4. Select your language from the dropdown
5. Click the copy button to copy the entire example

## Supported Languages

| Language | Framework | Package Manager |
|----------|-----------|-----------------|
| Go | google.golang.org/grpc | go modules |
| Python | grpcio | pip |
| Java | gRPC-Java | maven |
| C++ | gRPC C++ | cmake |
| C# | Grpc.Net.Client | nuget |
| Rust | tonic | cargo |
| TypeScript | @grpc/grpc-js | npm |
| JavaScript | @grpc/grpc-js | npm |
| Dart | grpc | pub |
| Swift | gRPC-Swift | swift package manager |
| Kotlin | gRPC-Kotlin | gradle |
| Objective-C | gRPC-ObjC | cocoapods |
| Ruby | grpc | gem |
| PHP | grpc | composer |
| Scala | ScalaPB | sbt |

## Example Structure

Each generated example includes:

1. **Imports**: Required packages and generated stubs
2. **Connection Setup**: gRPC channel/connection configuration
3. **Client Creation**: Service client instantiation
4. **Method Calls**: Example calls to each service method
5. **Error Handling**: Basic error handling patterns
6. **Cleanup**: Proper resource cleanup

## Language Examples

### Go

```go
package main

import (
    "context"
    "log"
    "time"

    pb "github.com/company/user-service"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func main() {
    // Connect to gRPC server
    conn, err := grpc.Dial("localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithBlock(),
        grpc.WithTimeout(5*time.Second),
    )
    if err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer conn.Close()

    // Create client
    client := pb.NewUserServiceClient(conn)
    ctx := context.Background()

    // Call CreateUser
    createResp, err := client.CreateUser(ctx, &pb.CreateUserRequest{
        Email: "user@example.com",
        Name:  "John Doe",
        Phone: "+1-555-0123",
    })
    if err != nil {
        log.Printf("Error calling CreateUser: %v", err)
    } else {
        log.Printf("CreateUser response: %+v", createResp)
    }

    // Call GetUser
    getResp, err := client.GetUser(ctx, &pb.GetUserRequest{
        UserId: createResp.UserId,
    })
    if err != nil {
        log.Printf("Error calling GetUser: %v", err)
    } else {
        log.Printf("GetUser response: %+v", getResp)
    }

    // Call UpdateUser
    updateResp, err := client.UpdateUser(ctx, &pb.UpdateUserRequest{
        UserId: createResp.UserId,
        Name:   "Jane Doe",
        Email:  "jane@example.com",
    })
    if err != nil {
        log.Printf("Error calling UpdateUser: %v", err)
    } else {
        log.Printf("UpdateUser response: %+v", updateResp)
    }
}
```

### Python

```python
import grpc
from user_service_pb2 import (
    CreateUserRequest,
    GetUserRequest,
    UpdateUserRequest,
)
from user_service_pb2_grpc import UserServiceStub


def main():
    # Connect to gRPC server
    channel = grpc.insecure_channel('localhost:50051')
    client = UserServiceStub(channel)

    try:
        # Call CreateUser
        create_response = client.CreateUser(
            CreateUserRequest(
                email="user@example.com",
                name="John Doe",
                phone="+1-555-0123"
            )
        )
        print(f"CreateUser response: {create_response}")

        # Call GetUser
        get_response = client.GetUser(
            GetUserRequest(user_id=create_response.user_id)
        )
        print(f"GetUser response: {get_response}")

        # Call UpdateUser
        update_response = client.UpdateUser(
            UpdateUserRequest(
                user_id=create_response.user_id,
                name="Jane Doe",
                email="jane@example.com"
            )
        )
        print(f"UpdateUser response: {update_response}")

    except grpc.RpcError as e:
        print(f"gRPC error: {e.code()}: {e.details()}")
    finally:
        channel.close()


if __name__ == '__main__':
    main()
```

### Rust

```rust
use tonic::transport::Channel;
use user_service::user_service_client::UserServiceClient;
use user_service::{CreateUserRequest, GetUserRequest, UpdateUserRequest};

pub mod user_service {
    tonic::include_proto!("user_service");
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Connect to gRPC server
    let channel = Channel::from_static("http://localhost:50051")
        .connect()
        .await?;

    let mut client = UserServiceClient::new(channel);

    // Call CreateUser
    let create_request = tonic::Request::new(CreateUserRequest {
        email: "user@example.com".to_string(),
        name: "John Doe".to_string(),
        phone: "+1-555-0123".to_string(),
    });

    let create_response = client.create_user(create_request).await?;
    println!("CreateUser response: {:?}", create_response.get_ref());

    // Call GetUser
    let get_request = tonic::Request::new(GetUserRequest {
        user_id: create_response.get_ref().user_id.clone(),
    });

    let get_response = client.get_user(get_request).await?;
    println!("GetUser response: {:?}", get_response.get_ref());

    // Call UpdateUser
    let update_request = tonic::Request::new(UpdateUserRequest {
        user_id: create_response.get_ref().user_id.clone(),
        name: Some("Jane Doe".to_string()),
        email: Some("jane@example.com".to_string()),
        ..Default::default()
    });

    let update_response = client.update_user(update_request).await?;
    println!("UpdateUser response: {:?}", update_response.get_ref());

    Ok(())
}
```

### TypeScript

```typescript
import * as grpc from '@grpc/grpc-js';
import { UserServiceClient } from './generated/user_service_grpc_pb';
import {
  CreateUserRequest,
  GetUserRequest,
  UpdateUserRequest,
} from './generated/user_service_pb';

async function main() {
  // Connect to gRPC server
  const client = new UserServiceClient(
    'localhost:50051',
    grpc.credentials.createInsecure()
  );

  // Call CreateUser
  const createRequest = new CreateUserRequest();
  createRequest.setEmail('user@example.com');
  createRequest.setName('John Doe');
  createRequest.setPhone('+1-555-0123');

  client.createUser(createRequest, (err, response) => {
    if (err) {
      console.error('Error calling CreateUser:', err);
      return;
    }
    console.log('CreateUser response:', response.toObject());

    // Call GetUser
    const getRequest = new GetUserRequest();
    getRequest.setUserId(response.getUserId());

    client.getUser(getRequest, (err, response) => {
      if (err) {
        console.error('Error calling GetUser:', err);
        return;
      }
      console.log('GetUser response:', response.toObject());
    });

    // Call UpdateUser
    const updateRequest = new UpdateUserRequest();
    updateRequest.setUserId(response.getUserId());
    updateRequest.setName('Jane Doe');
    updateRequest.setEmail('jane@example.com');

    client.updateUser(updateRequest, (err, response) => {
      if (err) {
        console.error('Error calling UpdateUser:', err);
        return;
      }
      console.log('UpdateUser response:', response.toObject());
    });
  });
}

main();
```

### Java

```java
import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import com.company.userservice.UserServiceGrpc;
import com.company.userservice.UserService.*;

import java.util.concurrent.TimeUnit;

public class UserServiceClient {
    private final ManagedChannel channel;
    private final UserServiceGrpc.UserServiceBlockingStub blockingStub;

    public UserServiceClient(String host, int port) {
        this.channel = ManagedChannelBuilder
            .forAddress(host, port)
            .usePlaintext()
            .build();
        this.blockingStub = UserServiceGrpc.newBlockingStub(channel);
    }

    public void shutdown() throws InterruptedException {
        channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    }

    public void createUser() {
        CreateUserRequest request = CreateUserRequest.newBuilder()
            .setEmail("user@example.com")
            .setName("John Doe")
            .setPhone("+1-555-0123")
            .build();

        try {
            CreateUserResponse response = blockingStub.createUser(request);
            System.out.println("CreateUser response: " + response);

            // Call GetUser with created user ID
            getUser(response.getUserId());

            // Call UpdateUser
            updateUser(response.getUserId());
        } catch (StatusRuntimeException e) {
            System.err.println("RPC failed: " + e.getStatus());
        }
    }

    public void getUser(String userId) {
        GetUserRequest request = GetUserRequest.newBuilder()
            .setUserId(userId)
            .build();

        try {
            GetUserResponse response = blockingStub.getUser(request);
            System.out.println("GetUser response: " + response);
        } catch (StatusRuntimeException e) {
            System.err.println("RPC failed: " + e.getStatus());
        }
    }

    public void updateUser(String userId) {
        UpdateUserRequest request = UpdateUserRequest.newBuilder()
            .setUserId(userId)
            .setName("Jane Doe")
            .setEmail("jane@example.com")
            .build();

        try {
            UpdateUserResponse response = blockingStub.updateUser(request);
            System.out.println("UpdateUser response: " + response);
        } catch (StatusRuntimeException e) {
            System.err.println("RPC failed: " + e.getStatus());
        }
    }

    public static void main(String[] args) throws Exception {
        UserServiceClient client = new UserServiceClient("localhost", 50051);
        try {
            client.createUser();
        } finally {
            client.shutdown();
        }
    }
}
```

## Sample Field Value Generation

Examples include realistic sample values based on field names and types:

| Field Name Pattern | Sample Value | Type |
|-------------------|--------------|------|
| `*email*` | `"user@example.com"` | string |
| `*name*` | `"John Doe"` | string |
| `*id*`, `*user_id*` | `"user-123"` | string |
| `*phone*` | `"+1-555-0123"` | string |
| `*address*` | `"123 Main Street"` | string |
| `*description*` | `"Example description"` | string |
| `*url*`, `*uri*` | `"https://example.com"` | string |
| `*timestamp*`, `*created_at*` | Current timestamp | int64/Timestamp |
| `*count*`, `*age*` | `42` | int32/int64 |
| `*enabled*`, `*active*` | `true` | bool |
| Enum fields | First enum value | enum |

## Streaming Examples

For streaming methods, examples include appropriate handling:

### Server Streaming (Go)

```go
stream, err := client.ListUsersStream(ctx, &pb.ListUsersRequest{})
if err != nil {
    log.Fatalf("Error calling ListUsersStream: %v", err)
}

for {
    user, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatalf("Error receiving: %v", err)
    }
    log.Printf("Received user: %+v", user)
}
```

### Client Streaming (Python)

```python
def generate_requests():
    for i in range(10):
        yield CreateUserRequest(
            email=f"user{i}@example.com",
            name=f"User {i}"
        )

response = client.BatchCreateUsers(generate_requests())
print(f"BatchCreateUsers response: {response}")
```

### Bidirectional Streaming (Rust)

```rust
let outbound = async_stream::stream! {
    for i in 0..10 {
        yield UpdateUserRequest {
            user_id: format!("user-{}", i),
            name: Some(format!("User {}", i)),
            ..Default::default()
        };
    }
};

let mut inbound = client
    .batch_update_users(Request::new(outbound))
    .await?
    .into_inner();

while let Some(response) = inbound.message().await? {
    println!("Received: {:?}", response);
}
```

## Using Generated Examples

### Step 1: Copy Example Code

Click the copy button in the UI or select and copy the entire example.

### Step 2: Set Up Project

Ensure you have the language-specific dependencies:

**Go:**
```bash
go get google.golang.org/grpc
go get <module-package-path>
```

**Python:**
```bash
pip install grpcio grpcio-tools
```

**Rust:**
```toml
[dependencies]
tonic = "0.10"
prost = "0.12"
tokio = { version = "1", features = ["full"] }
```

**TypeScript:**
```bash
npm install @grpc/grpc-js @grpc/proto-loader
```

**Java:**
```xml
<dependency>
    <groupId>io.grpc</groupId>
    <artifactId>grpc-netty-shaded</artifactId>
    <version>1.56.0</version>
</dependency>
```

### Step 3: Get Compiled Artifacts

Download pre-compiled stubs from the Overview tab or compile locally:

```bash
# Download from Spoke
curl http://localhost:8080/modules/user-service/versions/v1.0.0/download/go > user-service-go.tar.gz
tar -xzf user-service-go.tar.gz

# Or compile locally
spoke pull -module user-service -version v1.0.0
spoke compile -lang go -dir ./proto -out ./generated
```

### Step 4: Update Connection Details

Replace `localhost:50051` with your actual server address:

```go
conn, err := grpc.Dial("production.example.com:443", ...)
```

### Step 5: Customize Request Data

Replace sample values with your actual data:

```go
client.CreateUser(ctx, &pb.CreateUserRequest{
    Email: userInput.Email,  // Your actual data
    Name:  userInput.Name,
    Phone: userInput.Phone,
})
```

### Step 6: Add Production Features

Enhance examples with:
- Authentication (JWT, API keys, mTLS)
- Retry logic
- Timeout configuration
- Logging and monitoring
- Connection pooling
- Circuit breakers

## API Endpoint

Examples are also available via REST API:

```bash
curl http://localhost:8080/api/v1/modules/user-service/versions/v1.0.0/examples/go
```

Response is plain text with the generated code.

## Customizing Examples

For advanced use cases, you can customize example generation:

### Template Location

Templates are in `pkg/docs/examples/templates/{language}/grpc-client.tmpl`

### Adding Custom Sample Values

Edit `pkg/docs/examples/generator.go` to add custom field value logic:

```go
func generateSampleValue(field Field) string {
    if strings.Contains(field.Name, "company") {
        return "Acme Corp"
    }
    // ... existing logic
}
```

### Language-Specific Options

Add options to template data structure:

```go
type ExampleData struct {
    Language    string
    ModuleName  string
    Version     string
    PackagePath string
    ServiceName string
    Methods     []MethodExample
    Options     map[string]string  // Custom options
}
```

## Troubleshooting

### Example not loading

**Symptoms**: Loading spinner doesn't resolve
**Fix**: Check backend is running and module version exists

### Incorrect package paths

**Symptoms**: Import errors in generated code
**Fix**: Verify `go_package` option in proto files or use compile options

### Method not appearing

**Symptoms**: Method missing from example
**Fix**: Ensure method is defined in service and proto is valid

### Wrong sample values

**Symptoms**: Sample data doesn't make sense for field
**Fix**: Field naming might not match pattern, or customize generator

## Best Practices

### Review Before Using

- Check import paths match your project structure
- Verify sample data types match your requirements
- Add authentication if needed
- Configure timeouts appropriately

### Test in Development First

- Use examples in development/staging first
- Test error handling with invalid data
- Verify connection pooling works as expected
- Profile performance under load

### Keep Examples Updated

- Regenerate examples after proto changes
- Test examples after dependency updates
- Document custom modifications

## What's Next?

- [**API Explorer**](api-explorer) - Understand the service structure first
- [**Request Playground**](playground) - Test requests interactively
- [**CLI Reference**](../cli-reference) - Download and compile artifacts
- [**gRPC Best Practices**](../../examples/grpc-patterns) - Production patterns

## Related Documentation

- [Code Generation Guide](../../CODE_GENERATION_GUIDE.md) - Detailed compilation guide
- [API Reference](../../API_REFERENCE.md) - REST API for examples endpoint
- [Language Plugins](../../deployment/language-plugins) - Plugin versions and features
