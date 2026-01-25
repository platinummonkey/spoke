# Schema Analytics Implementation Status

## Overview

This document tracks the implementation of comprehensive schema analytics for Spoke, enabling data-driven insights into schema usage, performance, and health.

**Original Plan**: 6 phases over 6 weeks
**Current Status**: Phases 1-2 complete, Phase 3 in progress

---

## ✅ Completed: Phase 1 - Event Tracking Infrastructure

**Files Created:**
- `migrations/007_analytics_events.up.sql` - Event tables migration
- `migrations/007_analytics_events.down.sql` - Rollback migration
- `pkg/analytics/events.go` - EventTracker service
- `pkg/analytics/helpers.go` - Request metadata extraction helpers

**Files Modified:**
- `pkg/api/handlers.go` - Added event tracking to download/view handlers

**Features:**
- ✅ Partitioned event tables (download_events, module_view_events, compilation_events)
- ✅ Monthly partitions (2026-01, 02, 03) with auto-creation support
- ✅ Indexes for fast time-range and analytics queries
- ✅ Non-blocking async event tracking (goroutines)
- ✅ User/org/IP/UA extraction from requests
- ✅ Download event tracking (module, version, language, file size, duration, success)
- ✅ Module view event tracking (source, page type, referrer)

**Database Tables:**
```sql
download_events (partitioned by downloaded_at)
  - user_id, organization_id, module_name, version, language
  - file_size, duration_ms, success, error_message
  - ip_address, user_agent, client_sdk, client_version, cache_hit

module_view_events (partitioned by viewed_at)
  - user_id, organization_id, module_name, version
  - source (web/api/cli), page_type (list/detail/search)
  - referrer, ip_address, user_agent

compilation_events (partitioned by started_at)
  - module_name, version, language
  - started_at, completed_at, duration_ms, success
  - error_message, error_type, cache_hit
  - file_count, output_size, compiler_version
```

---

## ✅ Completed: Phase 2 - Aggregation Infrastructure

**Files Created:**
- `migrations/008_analytics_aggregates.up.sql` - Aggregation tables migration
- `migrations/008_analytics_aggregates.down.sql` - Rollback migration
- `pkg/analytics/aggregator.go` - Aggregation service
- `cmd/spoke-aggregator/main.go` - Background job service
- `deployments/systemd/spoke-aggregator.service` - Systemd service file

**Features:**
- ✅ Daily/weekly/monthly aggregation tables (module_stats_*)
- ✅ Language compilation performance stats (language_stats_daily)
- ✅ Organization usage tracking (org_stats_daily)
- ✅ Materialized views: top_modules_30d, trending_modules
- ✅ Cron-based background aggregation (daily at 00:05 UTC)
- ✅ Hourly materialized view refresh
- ✅ Idempotent aggregation (safe to re-run)
- ✅ Manual aggregation support (--run-once --date=YYYY-MM-DD)

**Database Tables:**
```sql
module_stats_daily (date, module_name)
  - view_count, download_count, unique_users, unique_orgs
  - compilation_count, compilation_success_count
  - total_download_bytes, avg_compilation_duration_ms

module_stats_weekly (week_start, module_name)
  - Same metrics as daily, aggregated weekly

module_stats_monthly (month, module_name)
  - Same metrics as daily, aggregated monthly

language_stats_daily (date, language)
  - compilation_count, success_count, failure_count
  - avg_duration_ms, p50_duration_ms, p95_duration_ms, p99_duration_ms
  - total_output_bytes, cache_hit_count, cache_miss_count

org_stats_daily (date, organization_id)
  - api_requests, downloads, compilations
  - storage_bytes, active_users
  - modules_created, versions_created

Materialized Views:
  - top_modules_30d: Top 100 modules by downloads (last 30 days)
  - trending_modules: Top 50 by growth rate (7d vs previous 7d)
```

**Background Service:**
```bash
# Run aggregator in scheduled mode
spoke-aggregator --db-url="postgres://localhost/spoke"

# Run once for testing/backfilling
spoke-aggregator --run-once --date="2026-01-24"

# Deploy with systemd
sudo systemctl enable spoke-aggregator
sudo systemctl start spoke-aggregator
```

---

## 🚧 In Progress: Phase 3 - Analytics API

**Files Created:**
- `pkg/analytics/service.go` - Business logic service ✅

**Files To Create:**
- `pkg/api/analytics_handlers.go` - HTTP handlers
- `pkg/api/analytics_handlers_test.go` - Handler tests

