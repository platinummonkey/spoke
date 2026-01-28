.PHONY: all clean spoke spoke-server sprocket build-all

BINDIR := bin
CMDDIR := cmd

all: build-all

build-all: spoke spoke-server sprocket

spoke-server:
	@echo "Building spoke server..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/spoke-server $(CMDDIR)/spoke/main.go

spoke:
	@echo "Building spoke-cli tool..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/spoke $(CMDDIR)/spoke-cli/main.go

sprocket:
	@echo "Building sprocket watcher..."
	@mkdir -p $(BINDIR)
	go build -o $(BINDIR)/sprocket $(CMDDIR)/sprocket/*.go

clean:
	@echo "Cleaning up..."
	rm -rf $(BINDIR)

test:
	go test -v ./...

# OpenAPI/Swagger targets
.PHONY: openapi-validate openapi-serve openapi-diff openapi-gen-client

openapi-validate:
	@echo "Validating OpenAPI specification..."
	@which spectral > /dev/null || (echo "spectral not found. Install: npm install -g @stoplight/spectral-cli" && exit 1)
	spectral lint openapi.yaml

openapi-serve:
	@echo "Starting Swagger UI server (requires spoke-server to be running)..."
	@echo "Access Swagger UI at: http://localhost:8080/swagger-ui"
	@echo "OpenAPI spec at: http://localhost:8080/openapi.yaml"

openapi-diff:
	@echo "Comparing OpenAPI specifications for breaking changes..."
	@which oasdiff > /dev/null || (echo "oasdiff not found. Install: go install github.com/tufin/oasdiff@latest" && exit 1)
	@if [ ! -f openapi-old.yaml ]; then echo "openapi-old.yaml not found. Create it first."; exit 1; fi
	oasdiff breaking openapi-old.yaml openapi.yaml

openapi-gen-client:
	@echo "Generating Go client from OpenAPI spec..."
	@which oapi-codegen > /dev/null || (echo "oapi-codegen not found. Install: go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen@latest" && exit 1)
	@mkdir -p client
	oapi-codegen -package spokeclient -generate types,client openapi.yaml > client/spoke_client.go
	@echo "Client generated at: client/spoke_client.go"

.DEFAULT_GOAL := all 