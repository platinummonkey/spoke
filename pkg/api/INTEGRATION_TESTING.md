# Integration Testing Guide

This directory contains integration tests that test handler workflows with real database connections.

## Test Types

### Unit Tests (default)
- Use mocks (`mockStorage`, `mockOrgService`, `sqlmock`)
- Run with: `go test ./pkg/api/`
- Fast, no external dependencies

### Integration Tests
- Use real PostgreSQL via testcontainers
- Test complete workflows end-to-end
- Run with: `go test -tags=integration ./pkg/api/`
- Require Docker/Podman to be running

## Running Integration Tests

### Prerequisites

1. **Docker or Podman**: Integration tests use testcontainers to spin up PostgreSQL
   ```bash
   # Check Docker is running
   docker ps

   # Or check Podman
   podman ps
   ```

2. **Sufficient disk space**: Testcontainers will pull postgres:15-alpine image (~80MB)

### Running Tests

```bash
# Run all integration tests
go test -tags=integration -v ./pkg/api/

# Run specific integration test
go test -tags=integration -v ./pkg/api/ -run TestIntegration_AuthWorkflow

# Run with timeout (container startup can be slow)
go test -tags=integration -v -timeout=5m ./pkg/api/

# Skip integration tests (default behavior)
go test ./pkg/api/
```

### CI/CD

Integration tests are skipped by default in CI unless explicitly enabled.
To run them in CI, ensure Docker is available and use:

```yaml
- name: Run integration tests
  run: go test -tags=integration -v -timeout=10m ./...
```

## Available Integration Tests

### TestIntegration_AuthWorkflow
Tests authentication and authorization workflows:
- User creation, retrieval, update
- Organization creation and retrieval
- Organization membership management

### TestIntegration_ModuleWorkflow
Tests the complete module CRUD workflow:
- Module creation
- Version creation and retrieval
- Module listing

### TestIntegration_VersionDependencies
Tests version dependency management:
- Creating modules with dependencies
- Dependency resolution

## Test Database Schema

Integration tests automatically:
1. Start a PostgreSQL 15 container
2. Run migrations from `migrations/` directory:
   - `001_create_base_schema.up.sql`
   - `002_create_auth_schema.up.sql`
3. Execute tests
4. Clean up the container

## Troubleshooting

### "Docker/Podman not available"
- Ensure Docker or Podman is installed and running
- Check permissions: `docker ps` should work without sudo

### "Failed to start PostgreSQL container"
- Check Docker has enough resources (memory, disk)
- Try pulling the image manually: `docker pull postgres:15-alpine`
- Check network connectivity

### "Network not found"
- This can happen on some Docker setups
- Try: `docker network create bridge` or use default network

### Tests timing out
- Increase timeout: `-timeout=10m`
- Container startup can be slow on first run (image download)

## Writing New Integration Tests

```go
func TestIntegration_YourFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    db, cleanup := setupIntegrationTestDB(t)
    defer cleanup()

    // Your test code here using real database
}
```

## Performance

Integration tests are slower than unit tests:
- **Unit tests**: ~0.5s for pkg/api
- **Integration tests**: ~60-90s (includes container startup)

Run unit tests by default during development. Save integration tests for:
- Pre-commit hooks
- Pull requests
- Release testing
