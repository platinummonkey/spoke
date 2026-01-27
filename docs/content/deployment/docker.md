---
title: "Docker Deployment"
weight: 1
---

# Docker Deployment

Deploy Spoke using Docker containers.

## Quick Start

### Single Container

```bash
docker run -d \
  --name spoke \
  -p 8080:8080 \
  -v /var/spoke/storage:/data \
  -e DATABASE_URL="postgres://spoke:password@postgres:5432/spoke?sslmode=disable" \
  -e STORAGE_DIR="/data" \
  spoke:latest
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: spoke
      POSTGRES_USER: spoke
      POSTGRES_PASSWORD: changeme
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U spoke"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass changeme
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5

  spoke:
    image: spoke:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://spoke:changeme@postgres:5432/spoke?sslmode=disable
      - REDIS_URL=redis://:changeme@redis:6379/0
      - STORAGE_DIR=/data
      - JWT_SECRET=your-secret-key
      - LOG_LEVEL=info
    volumes:
      - spoke_storage:/data
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    restart: unless-stopped

  sprocket:
    image: spoke:latest
    command: ["/usr/local/bin/sprocket", "-storage-dir", "/data", "-delay", "10s"]
    environment:
      - STORAGE_DIR=/data
      - LOG_LEVEL=info
    volumes:
      - spoke_storage:/data
    depends_on:
      - spoke
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
  spoke_storage:
```

Start services:

```bash
docker-compose up -d
```

Check status:

```bash
docker-compose ps
docker-compose logs -f spoke
```

## Building Docker Image

### Dockerfile

Create `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make protobuf-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN make build

# Runtime stage
FROM alpine:latest

RUN apk add --no-cache ca-certificates protobuf

# Create non-root user
RUN addgroup -g 1000 spoke && \
    adduser -D -u 1000 -G spoke spoke

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /build/spoke /usr/local/bin/spoke
COPY --from=builder /build/spoke-cli /usr/local/bin/spoke-cli
COPY --from=builder /build/sprocket /usr/local/bin/sprocket

# Copy migrations
COPY --from=builder /build/migrations /app/migrations

# Create data directory
RUN mkdir -p /data && chown -R spoke:spoke /data

USER spoke

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command
CMD ["/usr/local/bin/spoke", "-port", "8080", "-storage-dir", "/data"]
```

Build image:

```bash
docker build -t spoke:latest .
```

Tag and push:

```bash
docker tag spoke:latest myregistry/spoke:v1.0.0
docker push myregistry/spoke:v1.0.0
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_URL` | Redis connection string | Optional |
| `STORAGE_DIR` | Storage directory | `/data` |
| `JWT_SECRET` | JWT signing key | Required |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |
| `SSO_ENABLED` | Enable SSO | `false` |
| `RBAC_ENABLED` | Enable RBAC | `false` |

### Volume Mounts

| Path | Description |
|------|-------------|
| `/data` | Module storage directory |
| `/app/config` | Configuration files |
| `/app/migrations` | Database migrations |

## Production Setup

### With PostgreSQL and Redis

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: spoke
      POSTGRES_USER: spoke
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - spoke-network
    restart: always

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    networks:
      - spoke-network
    restart: always

  spoke:
    image: myregistry/spoke:v1.0.0
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://spoke:${POSTGRES_PASSWORD}@postgres:5432/spoke?sslmode=require
      REDIS_URL: redis://:${REDIS_PASSWORD}@redis:6379/0
      STORAGE_DIR: /data
      JWT_SECRET: ${JWT_SECRET}
      LOG_LEVEL: info
      LOG_FORMAT: json
      SSO_ENABLED: true
      RBAC_ENABLED: true
    volumes:
      - spoke_storage:/data
      - ./config:/app/config:ro
    networks:
      - spoke-network
    depends_on:
      - postgres
      - redis
    restart: always

  sprocket:
    image: myregistry/spoke:v1.0.0
    command: ["/usr/local/bin/sprocket", "-storage-dir", "/data"]
    environment:
      STORAGE_DIR: /data
      LOG_LEVEL: info
    volumes:
      - spoke_storage:/data
    networks:
      - spoke-network
    depends_on:
      - spoke
    restart: always

  nginx:
    image: nginx:alpine
    ports:
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    networks:
      - spoke-network
    depends_on:
      - spoke
    restart: always

