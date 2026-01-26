# Buf Connect for Go Plugin

This plugin wraps the [Buf Connect for Go](https://buf.build/library/connect-go) plugin to work seamlessly with Spoke.

## What is Connect?

Connect is a simple, reliable RPC framework from Buf Technologies that works with Protocol Buffers. It provides a simpler alternative to gRPC with better browser support and cleaner APIs.

## Features

- **Automatic Download**: Plugin binary downloaded from Buf registry on first use
- **Caching**: Downloaded binaries cached in `~/.buf/plugins/`
- **Native Execution**: Runs without Docker (faster compilation)
- **gRPC Compatible**: Works with existing gRPC services

## Installation

### From Spoke Plugin Directory

1. Copy to your plugins folder:
   ```bash
   cp -r plugins/buf-connect-go ~/.spoke/plugins/
   ```

2. Restart Spoke/Sprocket - plugin auto-downloads on first use

### Verification

Check the plugin loaded:
```bash
spoke plugin list | grep connect
```

## Usage

### Compile with Spoke CLI

```bash
spoke compile -module myservice -version v1.0.0 -lang connect-go -out ./generated
```

### Auto-compilation with Sprocket

Sprocket automatically detects and compiles with this plugin when enabled.

## How It Works

1. **Plugin Discovery**: Spoke finds `plugin.yaml` with `buf_registry` metadata
2. **Binary Download**: On first use, downloads from `buf.build/library/connect-go`
3. **Caching**: Binary cached at `~/.buf/plugins/connect-go/v1.5.0/`
4. **Compilation**: Runs `protoc` with downloaded plugin binary

## Example

Given `user.proto`:
```protobuf
syntax = "proto3";

package user.v1;

service UserService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

message GetUserRequest {
  string user_id = 1;
}

message GetUserResponse {
  string name = 1;
  string email = 2;
}
```

Generated files:
- `user.pb.go` - Message definitions
- `userconnect/user.connect.go` - Connect service stubs

## Generated Code Example

```go
package userconnect

import (
    "context"
    userv1 "example/gen/user/v1"
    "connectrpc.com/connect"
)

// UserServiceClient is the client API for UserService.
type UserServiceClient interface {
    GetUser(context.Context, *connect.Request[userv1.GetUserRequest]) (
        *connect.Response[userv1.GetUserResponse], error)
}

// UserServiceHandler is the server API for UserService.
type UserServiceHandler interface {
    GetUser(context.Context, *connect.Request[userv1.GetUserRequest]) (
        *connect.Response[userv1.GetUserResponse], error)
}
```

## Cache Management

View cached plugins:
```bash
ls ~/.buf/plugins/
```

Clear cache:
```bash
rm -rf ~/.buf/plugins/connect-go
```

Verify cache integrity:
```bash
spoke plugin verify-cache
```

## Configuration

The plugin uses these settings from `plugin.yaml`:

```yaml
metadata:
  buf_registry: buf.build/library/connect-go  # Buf registry URL
  buf_version: v1.5.0  # Plugin version to download
```

## Troubleshooting

### Plugin binary not found

**Solution**: Clear cache and retry:
```bash
rm -rf ~/.buf/plugins/connect-go
spoke compile -module test -version v1.0.0 -lang connect-go -out ./test
```

### Download fails

**Check**:
1. Internet connection
2. Buf registry is accessible: `curl https://buf.build`
3. Platform support: Only `linux_amd64`, `darwin_amd64`, `darwin_arm64`, `windows_amd64` supported

### Compilation errors

**Verify**:
```bash
# Check protoc installed
protoc --version  # Should be 3.15.0+

# Test plugin binary directly
~/.buf/plugins/connect-go/v1.5.0/protoc-gen-connect-go --version
```

## Differences from Standard gRPC

**Connect Benefits**:
- Simpler API (no streaming complexity for simple RPCs)
- Better browser support (works with fetch API)
- Smaller generated code
- HTTP/1.1 and HTTP/2 support
- JSON and binary encoding

**When to use Connect**:
- ✅ Browser clients
- ✅ Simple request/response APIs
- ✅ HTTP/1.1 environments
- ✅ JSON-based APIs with type safety

**When to use gRPC**:
- Bidirectional streaming needed
- Maximum performance critical
- Existing gRPC infrastructure

## Resources

- **Buf Registry**: https://buf.build/library/connect-go
- **Connect Documentation**: https://connectrpc.com/docs/go/getting-started
- **GitHub**: https://github.com/bufbuild/connect-go
- **Spoke Plugin Docs**: https://docs.spoke.dev/plugins

## Plugin Details

- **ID**: `buf-connect-go`
- **Version**: 1.5.0
- **API Version**: 1.0.0
- **Type**: language (Buf plugin)
- **Security Level**: verified
- **License**: Apache-2.0

## Support

- **Buf Issues**: https://github.com/bufbuild/connect-go/issues
- **Spoke Issues**: https://github.com/platinummonkey/spoke/issues
- **Discussions**: https://github.com/platinummonkey/spoke/discussions
