#!/bin/bash
# generate-sdks.sh - Generate client SDKs from OpenAPI specification
#
# Usage:
#   ./scripts/generate-sdks.sh [go|python|all]
#
# This script generates client SDKs for the Spoke API from the OpenAPI specification.
#
# Requirements:
#   - oapi-codegen (for Go): go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
#   - openapi-generator (for Python): npm install -g @openapitools/openapi-generator-cli

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Directories
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
OPENAPI_SPEC="$ROOT_DIR/openapi.yaml"
OUTPUT_DIR="$ROOT_DIR/generated-clients"

# Check if OpenAPI spec exists
if [ ! -f "$OPENAPI_SPEC" ]; then
    echo -e "${RED}✗ OpenAPI spec not found at $OPENAPI_SPEC${NC}"
    exit 1
fi

echo -e "${GREEN}📋 Spoke API Client SDK Generator${NC}"
echo "=================================="
echo ""

# Parse arguments
GENERATE_GO=false
GENERATE_PYTHON=false

if [ $# -eq 0 ] || [ "$1" == "all" ]; then
    GENERATE_GO=true
    GENERATE_PYTHON=true
elif [ "$1" == "go" ]; then
    GENERATE_GO=true
elif [ "$1" == "python" ]; then
    GENERATE_PYTHON=true
else
    echo -e "${RED}✗ Invalid argument: $1${NC}"
    echo "Usage: $0 [go|python|all]"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate Go client
if [ "$GENERATE_GO" = true ]; then
    echo -e "${YELLOW}🔨 Generating Go client SDK...${NC}"

    # Check if oapi-codegen is installed
    if ! command -v oapi-codegen &> /dev/null; then
        echo -e "${RED}✗ oapi-codegen not found${NC}"
        echo "Install it with: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest"
        exit 1
    fi

    GO_OUTPUT_DIR="$OUTPUT_DIR/go"
    mkdir -p "$GO_OUTPUT_DIR"

    # Generate types and client
    echo "  Generating types and client code..."
    oapi-codegen -package spokeclient -generate types,client "$OPENAPI_SPEC" > "$GO_OUTPUT_DIR/spoke_client.go"

    # Create go.mod if it doesn't exist
    if [ ! -f "$GO_OUTPUT_DIR/go.mod" ]; then
        cd "$GO_OUTPUT_DIR"
        go mod init github.com/platinummonkey/spoke/client/go
        go mod tidy
        cd "$ROOT_DIR"
    fi

    # Create README
    cat > "$GO_OUTPUT_DIR/README.md" << 'EOF'
# Spoke API Go Client

Auto-generated Go client for the Spoke Protocol Registry API.

## Installation

```bash
go get github.com/platinummonkey/spoke/client/go
```

## Usage

```go
package main

import (
    "context"
    "fmt"

    spokeclient "github.com/platinummonkey/spoke/client/go"
)

func main() {
    // Create client
    client, err := spokeclient.NewClient("https://api.spoke.dev")
    if err != nil {
        panic(err)
    }

    // Set authentication token
    ctx := context.Background()
    // Add bearer token to context if needed

    // List modules
    resp, err := client.ListModules(ctx, &spokeclient.ListModulesParams{})
    if err != nil {
        panic(err)
    }

    fmt.Printf("Found %d modules\n", len(resp.JSON200.Modules))
}
```

## Authentication

The API uses Bearer token authentication. Add the token to requests:

```go
import "net/http"

// Create request with auth
req, _ := http.NewRequest("GET", "https://api.spoke.dev/modules", nil)
req.Header.Set("Authorization", "Bearer "+token)
```

## Generated Types

All API types are available in the package:
- `Module` - Module definition
- `Version` - Version definition
- `CompileRequest` - Compilation request
- `SearchResults` - Search results
- etc.

## API Documentation

See the full API documentation at https://api.spoke.dev/swagger-ui
EOF

    echo -e "${GREEN}✓ Go client generated at $GO_OUTPUT_DIR${NC}"
fi

# Generate Python client
if [ "$GENERATE_PYTHON" = true ]; then
    echo -e "${YELLOW}🔨 Generating Python client SDK...${NC}"

    # Check if openapi-generator is installed
    if ! command -v openapi-generator-cli &> /dev/null; then
        echo -e "${YELLOW}⚠ openapi-generator-cli not found, attempting to use npx...${NC}"
        if ! command -v npx &> /dev/null; then
            echo -e "${RED}✗ Neither openapi-generator-cli nor npx found${NC}"
            echo "Install with: npm install -g @openapitools/openapi-generator-cli"
            exit 1
        fi
        GENERATOR_CMD="npx @openapitools/openapi-generator-cli"
    else
        GENERATOR_CMD="openapi-generator-cli"
    fi

    PYTHON_OUTPUT_DIR="$OUTPUT_DIR/python"
    mkdir -p "$PYTHON_OUTPUT_DIR"

    echo "  Generating Python client code..."
    $GENERATOR_CMD generate \
        -i "$OPENAPI_SPEC" \
        -g python \
        -o "$PYTHON_OUTPUT_DIR" \
        --package-name spoke_client \
        --additional-properties=projectName=spoke-client,packageVersion=1.0.0

    # Create simplified README
    cat > "$PYTHON_OUTPUT_DIR/README-USAGE.md" << 'EOF'
# Spoke API Python Client

Auto-generated Python client for the Spoke Protocol Registry API.

## Installation

```bash
pip install -e ./generated-clients/python
```

Or directly:

```bash
cd generated-clients/python
python setup.py install
```

## Quick Start

```python
import spoke_client
from spoke_client.api import modules_api
from spoke_client.model import Module

# Configure API client
configuration = spoke_client.Configuration(
    host = "https://api.spoke.dev"
)

# Add authentication
configuration.access_token = "your-api-token"

# Create API client
with spoke_client.ApiClient(configuration) as api_client:
    # Create API instance
    api_instance = modules_api.ModulesApi(api_client)

    # List modules
    try:
        response = api_instance.list_modules()
        print(f"Found {len(response['modules'])} modules")
        for module in response['modules']:
            print(f"  - {module['name']}: {module['description']}")
    except spoke_client.ApiException as e:
        print(f"Exception: {e}")
```

## Authentication

Set the bearer token in configuration:

```python
configuration = spoke_client.Configuration(
    host = "https://api.spoke.dev",
    access_token = "your-bearer-token-here"
)
```

## API Documentation

Full API documentation: https://api.spoke.dev/swagger-ui
Generated client docs: See docs/ directory

## Examples

See examples/ directory for more usage examples.
EOF

    echo -e "${GREEN}✓ Python client generated at $PYTHON_OUTPUT_DIR${NC}"
fi

echo ""
echo -e "${GREEN}✓ SDK generation complete!${NC}"
echo ""
echo "Generated SDKs can be found in: $OUTPUT_DIR/"
echo ""
echo "Next steps:"
if [ "$GENERATE_GO" = true ]; then
    echo "  Go:     cd $OUTPUT_DIR/go && go build"
fi
if [ "$GENERATE_PYTHON" = true ]; then
    echo "  Python: pip install -e $OUTPUT_DIR/python"
fi
echo ""