networks:
  spoke-network:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
  spoke_storage:
```

### NGINX Configuration

Create `nginx.conf`:

```nginx
events {
    worker_connections 1024;
}

http {
    upstream spoke {
        server spoke:8080;
    }

    server {
        listen 443 ssl http2;
        server_name spoke.company.com;

        ssl_certificate /etc/nginx/ssl/cert.pem;
        ssl_certificate_key /etc/nginx/ssl/key.pem;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;

        client_max_body_size 100M;

        location / {
            proxy_pass http://spoke;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }
}
```

### Environment File

Create `.env`:

```bash
POSTGRES_PASSWORD=your-postgres-password
REDIS_PASSWORD=your-redis-password
JWT_SECRET=your-jwt-secret-key
```

Load environment:

```bash
docker-compose --env-file .env up -d
```

## Database Migrations

Run migrations on first startup:

```bash
docker-compose exec spoke /usr/local/bin/spoke migrate \
  -db-url "postgres://spoke:password@postgres:5432/spoke?sslmode=disable"
```

Or create init script:

```bash
#!/bin/bash
# init.sh

echo "Waiting for postgres..."
while ! pg_isready -h postgres -U spoke; do
  sleep 1
done

echo "Running migrations..."
/usr/local/bin/spoke migrate -db-url "$DATABASE_URL"

echo "Starting server..."
exec /usr/local/bin/spoke -port 8080 -storage-dir /data
```

Update Dockerfile:

```dockerfile
COPY init.sh /app/init.sh
RUN chmod +x /app/init.sh

CMD ["/app/init.sh"]
```

## Monitoring

### Health Checks

```bash
# Check spoke health
curl http://localhost:8080/health

# Check with Docker
docker-compose exec spoke wget -qO- http://localhost:8080/health
```

### Logs

```bash
# View logs
docker-compose logs -f spoke

# Specific service
docker-compose logs -f sprocket

# Last 100 lines
docker-compose logs --tail=100 spoke
```

### Metrics

Export metrics to Prometheus:

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    ports:
      - "9090:9090"
    networks:
      - spoke-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana_data:/var/lib/grafana
    networks:
      - spoke-network
```

## Backup and Restore

### Backup

```bash
# Backup PostgreSQL
docker-compose exec postgres pg_dump -U spoke spoke > spoke-backup.sql

# Backup storage
tar -czf spoke-storage-backup.tar.gz -C /var/lib/docker/volumes/spoke_spoke_storage/_data .
```

### Restore

```bash
# Restore PostgreSQL
docker-compose exec -T postgres psql -U spoke spoke < spoke-backup.sql

# Restore storage
tar -xzf spoke-storage-backup.tar.gz -C /var/lib/docker/volumes/spoke_spoke_storage/_data
```

## Troubleshooting

### Container Won't Start

Check logs:

```bash
docker-compose logs spoke
```

Common issues:
- Database connection failed
- Missing environment variables
- Port already in use

### Permission Issues

Ensure volumes have correct permissions:

```bash
docker-compose exec spoke chown -R spoke:spoke /data
```

### Out of Memory

Increase container memory:

```yaml
services:
  spoke:
    deploy:
      resources:
        limits:
          memory: 2G
        reservations:
          memory: 512M
```

## Next Steps

- [Kubernetes Deployment](/deployment/kubernetes/) - K8s setup
- [High Availability](/deployment/ha/) - HA configuration
- [Monitoring](/guides/monitoring/) - Observability setup
