# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o spoke ./cmd/spoke

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget

# Create non-root user
RUN addgroup -g 1000 spoke && \
    adduser -D -u 1000 -G spoke spoke

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/spoke .

# Change ownership
RUN chown -R spoke:spoke /app

# Switch to non-root user
USER spoke

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=3 \
  CMD wget --spider -q http://localhost:9090/health/ready || exit 1

# Run the application
ENTRYPOINT ["./spoke"]
