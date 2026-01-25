package analytics

import (
	"context"
	"database/sql"
	"time"
)

// Aggregator computes daily/weekly/monthly statistics
type Aggregator struct {
	db *sql.DB
}

// NewAggregator creates a new aggregator
func NewAggregator(db *sql.DB) *Aggregator {
	return &Aggregator{db: db}
}

// AggregateModuleStatsDaily computes daily stats for all modules
func (a *Aggregator) AggregateModuleStatsDaily(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO module_stats_daily (
			module_name, date,
			view_count, download_count, unique_users, unique_orgs,
			compilation_count, compilation_success_count,
			total_download_bytes, avg_compilation_duration_ms
		)
		SELECT
			m.name AS module_name,
			$1::date AS date,
			COALESCE(COUNT(DISTINCT v.id), 0) AS view_count,
			COALESCE(COUNT(DISTINCT d.id), 0) AS download_count,
			COALESCE(COUNT(DISTINCT COALESCE(v.user_id, d.user_id)), 0) AS unique_users,
			COALESCE(COUNT(DISTINCT COALESCE(v.organization_id, d.organization_id)), 0) AS unique_orgs,
			COALESCE(COUNT(DISTINCT c.id), 0) AS compilation_count,
			COALESCE(COUNT(DISTINCT CASE WHEN c.success THEN c.id END), 0) AS compilation_success_count,
			COALESCE(SUM(d.file_size), 0) AS total_download_bytes,
			AVG(c.duration_ms)::integer AS avg_compilation_duration_ms
		FROM modules m
		LEFT JOIN module_view_events v ON m.name = v.module_name
			AND v.viewed_at >= $1::date
			AND v.viewed_at < $1::date + INTERVAL '1 day'
		LEFT JOIN download_events d ON m.name = d.module_name
			AND d.downloaded_at >= $1::date
			AND d.downloaded_at < $1::date + INTERVAL '1 day'
		LEFT JOIN compilation_events c ON m.name = c.module_name
			AND c.started_at >= $1::date
			AND c.started_at < $1::date + INTERVAL '1 day'
		GROUP BY m.name
		ON CONFLICT (module_name, date) DO UPDATE SET
			view_count = EXCLUDED.view_count,
			download_count = EXCLUDED.download_count,
			unique_users = EXCLUDED.unique_users,
			unique_orgs = EXCLUDED.unique_orgs,
			compilation_count = EXCLUDED.compilation_count,
			compilation_success_count = EXCLUDED.compilation_success_count,
			total_download_bytes = EXCLUDED.total_download_bytes,
			avg_compilation_duration_ms = EXCLUDED.avg_compilation_duration_ms
	`
	_, err := a.db.ExecContext(ctx, query, date)
	return err
}

// AggregateModuleStatsWeekly computes weekly stats for all modules
func (a *Aggregator) AggregateModuleStatsWeekly(ctx context.Context, weekStart time.Time) error {
	weekEnd := weekStart.AddDate(0, 0, 7)

	query := `
		INSERT INTO module_stats_weekly (
			module_name, week_start, week_end,
			view_count, download_count, unique_users, unique_orgs,
			compilation_count, compilation_success_count,
			total_download_bytes, avg_compilation_duration_ms
		)
		SELECT
			module_name,
			$1::date AS week_start,
			$2::date AS week_end,
			SUM(view_count) AS view_count,
			SUM(download_count) AS download_count,
			SUM(unique_users) AS unique_users,
			SUM(unique_orgs) AS unique_orgs,
			SUM(compilation_count) AS compilation_count,
			SUM(compilation_success_count) AS compilation_success_count,
			SUM(total_download_bytes) AS total_download_bytes,
			AVG(avg_compilation_duration_ms)::integer AS avg_compilation_duration_ms
		FROM module_stats_daily
		WHERE date >= $1::date AND date < $2::date
		GROUP BY module_name
		ON CONFLICT (module_name, week_start) DO UPDATE SET
			view_count = EXCLUDED.view_count,
			download_count = EXCLUDED.download_count,
			unique_users = EXCLUDED.unique_users,
			unique_orgs = EXCLUDED.unique_orgs,
			compilation_count = EXCLUDED.compilation_count,
			compilation_success_count = EXCLUDED.compilation_success_count,
			total_download_bytes = EXCLUDED.total_download_bytes,
			avg_compilation_duration_ms = EXCLUDED.avg_compilation_duration_ms
	`
	_, err := a.db.ExecContext(ctx, query, weekStart, weekEnd)
	return err
}

// AggregateModuleStatsMonthly computes monthly stats for all modules
func (a *Aggregator) AggregateModuleStatsMonthly(ctx context.Context, month time.Time) error {
	// month should be first day of month
	nextMonth := month.AddDate(0, 1, 0)

	query := `
		INSERT INTO module_stats_monthly (
			module_name, month,
			view_count, download_count, unique_users, unique_orgs,
			compilation_count, compilation_success_count,
			total_download_bytes, avg_compilation_duration_ms
		)
		SELECT
			module_name,
			$1::date AS month,
			SUM(view_count) AS view_count,
			SUM(download_count) AS download_count,
			SUM(unique_users) AS unique_users,
			SUM(unique_orgs) AS unique_orgs,
			SUM(compilation_count) AS compilation_count,
			SUM(compilation_success_count) AS compilation_success_count,
			SUM(total_download_bytes) AS total_download_bytes,
			AVG(avg_compilation_duration_ms)::integer AS avg_compilation_duration_ms
		FROM module_stats_daily
		WHERE date >= $1::date AND date < $2::date
		GROUP BY module_name
		ON CONFLICT (module_name, month) DO UPDATE SET
			view_count = EXCLUDED.view_count,
			download_count = EXCLUDED.download_count,
			unique_users = EXCLUDED.unique_users,
			unique_orgs = EXCLUDED.unique_orgs,
			compilation_count = EXCLUDED.compilation_count,
			compilation_success_count = EXCLUDED.compilation_success_count,
			total_download_bytes = EXCLUDED.total_download_bytes,
			avg_compilation_duration_ms = EXCLUDED.avg_compilation_duration_ms
	`
	_, err := a.db.ExecContext(ctx, query, month, nextMonth)
	return err
}

// AggregateLanguageStatsDaily computes daily compilation stats per language
func (a *Aggregator) AggregateLanguageStatsDaily(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO language_stats_daily (
			language, date,
			compilation_count, compilation_success_count, compilation_failure_count,
			avg_duration_ms, p50_duration_ms, p95_duration_ms, p99_duration_ms,
			total_output_bytes, cache_hit_count, cache_miss_count
		)
		SELECT
			language,
			$1::date AS date,
			COUNT(*) AS compilation_count,
			COUNT(*) FILTER (WHERE success) AS compilation_success_count,
			COUNT(*) FILTER (WHERE NOT success) AS compilation_failure_count,
			AVG(duration_ms)::integer AS avg_duration_ms,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY duration_ms)::integer AS p50_duration_ms,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms)::integer AS p95_duration_ms,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms)::integer AS p99_duration_ms,
			SUM(output_size) AS total_output_bytes,
			COUNT(*) FILTER (WHERE cache_hit) AS cache_hit_count,
			COUNT(*) FILTER (WHERE NOT cache_hit) AS cache_miss_count
		FROM compilation_events
		WHERE started_at >= $1::date
		  AND started_at < $1::date + INTERVAL '1 day'
		GROUP BY language
		ON CONFLICT (language, date) DO UPDATE SET
			compilation_count = EXCLUDED.compilation_count,
			compilation_success_count = EXCLUDED.compilation_success_count,
			compilation_failure_count = EXCLUDED.compilation_failure_count,
			avg_duration_ms = EXCLUDED.avg_duration_ms,
			p50_duration_ms = EXCLUDED.p50_duration_ms,
			p95_duration_ms = EXCLUDED.p95_duration_ms,
			p99_duration_ms = EXCLUDED.p99_duration_ms,
			total_output_bytes = EXCLUDED.total_output_bytes,
			cache_hit_count = EXCLUDED.cache_hit_count,
			cache_miss_count = EXCLUDED.cache_miss_count
	`
	_, err := a.db.ExecContext(ctx, query, date)
	return err
}

