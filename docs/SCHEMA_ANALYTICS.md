# Schema Analytics Guide

## Overview

Spoke's Schema Analytics provides comprehensive insights into:
- **Usage patterns** (downloads, views, popular modules)
- **Performance metrics** (compilation times, cache hit rates)
- **Schema health** (complexity, maintainability, unused fields)
- **Optimization recommendations** for improving schema quality

## Dashboard Access

**Web UI:** Navigate to the Analytics tab in the Spoke Registry web interface

**Routes:**
- Global Dashboard: `/analytics`
- Per-Module Analytics: Module detail page → Analytics tab

**API Endpoints:**
- `GET /api/v2/analytics/overview` - High-level KPIs
- `GET /api/v2/analytics/modules/popular` - Top 100 modules
- `GET /api/v2/analytics/modules/{name}/stats` - Per-module analytics
- `GET /api/v2/analytics/modules/{name}/health` - Schema health assessment

## Health Scoring

Health scores range from 0-100:
- **80-100**: Excellent - Well-designed, maintainable schema
- **60-79**: Good - Minor improvements recommended
- **40-59**: Fair - Several issues to address
- **0-39**: Poor - Requires immediate attention

### Factors Affecting Health:

The health score is calculated using a weighted multi-factor algorithm:

1. **Complexity (25%)**: Entity count, average fields per message, nesting depth
   - Lower complexity = better score
   - Large schemas (>50 entities) or messages with many fields (>15) are penalized

2. **Maintainability (35%)**: Code quality, deprecations, breaking changes
   - Higher maintainability = better score
   - Frequent breaking changes reduce maintainability

3. **Unused Fields (15%)**: Fields with no recorded usage in 90 days
   - Fewer unused fields = better score
   - >5 unused fields triggers penalty

4. **Deprecated Fields (10%)**: Fields marked as deprecated in proto
   - Fewer deprecated fields = better score
   - >3 deprecated fields triggers penalty

5. **Breaking Changes (15%)**: Breaking changes in last 30 days
   - Fewer breaking changes = better score
   - >2 breaking changes per month triggers penalty

### Health Score Calculation Example:

```
Module: user-service
- Complexity: 45/100 (moderate complexity)
- Maintainability: 82/100 (good maintainability)
- Unused Fields: 3 (penalty: 6 points)
- Deprecated Fields: 1 (penalty: 3 points)
- Breaking Changes: 0 (no penalty)

Health Score = 0.25*(100-45) + 0.35*82 + 0.15*(100-6) + 0.10*(100-3) + 0.15*100
             = 13.75 + 28.7 + 14.1 + 9.7 + 15
             = 81.25 → 81/100 (Excellent)
```

## Optimization Recommendations

The system provides actionable recommendations based on detected issues:

### High Complexity (>70)
**Recommendation**: "Consider splitting this module into smaller, focused modules to reduce complexity."

**Why**: Large, complex schemas are harder to maintain, test, and evolve. Breaking them into smaller modules improves maintainability and allows independent versioning.

**Example Action**: Split a 100-entity `services.proto` into domain-specific modules like `user-service.proto`, `payment-service.proto`, `notification-service.proto`.

### Unused Fields (>5)
**Recommendation**: "Remove unused fields to simplify the schema and reduce maintenance burden."

**Why**: Fields that haven't been accessed in 90+ days likely indicate dead code or abandoned features. Removing them reduces schema complexity and cognitive load.

**Example Action**:
```protobuf
// Before
message User {
  string id = 1;
  string name = 2;
  string legacy_field = 3;  // Unused for 120 days
}

// After
message User {
  string id = 1;
  string name = 2;
}
```

### Deprecated Fields (>3)
**Recommendation**: "Remove deprecated fields in the next major version to clean up technical debt."

**Why**: Accumulated deprecated fields increase schema size and complicate understanding. Major version bumps are the appropriate time to remove them.

**Example Action**:
```protobuf
// v1.x.x - Deprecate
message Config {
  string name = 1;
  string old_setting = 2 [deprecated = true];
  string new_setting = 3;
}

// v2.0.0 - Remove
message Config {
  string name = 1;
  string new_setting = 2;  // Renumber fields
}
```

### Frequent Breaking Changes (>2 in 30 days)
**Recommendation**: "Frequent breaking changes detected. Consider backward-compatible changes or better versioning."

**Why**: Breaking changes force all dependents to update simultaneously, which can be disruptive in distributed systems.

**Example Action**: Use field additions instead of removals, or introduce new optional fields:
```protobuf
// Bad: Breaking change
message User {
  string id = 1;
  // Removed: string name = 2; (BREAKING!)
  string full_name = 3;  // New field
}

// Good: Backward compatible
message User {
  string id = 1;
  string name = 2 [deprecated = true];  // Keep for compatibility
  string full_name = 3;  // Add new field
}
```

### High Impact Modules (>10 dependents + breaking changes)
**Recommendation**: "This module has {count} dependents and {changes} breaking change(s) in the last 30 days. Coordinate changes carefully with downstream consumers."

**Why**: Changes to widely-used modules have cascading effects. Communication and coordination are essential.