**Features Completed:**
- ✅ GetOverview() - High-level KPIs (modules, versions, downloads, active users, cache hit rate)
- ✅ GetModuleStats() - Per-module analytics with time series
- ✅ GetPopularModules() - Top modules by downloads
- ✅ GetTrendingModules() - Modules with highest growth rate

**Features To Implement:**
- ⏳ HTTP handlers for analytics endpoints
- ⏳ Route registration in server.go
- ⏳ API documentation

**API Endpoints (To Implement):**
```
GET /api/v2/analytics/overview
GET /api/v2/analytics/modules/popular?period=30d&limit=100
GET /api/v2/analytics/modules/trending?limit=50
GET /api/v2/analytics/modules/{name}/stats?period=30d
GET /api/v2/analytics/modules/{name}/downloads?period=7d&groupBy=day
GET /api/v2/analytics/modules/{name}/health?version=v1.0.0
GET /api/v2/analytics/performance/compilation?language=go
GET /api/v2/analytics/languages
GET /api/v2/analytics/organizations/{id}/dashboard
```

---

## 📋 Remaining: Phase 4 - Health Scoring & Recommendations

**Files To Create:**
- `pkg/analytics/health.go` - Health scoring engine
- `pkg/analytics/health_test.go` - Health scoring tests

**Features To Implement:**
- Schema health scoring (0-100 scale)
- Complexity scoring (entity count, avg fields per message, nesting depth)
- Unused field detection (fields with no usage in 90 days)
- Deprecated field counting
- Breaking change frequency tracking
- Maintainability index calculation
- Actionable recommendations generation

**Health Scoring Algorithm:**
```
Health Score = weighted average of:
  - Complexity (25%): Lower is better
  - Maintainability (35%): Higher is better
  - Unused fields (15%): Fewer is better
  - Deprecated fields (10%): Fewer is better
  - Breaking changes (15%): Fewer is better

Recommendations:
  - Split large modules (complexity > 70)
  - Remove unused fields (>5 detected)
  - Clean up deprecations (>3 fields)
  - Reduce breaking changes (>2 per month)
  - Coordinate with dependents (high impact)
```

---

## 📋 Remaining: Phase 5 - Dashboard UI

**Frontend Dependencies:**
```bash
cd web
npm install recharts@^2.10.3
```

**Files To Create:**
- `web/src/components/analytics/AnalyticsDashboard.tsx` - Main dashboard
- `web/src/components/analytics/ModuleAnalytics.tsx` - Per-module analytics
- `web/src/components/analytics/DownloadChart.tsx` - Download trends (line chart)
- `web/src/components/analytics/LanguageChart.tsx` - Language distribution (pie chart)
- `web/src/components/analytics/TopModulesChart.tsx` - Popular modules (bar chart)
- `web/src/components/analytics/PerformanceChart.tsx` - Compilation metrics
- `web/src/hooks/useAnalytics.ts` - React Query hooks

**Files To Modify:**
- `web/src/components/ModuleDetail.tsx` - Add "Analytics" tab
- `web/src/App.tsx` - Add /analytics route
- `web/package.json` - Add recharts dependency

**UI Components:**
- KPI cards (modules, downloads, users, cache hit rate)
- Download trend line chart (time series)
- Language distribution pie chart
- Top modules bar chart
- Compilation performance chart
- Health score display with color-coded badges
- Recommendations list
- Unused fields warning alerts

---

## 📋 Remaining: Phase 6 - Polish & Production Readiness

**Performance Optimizations:**
- Redis caching for expensive queries (5-10 min TTL)
- Covering indexes for common analytics queries
- Query timeout enforcement (5 seconds)
- Response compression (gzip)
- X-Cache headers for debugging

**Alerting:**
- `pkg/analytics/alerts.go` - Alerting service
- `deployments/prometheus/analytics_alerts.yml` - Prometheus alerts
- Low health score alerts (<50)
- Slow compilation alerts (>5s p95)
- High error rate alerts (>10%)

**Documentation:**
- `docs/SCHEMA_ANALYTICS.md` - User guide
- `docs/HEALTH_SCORING.md` - Algorithm explanation
- `docs/API_REFERENCE.md` - Update with analytics endpoints
- Troubleshooting guide
- Performance tuning guide

**Testing:**
- Unit tests: `pkg/analytics/*_test.go`
- Integration tests: `tests/integration/analytics_test.go`
- Performance tests: `tests/performance/analytics_test.js` (k6)
- E2E tests: Cypress tests for dashboard