// AggregateOrgStatsDaily computes daily organization usage stats
func (a *Aggregator) AggregateOrgStatsDaily(ctx context.Context, date time.Time) error {
	query := `
		INSERT INTO org_stats_daily (
			organization_id, date,
			api_requests, downloads, compilations,
			active_users, modules_created, versions_created
		)
		SELECT
			o.id AS organization_id,
			$1::date AS date,
			0 AS api_requests, -- TODO: integrate with API request tracking
			COALESCE(COUNT(DISTINCT d.id), 0) AS downloads,
			COALESCE(COUNT(DISTINCT c.id), 0) AS compilations,
			COALESCE(COUNT(DISTINCT COALESCE(d.user_id, v.user_id)), 0) AS active_users,
			0 AS modules_created, -- TODO: track module creation events
			0 AS versions_created -- TODO: track version creation events
		FROM organizations o
		LEFT JOIN download_events d ON o.id = d.organization_id
			AND d.downloaded_at >= $1::date
			AND d.downloaded_at < $1::date + INTERVAL '1 day'
		LEFT JOIN module_view_events v ON o.id = v.organization_id
			AND v.viewed_at >= $1::date
			AND v.viewed_at < $1::date + INTERVAL '1 day'
		LEFT JOIN compilation_events c ON EXISTS (
			SELECT 1 FROM download_events de
			WHERE de.organization_id = o.id
			  AND de.module_name = c.module_name
			  AND de.version = c.version
			  AND c.started_at >= $1::date
			  AND c.started_at < $1::date + INTERVAL '1 day'
		)
		GROUP BY o.id
		ON CONFLICT (organization_id, date) DO UPDATE SET
			api_requests = EXCLUDED.api_requests,
			downloads = EXCLUDED.downloads,
			compilations = EXCLUDED.compilations,
			active_users = EXCLUDED.active_users,
			modules_created = EXCLUDED.modules_created,
			versions_created = EXCLUDED.versions_created
	`
	_, err := a.db.ExecContext(ctx, query, date)
	return err
}

