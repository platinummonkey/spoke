---
title: "Deployment"
weight: 6
bookFlatSection: false
bookCollapseSection: false
---

# Deployment

Production deployment guides for Spoke.

## Deployment Options

- [Docker Deployment](/deployment/docker/) - Container-based deployment
- [Kubernetes](/deployment/kubernetes/) - K8s deployment with Helm
- [Systemd Service](/deployment/systemd/) - Linux systemd service
- [High Availability](/deployment/ha/) - HA setup with load balancing
- [Production Best Practices](/deployment/best-practices/) - Security and optimization

## Quick Start

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -v /var/spoke/storage:/data \
  -e DATABASE_URL="postgres://..." \
  spoke:latest
```

### Docker Compose

```bash
curl -O https://raw.githubusercontent.com/platinummonkey/spoke/main/docker-compose.yml
docker-compose up -d
```

### Kubernetes

```bash
helm repo add spoke https://platinummonkey.github.io/spoke
helm install spoke spoke/spoke
```

## Requirements

### Minimum Requirements
- 2 CPU cores
- 2GB RAM
- 10GB storage
- PostgreSQL 12+

### Recommended Production
- 4 CPU cores
- 8GB RAM
- 100GB+ storage (depends on modules)
- PostgreSQL 14+
- Redis 6+
- Load balancer

## Next Steps

Choose a deployment method that fits your infrastructure.