---

## Implementation Checklist

### Phase 1: Event Tracking ✅
- [x] Migration 007 (event tables)
- [x] EventTracker service
- [x] Request metadata helpers
- [x] Download event tracking
- [x] Module view event tracking
- [x] Compilation event tracking (needs sprocket integration)

### Phase 2: Aggregation ✅
- [x] Migration 008 (aggregation tables)
- [x] Aggregator service
- [x] Background job service (spoke-aggregator)
- [x] Systemd service file
- [x] Daily aggregation
- [x] Weekly aggregation
- [x] Monthly aggregation
- [x] Materialized view refresh

### Phase 3: Analytics API 🚧
- [x] Service layer (business logic)
- [ ] HTTP handlers
- [ ] Route registration
- [ ] API documentation
- [ ] Handler tests
- [ ] Integration tests

### Phase 4: Health Scoring 📋
- [ ] Health scoring engine
- [ ] Complexity calculation
- [ ] Unused field detection
- [ ] Deprecated field counting
- [ ] Breaking change tracking
- [ ] Maintainability index
- [ ] Recommendations generator
- [ ] Health scoring tests

### Phase 5: Dashboard UI 📋
- [ ] Install recharts
- [ ] Analytics dashboard component
- [ ] Module analytics component
- [ ] Chart components (download, language, top modules, performance)
- [ ] React Query hooks
- [ ] Route integration
- [ ] Mobile responsive design
- [ ] E2E tests

### Phase 6: Polish 📋
- [ ] Redis caching
- [ ] Performance optimizations
- [ ] Alerting system
- [ ] Prometheus alerts
- [ ] Documentation (user guide, API reference, health scoring)
- [ ] Unit tests
- [ ] Integration tests
- [ ] Performance tests (k6)
- [ ] Load testing

---

## Quick Start Guide

### 1. Apply Migrations

```bash
# Apply event tracking migration
psql -d spoke -f migrations/007_analytics_events.up.sql

# Apply aggregation migration
psql -d spoke -f migrations/008_analytics_aggregates.up.sql
```

### 2. Start Background Aggregator

```bash
# Build
go build -o spoke-aggregator cmd/spoke-aggregator/main.go

# Run (scheduled mode)
./spoke-aggregator --db-url="postgres://localhost/spoke?sslmode=disable"

# Or install as systemd service
sudo cp deployments/systemd/spoke-aggregator.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable spoke-aggregator
sudo systemctl start spoke-aggregator
```

### 3. Verify Event Tracking

```bash
# Check if events are being recorded
psql -d spoke -c "SELECT COUNT(*) FROM download_events WHERE downloaded_at >= NOW() - INTERVAL '1 hour';"
psql -d spoke -c "SELECT COUNT(*) FROM module_view_events WHERE viewed_at >= NOW() - INTERVAL '1 hour';"
```

### 4. Manual Aggregation (Testing)

```bash
# Aggregate yesterday's data
./spoke-aggregator --run-once --date="2026-01-24"

# Check aggregated stats
psql -d spoke -c "SELECT * FROM module_stats_daily WHERE date = '2026-01-24' ORDER BY download_count DESC LIMIT 10;"
```

### 5. Query Analytics

```go
// In your application code
service := analytics.NewService(db)
overview, err := service.GetOverview(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Total Downloads (30d): %d\n", overview.TotalDownloads30d)
fmt.Printf("Active Users (7d): %d\n", overview.ActiveUsers7d)
fmt.Printf("Cache Hit Rate: %.2f%%\n", overview.CacheHitRate * 100)
```

---

## Next Steps

To complete the implementation:

1. **Phase 3 Completion** (2-3 hours):
   - Create `pkg/api/analytics_handlers.go` with HTTP handlers
   - Register routes in `pkg/api/server.go`
   - Write handler tests
   - Update API documentation

2. **Phase 4: Health Scoring** (1 day):
   - Implement health scoring algorithm in `pkg/analytics/health.go`
   - Add API endpoint for health scores
   - Write tests

3. **Phase 5: Dashboard UI** (2-3 days):
   - Install recharts
   - Create dashboard components
   - Create chart components
   - Add React Query hooks
   - Integrate into existing UI

4. **Phase 6: Polish** (2-3 days):
   - Add Redis caching
   - Performance optimizations
   - Alerting system
   - Documentation
   - Testing (unit, integration, performance)

---

## Database Maintenance

### Partition Management

