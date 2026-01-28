# OpenAPI / Swagger Documentation

The Spoke API includes a comprehensive OpenAPI 3.0 specification that documents all endpoints, request/response schemas, authentication requirements, and more.

## Accessing the Documentation

Once the Spoke server is running, you can access the API documentation at:

- **Swagger UI (Interactive)**: `http://localhost:8080/swagger-ui` or `http://localhost:8080/api-docs`
- **OpenAPI Spec (YAML)**: `http://localhost:8080/openapi.yaml`
- **OpenAPI Spec (JSON)**: `http://localhost:8080/openapi.json` (planned)

## Using Swagger UI

The Swagger UI provides an interactive interface where you can:

1. **Browse all endpoints** organized by tags (Modules, Authentication, Billing, etc.)
2. **View request/response schemas** with detailed descriptions
3. **Try out API calls** directly from the browser
4. **Authenticate** by clicking "Authorize" and entering your Bearer token

### Authentication in Swagger UI

To make authenticated requests through Swagger UI:

1. Click the "Authorize" button at the top
2. Enter your API token in the format: `Bearer <your-token>`
3. Click "Authorize" and then "Close"
4. All subsequent requests will include your authentication

Alternatively, your token will be automatically included if stored in browser localStorage as `spoke_api_token`.

## OpenAPI Specification Details

The specification includes:

- **12 API groups**: Modules, Versions, Compilation, Validation, Compatibility, Authentication, Organizations, Billing, Search, Analytics, User Features, Plugin Verification
- **100+ endpoints** with full documentation
- **30+ schemas** for request/response objects
- **Authentication requirements** (Bearer JWT tokens)
- **Rate limiting information**
- **Error response formats**
- **Query parameters and path parameters**
- **Request body validation rules**

## Using the Specification

### Generate Client SDKs

You can use the OpenAPI specification to generate client libraries in various languages:

#### Go Client
```bash
# Using oapi-codegen
go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
oapi-codegen -package spokeclient -generate types,client openapi.yaml > client/spoke_client.go
```

#### Python Client
```bash
# Using openapi-generator
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
  -i /local/openapi.yaml \
  -g python \
  -o /local/client/python
```

#### TypeScript/JavaScript Client
```bash
# Using openapi-generator
npx @openapitools/openapi-generator-cli generate \
  -i openapi.yaml \
  -g typescript-fetch \
  -o client/typescript
```

#### Java Client
```bash
# Using openapi-generator
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli generate \
  -i /local/openapi.yaml \
  -g java \
  -o /local/client/java \
  --library=okhttp-gson
```

### Validate API Responses

Use the specification to validate that API responses match the documented schemas:

```bash
# Install spectral for validation
npm install -g @stoplight/spectral-cli

# Validate the spec itself
spectral lint openapi.yaml

# Validate API responses against the spec
# (requires additional tooling like Prism or similar)
```

### API Testing with Prism

Use Prism to create a mock server based on the spec:

```bash
# Install Prism
npm install -g @stoplight/prism-cli

# Run mock server
prism mock openapi.yaml

# Or validate requests against live server
prism proxy openapi.yaml http://localhost:8080
```

### Import into Postman

1. Open Postman
2. Click "Import" in the top left
3. Select "Link" and enter: `http://localhost:8080/openapi.yaml`
4. Postman will create a collection with all endpoints

### Import into Insomnia

1. Open Insomnia
2. Click "Create" > "Import From"
3. Select "URL" and enter: `http://localhost:8080/openapi.yaml`
4. Insomnia will create a workspace with all endpoints

## Continuous Integration

### Validate Spec in CI

Add this to your CI pipeline to ensure the spec is valid:

```yaml
# GitHub Actions example
- name: Validate OpenAPI Spec
  run: |
    npm install -g @stoplight/spectral-cli
    spectral lint openapi.yaml
```

### Generate and Test Client

```yaml
# GitHub Actions example
- name: Generate Go Client
  run: |
    go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest
    oapi-codegen -package spokeclient -generate types,client openapi.yaml > client.go

- name: Verify Client Compiles
  run: go build client.go
```

### Breaking Change Detection

Use tools like `oasdiff` to detect breaking changes:

```bash
# Install oasdiff
go install github.com/tufin/oasdiff@latest

# Compare specs
oasdiff breaking openapi-old.yaml openapi.yaml
```

## Maintaining the Specification

### When Adding New Endpoints

When you add a new API endpoint to the code, update the OpenAPI spec:

1. Add the path and HTTP method to the `paths:` section
2. Define request/response schemas in `components/schemas`
3. Document query parameters, path parameters, and request bodies
4. Add appropriate error responses
5. Tag the endpoint with the appropriate category
6. Test the endpoint in Swagger UI

### When Modifying Endpoints

1. Update the path definition if parameters changed
2. Update request/response schemas if data structures changed
3. Update descriptions if behavior changed
4. Consider versioning if changes are breaking

### Spec Validation

Before committing changes, validate the spec:

```bash
# Lint the spec
spectral lint openapi.yaml

# Check for breaking changes (if you have the old spec)
oasdiff breaking openapi-old.yaml openapi.yaml

# Verify the spec builds
go build ./cmd/spoke
```

## Tools and Resources

### Recommended Tools

- **Spectral**: OpenAPI linting and validation
- **oapi-codegen**: Generate Go clients and servers
- **openapi-generator**: Generate clients in 40+ languages
- **Prism**: Mock server and validation proxy
- **oasdiff**: Breaking change detection
- **Redocly**: Alternative documentation UI
- **SwaggerHub**: Collaborative API design platform

### Resources

- [OpenAPI Specification 3.0](https://spec.openapis.org/oas/v3.0.3)
- [Swagger UI Documentation](https://swagger.io/tools/swagger-ui/)
- [OpenAPI Generator](https://openapi-generator.tech/)
- [Spectral Documentation](https://stoplight.io/open-source/spectral)
- [Best Practices](https://swagger.io/resources/articles/best-practices-in-api-design/)

## Troubleshooting

### Swagger UI Not Loading

1. Check that the server is running: `curl http://localhost:8080/openapi.yaml`
2. Check browser console for errors
3. Verify CDN access (Swagger UI loads from CDN)

### Authentication Issues

1. Ensure token format is correct: `Bearer <token>`
2. Verify token hasn't expired
3. Check token permissions/scopes

### Spec Validation Errors

1. Run `spectral lint openapi.yaml` to see detailed errors
2. Common issues:
   - Missing required fields
   - Invalid schema references
   - Incorrect response status codes
   - Missing example values

### Generated Client Issues

1. Verify the generator version is compatible
2. Check for custom type mappings needed
3. Review generator documentation for language-specific options

## Version History

- **v2.0.0** (2026-01-28): Initial OpenAPI 3.0 specification
  - 100+ endpoints documented
  - 12 API groups
  - Full authentication and error documentation
  - Swagger UI integration