**Example Action**:
1. Announce breaking changes in advance
2. Provide migration guides
3. Consider gradual rollout with dual support periods

## Performance Monitoring

### Compilation Metrics:

**p50/p95/p99 latency** by language
- Percentile-based latency tracking
- Identifies slow compilation targets
- Alerts trigger when p95 > 5 seconds

**Success rate** by language
- Tracks compilation failures
- Helps identify problematic schemas or compiler issues

**Cache hit rate** (target: >70%)
- Measures effectiveness of compilation caching
- Low hit rates indicate cache misses (frequent schema changes or cache eviction)

**Output size trends**
- Tracks compiled artifact sizes
- Helps identify bloated generated code

### API Metrics:

**Request count** by endpoint
- Tracks API usage patterns
- Identifies hotspots

**Response time** percentiles
- Monitors API performance
- Alerts on slow queries

**Error rates** by endpoint
- Tracks API failures
- Helps identify problematic endpoints

## Usage Analytics

### Module Popularity:

**View counts** (24h, 7d, 30d)
- Tracks module page views
- Indicates user interest

**Download counts** by language
- Shows actual usage by SDK
- Helps prioritize SDK support

**Unique users/organizations**
- Measures adoption breadth
- Identifies key users

### Trending Modules:

**Growth rate**: Current week vs previous week
- Formula: `(current_downloads - previous_downloads) / previous_downloads`
- Minimum: 10 downloads to qualify
- Sorted by growth rate descending

**Example**:
```
Module: auth-service
Previous week: 50 downloads
Current week: 80 downloads
Growth rate: (80-50)/50 = 0.6 = +60%
```

### Version Adoption:

**Most popular versions**
- Tracks which versions are actively used
- Helps plan deprecation schedules

**Version lifecycle**
- Time between releases
- Version longevity before replacement

## Best Practices

### 1. Review Health Scores Monthly

Set a calendar reminder to review analytics:
- Identify declining health scores early
- Address issues before they impact dependents
- Track improvement over time

### 2. Monitor Trending Modules

Trending modules indicate:
- Growing adoption (optimize performance)
- New use cases (consider feature additions)
- Community interest (prioritize documentation)

### 3. Track Compilation Performance

Watch for performance regressions:
- Slow compilations frustrate users
- High failure rates indicate schema issues
- Cache hit rate <70% suggests optimization opportunities

### 4. Remove Unused Fields

Regular cleanup prevents accumulation:
- Review unused fields quarterly
- Deprecate in minor versions
- Remove in major versions

### 5. Plan Breaking Changes

Use dependency count to assess impact:
- <5 dependents: Low impact, proceed with caution
- 5-10 dependents: Medium impact, coordinate with teams
- >10 dependents: High impact, require migration period

## API Examples

### Get Overview KPIs:
```bash
curl https://spoke.example.com/api/v2/analytics/overview
```

**Response**:
```json
{
  "total_modules": 1523,
  "total_versions": 8945,
  "total_downloads_24h": 12453,
  "total_downloads_7d": 78234,
  "total_downloads_30d": 342567,
  "active_users_24h": 145,
  "active_users_7d": 567,
  "top_language": "go",
  "avg_compilation_ms": 1245.6,
  "cache_hit_rate": 0.78
}
```

### Get Module Stats:
```bash
curl "https://spoke.example.com/api/v2/analytics/modules/user-service/stats?period=30d"
```

**Response**:
```json
{
  "module_name": "user-service",
  "total_views": 1245,
  "total_downloads": 892,
  "unique_users": 67,
  "downloads_by_day": [
    {"date": "2026-01-01", "value": 34},
    {"date": "2026-01-02", "value": 42}
  ],
  "downloads_by_language": {
    "go": 523,
    "python": 289,
    "java": 80
  },
  "popular_versions": [
    {"version": "v1.2.0", "downloads": 456},
    {"version": "v1.1.0", "downloads": 234}
  ],
  "avg_compilation_time_ms": 1024,
  "compilation_success_rate": 0.98
}
```

### Get Module Health:
```bash
curl "https://spoke.example.com/api/v2/analytics/modules/user-service/health?version=v1.2.0"
```

**Response**:
```json
{
  "module_name": "user-service",
  "version": "v1.2.0",
  "health_score": 81.2,
  "complexity_score": 45.0,
  "maintainability_index": 82.0,
  "unused_fields": ["legacy_field", "old_setting"],
  "deprecated_fields": 1,
  "breaking_changes_30d": 0,
  "dependents_count": 8,
  "recommendations": [
    "Schema health is excellent! Keep following protobuf best practices.",
    "Remove unused fields to simplify the schema and reduce maintenance burden."
  ]
}
```

### Get Popular Modules:
```bash
curl "https://spoke.example.com/api/v2/analytics/modules/popular?period=30d&limit=10"
```

**Response**:
```json
[
  {
    "module_name": "common-types",
    "total_views": 5678,
    "total_downloads": 3456,
    "active_days": 30,
    "avg_daily_downloads": 115.2
  },
  {
    "module_name": "user-service",
    "total_views": 1245,
    "total_downloads": 892,
    "active_days": 28,
    "avg_daily_downloads": 31.9
  }
]
```

