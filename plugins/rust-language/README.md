# Rust Language Plugin for Spoke

This plugin adds Rust code generation support to Spoke using the `prost` Protocol Buffers implementation and `tonic` for gRPC.

## Features

- **Protocol Buffers**: Generate Rust code from `.proto` files using `prost`
- **gRPC Support**: Generate gRPC service stubs using `tonic`
- **Cargo Integration**: Automatically generate `Cargo.toml` with correct dependencies
- **WebAssembly Ready**: Generated code works with WebAssembly targets

## Installation

### From Plugin Directory

1. Copy this directory to your Spoke plugins folder:
   ```bash
   cp -r plugins/rust-language ~/.spoke/plugins/
   ```

2. Restart Spoke or Sprocket to load the plugin

### Verification

Check that the plugin is loaded:
```bash
spoke plugin list
```

You should see:
```
rust-language (Rust Language Plugin) v1.2.0 - verified
```

## Usage

### With Spoke CLI

Compile a module to Rust:
```bash
spoke compile -module mymodule -version v1.0.0 -lang rust -out ./generated
```

### With Sprocket

Sprocket will automatically compile to Rust when new proto files are detected if the Rust plugin is enabled.

### Generated Files

For a proto file `user.proto`:
```protobuf
syntax = "proto3";

package example;

message User {
  string name = 1;
  int32 age = 2;
}
```

Generated file `user.rs`:
```rust
#[allow(clippy::derive_partial_eq_without_eq)]
#[derive(Clone, PartialEq, ::prost::Message)]
pub struct User {
    #[prost(string, tag = "1")]
    pub name: ::prost::alloc::string::String,
    #[prost(int32, tag = "2")]
    pub age: i32,
}
```

### Cargo.toml

The plugin generates a `Cargo.toml` with required dependencies:
```toml
[package]
name = "example"
version = "1.0.0"
edition = "2021"

[dependencies]
prost = "0.11"
tonic = "0.9"  # Only if gRPC services are present
```

## Configuration

### Custom Options

You can pass custom options when compiling:
```bash
spoke compile -module mymodule -version v1.0.0 -lang rust \
  -opt bytes=. \
  -opt file_descriptor_set=true
```

### Docker Image

The plugin uses the `spoke/rust-protoc:1.2` Docker image, which includes:
- Rust toolchain
- `protoc` 3.21+
- `protoc-gen-prost`
- `protoc-gen-tonic`

## Requirements

- Protocol Buffers compiler (`protoc`) 3.15.0 or later
- Rust 1.65.0 or later (for generated code)

## Plugin Details

- **ID**: `rust-language`
- **Version**: 1.2.0
- **API Version**: 1.0.0
- **Type**: language
- **Security Level**: verified
- **License**: MIT

## Permissions

This plugin requires the following permissions:
- `filesystem:read` - Read proto files
- `filesystem:write` - Write generated Rust files
- `process:exec` - Execute protoc compiler

## Development

### Building from Source

If you want to extend this plugin:

1. Clone the repository
2. Modify `language_spec.yaml` or add custom logic
3. Test with:
   ```bash
   spoke plugin validate ./rust-language
   ```

### Testing

Test the plugin generates valid code:
```bash
# Generate code
spoke compile -module test -version v1.0.0 -lang rust -out ./test-output

# Verify Rust code compiles
cd test-output
cargo check
```

## Troubleshooting

### Plugin not loaded

Check Spoke logs for errors:
```bash
spoke -log-level debug
```

Common issues:
- Plugin directory not in search path
- Invalid `plugin.yaml` format
- API version incompatibility

### Compilation fails

Verify protoc is available:
```bash
protoc --version
```

Check Docker image exists:
```bash
docker pull spoke/rust-protoc:1.2
```

### Generated code doesn't compile

Ensure you have the correct Rust version:
```bash
rustc --version  # Should be 1.65.0 or later
```

Add dependencies to your `Cargo.toml`:
```toml
[dependencies]
prost = "0.11"
tonic = "0.9"
```

## Support

- **Issues**: https://github.com/spoke-plugins/rust-language/issues
- **Discussions**: https://github.com/platinummonkey/spoke/discussions
- **Documentation**: https://plugins.spoke.dev/rust

## License

MIT License - see LICENSE file for details
