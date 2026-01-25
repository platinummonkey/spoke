# OpenTelemetry Setup Guide for Spoke

This guide explains how to configure and use OpenTelemetry with Spoke for comprehensive observability.

## Overview

Spoke integrates OpenTelemetry to export:
- **Traces**: Request flows through the system
- **Metrics**: Performance and business metrics
- **Logs**: Structured logs with trace correlation

**Important**: Spoke only emits telemetry. You are responsible for:
- Running the OpenTelemetry Collector
- Configuring your own backends (Prometheus, Jaeger, Loki, etc.)
- Setting up dashboards and alerting

## Quick Start

### 1. Enable OpenTelemetry in Spoke

Set environment variables:

```bash
export SPOKE_OTEL_ENABLED=true
export SPOKE_OTEL_ENDPOINT=localhost:4317
export SPOKE_OTEL_SERVICE_NAME=spoke-registry
export SPOKE_OTEL_SERVICE_VERSION=1.0.0
export SPOKE_OTEL_INSECURE=true  # Use insecure gRPC (dev only)
```

### 2. Run OpenTelemetry Collector

Create `otel-collector-config.yaml`:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 5s
    send_batch_size: 512

  memory_limiter:
    check_interval: 1s
    limit_mib: 512

exporters:
  # Export metrics to Prometheus
  prometheus:
    endpoint: "0.0.0.0:8889"
    namespace: spoke

  # Export traces to Jaeger
  jaeger:
    endpoint: jaeger:14250
    tls:
      insecure: true

  # Export logs to Loki
  loki:
    endpoint: http://loki:3100/loki/api/v1/push

  # Debug exporter (logs to console)
  logging:
    loglevel: info

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [jaeger, logging]

    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [prometheus, logging]

    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [loki, logging]
```

Run the collector:

```bash
docker run -d \
  --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 8889:8889 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otel-collector-config.yaml \
  otel/opentelemetry-collector:latest \
  --config=/etc/otel-collector-config.yaml
```

### 3. Start Spoke

```bash
./spoke-server
```

Check logs for confirmation:
```
{"level":"INFO","message":"OpenTelemetry initialized successfully"}
```

## Configuration Options

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SPOKE_OTEL_ENABLED` | `false` | Enable OpenTelemetry |
| `SPOKE_OTEL_ENDPOINT` | `localhost:4317` | OTLP gRPC endpoint |
| `SPOKE_OTEL_SERVICE_NAME` | `spoke-registry` | Service name in traces |
| `SPOKE_OTEL_SERVICE_VERSION` | `1.0.0` | Service version |
| `SPOKE_OTEL_INSECURE` | `true` | Use insecure gRPC (dev only) |

### Secure gRPC (Production)

For production, use TLS:

```bash
export SPOKE_OTEL_INSECURE=false
```

Ensure your collector has valid TLS certificates.

## Connecting to Backends

### Prometheus

The OpenTelemetry Collector exports metrics to Prometheus format.

**Prometheus scrape config** (`prometheus.yml`):

```yaml
scrape_configs:
  - job_name: 'spoke-otel'
    static_configs:
      - targets: ['otel-collector:8889']
```

**Key Metrics**:
- `http_server_requests` - HTTP request counter
- `http_server_duration` - Request duration histogram
- `db_connections_active` - Active database connections
- `db_query_duration` - Database query duration
- `cache_hits_total` / `cache_misses_total` - Cache performance
- `storage_operations_total` - Storage operation counter

### Jaeger (Traces)

View traces at: http://localhost:16686

**Trace Attributes**:
- `http.method`, `http.route`, `http.status_code`
- `db.operation`, `db.statement`
- `storage.operation`, `storage.type`

### Loki (Logs)

Query logs in Grafana with Loki data source.

**Log Correlation**:
All logs include `trace_id` and `span_id` for correlation with traces.

Example LogQL query:
```logql
{service_name="spoke-registry"} |= "error"
```

## Grafana Dashboards

### SLA Dashboard

**99.9% Availability Calculation**:

```promql
# Success rate over 1 hour
sum(rate(http_server_requests{status_code!~"5.."}[1h])) /
sum(rate(http_server_requests[1h]))

# SLA target: >= 0.999 (99.9%)
```

**Request Latency** (p95, p99):