### Get Trending Modules:
```bash
curl https://spoke.example.com/api/v2/analytics/modules/trending
```

**Response**:
```json
[
  {
    "module_name": "auth-service",
    "current_downloads": 80,
    "previous_downloads": 50,
    "growth_rate": 0.6
  },
  {
    "module_name": "payment-service",
    "current_downloads": 45,
    "previous_downloads": 30,
    "growth_rate": 0.5
  }
]
```

## Troubleshooting

### No Data Showing

**Symptoms**: Dashboard shows zeros or "No data available"

**Causes**:
1. Aggregation jobs not running
2. No event data collected yet
3. Database connection issues

**Solutions**:
```bash
# Check if aggregator is running
systemctl status spoke-aggregator

# Check last aggregation time
psql spoke -c "SELECT MAX(created_at) FROM module_stats_daily"

# Manually run aggregation
spoke-aggregator --run-once --date=$(date -d yesterday +%Y-%m-%d)

# Check event collection
psql spoke -c "SELECT COUNT(*) FROM download_events WHERE downloaded_at >= NOW() - INTERVAL '24 hours'"
```

### Inaccurate Health Scores

**Symptoms**: Health scores don't reflect actual schema quality

**Causes**:
1. Scores computed daily (not real-time)
2. Materialized views not refreshed
3. Missing event data

**Solutions**:
```bash
# Manually refresh materialized views
psql spoke -c "REFRESH MATERIALIZED VIEW CONCURRENTLY top_modules_30d"
psql spoke -c "REFRESH MATERIALIZED VIEW CONCURRENTLY trending_modules"

# Re-run health calculations (if implemented)
curl -X POST https://spoke.example.com/api/v2/analytics/health/recompute
```

### Slow Dashboard

**Symptoms**: Dashboard takes >5 seconds to load

**Causes**:
1. Missing indexes
2. Large aggregation tables
3. Expensive queries

**Solutions**:
```bash
# Apply performance indexes migration
psql spoke -f migrations/009_analytics_performance_indexes.up.sql

# Check query plans
psql spoke -c "EXPLAIN ANALYZE SELECT * FROM module_stats_daily WHERE date >= CURRENT_DATE - INTERVAL '30 days'"

# Partition old data
psql spoke -c "SELECT * FROM download_events WHERE downloaded_at < '2025-01-01' LIMIT 1"  # Check if old partitions exist

# Consider shorter time ranges
# Instead of: period=90d
# Use: period=30d
```

### Alerts Not Triggering

**Symptoms**: Known issues not generating alerts

**Causes**:
1. Alert thresholds too high
2. Alerter not running
3. No notification integration

**Solutions**:
```bash
# Check alerter schedule
systemctl status spoke-aggregator
journalctl -u spoke-aggregator -f  # Watch logs

# Manually trigger alerts
spoke-aggregator --alert-check-only  # (if flag exists)

# Adjust thresholds
# Edit spoke-aggregator flags:
# --health-threshold=60  (default: 50)
# --perf-threshold=3000  (default: 5000ms)
```

## Maintenance Tasks

### Daily:
- Aggregation jobs run automatically (00:05 UTC)
- Event partitions managed automatically

### Weekly:
- Review alert logs for recurring issues
- Check for slow-running aggregation jobs

### Monthly:
- Review health scores for all modules
- Clean up old partitions (>12 months)
- Archive historical analytics data

### Quarterly:
- Analyze trending patterns
- Adjust alert thresholds based on baseline
- Review and update recommendations

## Architecture Reference

### Database Tables:

**Event Tables** (partitioned monthly):
- `download_events` - Download tracking
- `module_view_events` - View tracking
- `compilation_events` - Compilation tracking

**Aggregation Tables**:
- `module_stats_daily` - Daily module statistics
- `module_stats_weekly` - Weekly aggregates
- `module_stats_monthly` - Monthly aggregates
- `language_stats_daily` - Language compilation stats
- `org_stats_daily` - Organization usage

**Materialized Views**:
- `top_modules_30d` - Top 100 modules (refreshed hourly)
- `trending_modules` - Growth rate rankings (refreshed hourly)

### Background Jobs:

**spoke-aggregator** service:
- Daily aggregation: 00:05 UTC
- Materialized view refresh: Every hour
- Alert checks: Every 6 hours

### API Endpoints:

- `/api/v2/analytics/overview` - Global KPIs
- `/api/v2/analytics/modules/popular` - Popular modules
- `/api/v2/analytics/modules/trending` - Trending modules
- `/api/v2/analytics/modules/{name}/stats` - Per-module stats
- `/api/v2/analytics/modules/{name}/health` - Health scoring

## Support

For issues or questions about Schema Analytics:

1. Check this documentation
2. Review troubleshooting section
3. Check application logs: `journalctl -u spoke-aggregator -f`
4. File an issue on GitHub with:
   - Dashboard screenshot
   - Query that's failing
   - Error logs
   - Expected vs actual behavior