// RefreshMaterializedViews refreshes all materialized views
func (a *Aggregator) RefreshMaterializedViews(ctx context.Context) error {
	views := []string{"top_modules_30d", "trending_modules"}
	for _, view := range views {
		query := "REFRESH MATERIALIZED VIEW CONCURRENTLY " + view
		if _, err := a.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

// AggregateAll runs all aggregation jobs for a given date
func (a *Aggregator) AggregateAll(ctx context.Context, date time.Time) error {
	if err := a.AggregateModuleStatsDaily(ctx, date); err != nil {
		return err
	}

	if err := a.AggregateLanguageStatsDaily(ctx, date); err != nil {
		return err
	}

	if err := a.AggregateOrgStatsDaily(ctx, date); err != nil {
		return err
	}

	// Aggregate weekly (if it's Sunday)
	if date.Weekday() == time.Sunday {
		weekStart := date.AddDate(0, 0, -6) // Start of week (7 days ago)
		if err := a.AggregateModuleStatsWeekly(ctx, weekStart); err != nil {
			return err
		}
	}

	// Aggregate monthly (if it's the first day of month)
	if date.Day() == 1 {
		month := time.Date(date.Year(), date.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		if err := a.AggregateModuleStatsMonthly(ctx, month); err != nil {
			return err
		}
	}

	return nil
}