```promql
# p95 latency
histogram_quantile(0.95,
  rate(http_server_duration_bucket[5m])
)

# p99 latency
histogram_quantile(0.99,
  rate(http_server_duration_bucket[5m])
)
```

### Database Health Dashboard

```promql
# Active connections
db_connections_active

# Query duration p95
histogram_quantile(0.95,
  rate(db_query_duration_bucket[5m])
)

# Query error rate
sum(rate(db_queries_total{error="true"}[5m]))
```

### Cache Performance Dashboard

```promql
# Cache hit rate
sum(rate(cache_hits_total[5m])) /
(sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m])))

# Cache size
cache_size
```

## Alert Rules

### Critical Alerts

**High Error Rate**:

```yaml
- alert: HighErrorRate
  expr: |
    sum(rate(http_server_requests{status_code=~"5.."}[5m])) /
    sum(rate(http_server_requests[5m])) > 0.01
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High error rate (>1%)"
```

**High Latency**:

```yaml
- alert: HighLatency
  expr: |
    histogram_quantile(0.99,
      rate(http_server_duration_bucket[5m])
    ) > 1.0
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "p99 latency > 1s"
```

**Database Connection Pool Exhausted**:

```yaml
- alert: DBPoolExhausted
  expr: db_connections_active >= db_connections_max * 0.9
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Database connection pool at 90% capacity"
```

### Warning Alerts

**Low Cache Hit Rate**:

```yaml
- alert: LowCacheHitRate
  expr: |
    sum(rate(cache_hits_total[5m])) /
    (sum(rate(cache_hits_total[5m])) + sum(rate(cache_misses_total[5m]))) < 0.5
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Cache hit rate below 50%"
```

## Troubleshooting

### Telemetry Not Appearing

**Check Spoke logs**:
```bash
{"level":"INFO","message":"OpenTelemetry initialized successfully"}
```

If you see errors, check:
1. Collector endpoint is reachable: `telnet localhost 4317`
2. Collector logs: `docker logs otel-collector`
3. Firewall rules allow gRPC traffic

**Test collector connectivity**:
```bash
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[]}'
```

### High Cardinality Metrics

If you see memory issues with Prometheus:

1. **Limit label values**: Configure collector to drop high-cardinality labels
2. **Increase retention**: Adjust Prometheus `--storage.tsdb.retention.time`
3. **Add recording rules**: Pre-aggregate metrics

### Trace Sampling

To reduce trace volume, configure the collector:

```yaml
processors:
  probabilistic_sampler:
    sampling_percentage: 10  # Sample 10% of traces

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [probabilistic_sampler, batch]
      exporters: [jaeger]
```

## Production Deployment

### Collector High Availability

Run multiple collector instances behind a load balancer:

```yaml
# Spoke instances → Load Balancer → Collector instances
SPOKE_OTEL_ENDPOINT=collector-lb:4317
```

### Collector Persistence

Use persistent storage for buffering:

```yaml
exporters:
  file:
    path: /var/otel/data
    rotation:
      max_megabytes: 512
      max_backups: 3
```

### Resource Limits

Set appropriate limits in Kubernetes:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "1000m"
```

## Example: Complete Stack with Docker Compose

See `deployments/docker-compose/ha-stack.yml` for a complete example with:
- 3 Spoke instances
- OpenTelemetry Collector
- Prometheus (optional)
- Jaeger (optional)
- Grafana (optional)

```bash
cd deployments/docker-compose
docker-compose -f ha-stack.yml up
```

## Security Considerations

### TLS

Always use TLS in production:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        tls:
          cert_file: /certs/collector.crt
          key_file: /certs/collector.key
```

### Authentication

Use bearer token authentication:

```yaml
extensions:
  bearertokenauth:
    token: "your-secret-token"

receivers:
  otlp:
    protocols:
      grpc:
        auth:
          authenticator: bearertokenauth
```

### Network Isolation

Run collector in a private network:
- Spoke → Collector: Internal network only
- Collector → Backends: VPN or private peering

## Further Reading

- [OpenTelemetry Collector Docs](https://opentelemetry.io/docs/collector/)
- [Spoke Metrics Reference](./metrics-reference.md)
- [Trace Attributes Reference](./trace-attributes.md)
- [Example Dashboards](./dashboards/)