Create future partitions:
```sql
-- Add partition for April 2026
CREATE TABLE download_events_2026_04 PARTITION OF download_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE module_view_events_2026_04 PARTITION OF module_view_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE TABLE compilation_events_2026_04 PARTITION OF compilation_events
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
```

Archive old partitions:
```sql
-- Detach partition (data retained but not queried)
ALTER TABLE download_events DETACH PARTITION download_events_2025_01;

-- Drop partition (deletes data)
DROP TABLE download_events_2025_01;
```

### Performance Monitoring

```sql
-- Check table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
    AND tablename LIKE '%events%'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan AS index_scans,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
WHERE schemaname = 'public'
    AND tablename LIKE '%stats%'
ORDER BY idx_scan DESC;

-- Check slow queries
SELECT
    query,
    calls,
    total_time / 1000 AS total_seconds,
    mean_time / 1000 AS mean_seconds
FROM pg_stat_statements
WHERE query LIKE '%module_stats%'
ORDER BY mean_time DESC
LIMIT 10;
```

---

## Architecture Summary

```
┌─────────────────────────────────────────────────────────┐
│                     Client Requests                      │
│              (Download, View Module, Compile)            │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   API Handlers                           │
│         (downloadCompiled, getModule, etc.)             │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ├─► Serve Response
                       │
                       └─► Track Event (async goroutine)
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────┐
│                   Event Tables                           │
│     (download_events, module_view_events, etc.)         │
│              Partitioned by Time                         │
└──────────────────────┬──────────────────────────────────┘
                       │
                       │ (Aggregated by spoke-aggregator)
                       ▼
┌─────────────────────────────────────────────────────────┐
│              Aggregation Tables                          │
│  (module_stats_daily, language_stats_daily, etc.)       │
└──────────────────────┬──────────────────────────────────┘
                       │
                       ├─► Materialized Views
                       │   (top_modules_30d, trending_modules)
                       │
                       └─► Analytics API
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────┐
│                  Dashboard UI                            │
│         (React + Recharts + React Query)                │
└─────────────────────────────────────────────────────────┘
```

**Key Design Principles:**
- Non-blocking event tracking (don't slow down user requests)
- Pre-computed aggregates (fast dashboard queries)
- Time-based partitioning (manageable data lifecycle)
- Materialized views (complex analytics pre-computed)
- Idempotent aggregation (safe to re-run)
- Horizontal scalability (partition by time, shard by module)

---

## Success Metrics

Track these metrics to measure analytics implementation success:

**Data Collection:**
- Event tracking error rate < 0.1%
- Event insertion latency < 10ms (p95)
- Partition coverage (no gaps in time range)

**Aggregation:**
- Daily aggregation completion time < 5 minutes
- Aggregation success rate > 99.9%
- Materialized view refresh time < 30 seconds

**API Performance:**
- Overview endpoint latency < 500ms (p95)
- Module stats endpoint latency < 1s (p95)
- Cache hit rate > 80%

**User Engagement:**
- Dashboard daily active users (track adoption)
- Health score improvement over time
- Recommendation follow-through rate

---

## Troubleshooting

**Problem: Events not being recorded**
```bash
# Check if event tables exist
psql -d spoke -c "\dt *events*"

# Check for errors in logs
journalctl -u spoke -f | grep -i "event"

# Verify event tracker is initialized
# (Check Server struct has non-nil eventTracker)
```

**Problem: Aggregation not running**
```bash
# Check aggregator service status
sudo systemctl status spoke-aggregator

# Check aggregator logs
journalctl -u spoke-aggregator -f

# Verify cron schedule
# (Should see "Daily aggregation started" at 00:05 UTC)

# Manual aggregation
./spoke-aggregator --run-once --date="2026-01-24"
```

**Problem: Slow dashboard queries**
```bash
# Check if aggregations are up to date
psql -d spoke -c "SELECT MAX(date) FROM module_stats_daily;"

# Refresh materialized views
psql -d spoke -c "REFRESH MATERIALIZED VIEW CONCURRENTLY top_modules_30d;"
psql -d spoke -c "REFRESH MATERIALIZED VIEW CONCURRENTLY trending_modules;"

# Check index usage
psql -d spoke -c "SELECT * FROM pg_stat_user_indexes WHERE schemaname='public';"
```

---

This implementation provides a solid foundation for comprehensive schema analytics. The remaining phases (HTTP handlers, health scoring, dashboard UI, and polish) build on this foundation to deliver the full analytics experience.
