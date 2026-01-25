package analytics

import (
	"context"
	"database/sql"
	"time"
)

// Service provides analytics business logic
type Service struct {
	db *sql.DB
}

// NewService creates a new analytics service
func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

// OverviewResponse contains high-level KPIs
type OverviewResponse struct {
	TotalModules      int64   `json:"total_modules"`
	TotalVersions     int64   `json:"total_versions"`
	TotalDownloads24h int64   `json:"total_downloads_24h"`
	TotalDownloads7d  int64   `json:"total_downloads_7d"`
	TotalDownloads30d int64   `json:"total_downloads_30d"`
	ActiveUsers24h    int64   `json:"active_users_24h"`
	ActiveUsers7d     int64   `json:"active_users_7d"`
	TopLanguage       string  `json:"top_language"`
	AvgCompilationMs  float64 `json:"avg_compilation_ms"`
	CacheHitRate      float64 `json:"cache_hit_rate"`
}

// GetOverview retrieves high-level KPIs
func (s *Service) GetOverview(ctx context.Context) (*OverviewResponse, error) {
	var overview OverviewResponse

	// Total modules
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM modules").Scan(&overview.TotalModules)
	if err != nil {
		return nil, err
	}

	// Total versions
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM versions").Scan(&overview.TotalVersions)
	if err != nil {
		return nil, err
	}

	// Downloads (24h, 7d, 30d)
	query := `
		SELECT
			SUM(CASE WHEN date >= CURRENT_DATE - INTERVAL '1 day' THEN download_count ELSE 0 END) AS downloads_24h,
			SUM(CASE WHEN date >= CURRENT_DATE - INTERVAL '7 days' THEN download_count ELSE 0 END) AS downloads_7d,
			SUM(CASE WHEN date >= CURRENT_DATE - INTERVAL '30 days' THEN download_count ELSE 0 END) AS downloads_30d
		FROM module_stats_daily
	`
	err = s.db.QueryRowContext(ctx, query).Scan(
		&overview.TotalDownloads24h,
		&overview.TotalDownloads7d,
		&overview.TotalDownloads30d,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Active users
	query = `
		SELECT
			COUNT(DISTINCT user_id) FILTER (WHERE downloaded_at >= NOW() - INTERVAL '24 hours'),
			COUNT(DISTINCT user_id) FILTER (WHERE downloaded_at >= NOW() - INTERVAL '7 days')
		FROM download_events
		WHERE user_id IS NOT NULL
	`
	err = s.db.QueryRowContext(ctx, query).Scan(&overview.ActiveUsers24h, &overview.ActiveUsers7d)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Top language
	query = `
		SELECT language
		FROM language_stats_daily
		WHERE date >= CURRENT_DATE - INTERVAL '30 days'
		GROUP BY language
		ORDER BY SUM(compilation_count) DESC
		LIMIT 1
	`
	err = s.db.QueryRowContext(ctx, query).Scan(&overview.TopLanguage)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Avg compilation time
	query = `
		SELECT AVG(avg_duration_ms)
		FROM language_stats_daily
		WHERE date >= CURRENT_DATE - INTERVAL '7 days'
	`
	err = s.db.QueryRowContext(ctx, query).Scan(&overview.AvgCompilationMs)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Cache hit rate
	query = `
		SELECT
			SUM(cache_hit_count)::float / NULLIF(SUM(cache_hit_count + cache_miss_count), 0)
		FROM language_stats_daily
		WHERE date >= CURRENT_DATE - INTERVAL '7 days'
	`
	var nullableRate sql.NullFloat64
	err = s.db.QueryRowContext(ctx, query).Scan(&nullableRate)
	if err == nil && nullableRate.Valid {
		overview.CacheHitRate = nullableRate.Float64
	}

	return &overview, nil
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Value int64  `json:"value"`
}

// VersionStats contains download stats for a version
type VersionStats struct {
	Version   string `json:"version"`
	Downloads int64  `json:"downloads"`
}

// ModuleStatsResponse contains per-module analytics
type ModuleStatsResponse struct {
	ModuleName             string                   `json:"module_name"`
	TotalViews             int64                    `json:"total_views"`
	TotalDownloads         int64                    `json:"total_downloads"`
	UniqueUsers            int64                    `json:"unique_users"`
	DownloadsByDay         []TimeSeriesPoint        `json:"downloads_by_day"`
	DownloadsByLanguage    map[string]int64         `json:"downloads_by_language"`
	PopularVersions        []VersionStats           `json:"popular_versions"`
	AvgCompilationTimeMs   int                      `json:"avg_compilation_time_ms"`
	CompilationSuccessRate float64                  `json:"compilation_success_rate"`
	LastDownloadedAt       *time.Time               `json:"last_downloaded_at"`
}

// GetModuleStats retrieves per-module analytics
func (s *Service) GetModuleStats(ctx context.Context, moduleName string, period string) (*ModuleStatsResponse, error) {
	// Convert period to days
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}

	var stats ModuleStatsResponse
	stats.ModuleName = moduleName

	// Aggregate totals
	query := `
		SELECT
			COALESCE(SUM(view_count), 0),
			COALESCE(SUM(download_count), 0),
			COALESCE(SUM(unique_users), 0),
			COALESCE(AVG(avg_compilation_duration_ms), 0)
		FROM module_stats_daily
		WHERE module_name = $1
		  AND date >= CURRENT_DATE - $2::integer * INTERVAL '1 day'
	`
	var avgCompilationMs sql.NullInt32
	err := s.db.QueryRowContext(ctx, query, moduleName, days).Scan(
		&stats.TotalViews,
		&stats.TotalDownloads,
		&stats.UniqueUsers,
		&avgCompilationMs,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if avgCompilationMs.Valid {
		stats.AvgCompilationTimeMs = int(avgCompilationMs.Int32)
	}

	// Time series data
	query = `
		SELECT date, download_count
		FROM module_stats_daily
		WHERE module_name = $1
		  AND date >= CURRENT_DATE - $2::integer * INTERVAL '1 day'
		ORDER BY date ASC
	`
	rows, err := s.db.QueryContext(ctx, query, moduleName, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var point TimeSeriesPoint
		var date time.Time
		if err := rows.Scan(&date, &point.Value); err != nil {
			return nil, err
		}
		point.Date = date.Format("2006-01-02")
		stats.DownloadsByDay = append(stats.DownloadsByDay, point)
	}

	// Downloads by language
	query = `
		SELECT language, COUNT(*)
		FROM download_events
		WHERE module_name = $1
		  AND downloaded_at >= NOW() - $2::integer * INTERVAL '1 day'
		GROUP BY language
	`
	rows, err = s.db.QueryContext(ctx, query, moduleName, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.DownloadsByLanguage = make(map[string]int64)
	for rows.Next() {
		var language string
		var count int64
		if err := rows.Scan(&language, &count); err != nil {
			return nil, err
		}
		stats.DownloadsByLanguage[language] = count
	}

	// Popular versions
	query = `
		SELECT version, COUNT(*)
		FROM download_events
		WHERE module_name = $1
		  AND downloaded_at >= NOW() - $2::integer * INTERVAL '1 day'
		GROUP BY version
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`
	rows, err = s.db.QueryContext(ctx, query, moduleName, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var vs VersionStats
		if err := rows.Scan(&vs.Version, &vs.Downloads); err != nil {
			return nil, err
		}
		stats.PopularVersions = append(stats.PopularVersions, vs)
	}

	// Compilation success rate
	query = `
		SELECT
			COALESCE(SUM(compilation_success_count)::float / NULLIF(SUM(compilation_count), 0), 0)
		FROM module_stats_daily
		WHERE module_name = $1
		  AND date >= CURRENT_DATE - $2::integer * INTERVAL '1 day'
	`
	err = s.db.QueryRowContext(ctx, query, moduleName, days).Scan(&stats.CompilationSuccessRate)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Last downloaded at
	query = `
		SELECT MAX(downloaded_at)
		FROM download_events
		WHERE module_name = $1
	`
	var lastDownload sql.NullTime
	err = s.db.QueryRowContext(ctx, query, moduleName).Scan(&lastDownload)
	if err == nil && lastDownload.Valid {
		stats.LastDownloadedAt = &lastDownload.Time
	}

	return &stats, nil
}

// PopularModule represents a popular module entry
type PopularModule struct {
	ModuleName        string  `json:"module_name"`
	TotalDownloads    int64   `json:"total_downloads"`
	TotalViews        int64   `json:"total_views"`
	ActiveDays        int     `json:"active_days"`
	AvgDailyDownloads float64 `json:"avg_daily_downloads"`
}

// GetPopularModules retrieves top modules by downloads
func (s *Service) GetPopularModules(ctx context.Context, period string, limit int) ([]PopularModule, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}

	query := `
		SELECT
			module_name,
			SUM(download_count) AS total_downloads,
			SUM(view_count) AS total_views,
			COUNT(DISTINCT date) AS active_days,
			AVG(download_count) AS avg_daily_downloads
		FROM module_stats_daily
		WHERE date >= CURRENT_DATE - $1::integer * INTERVAL '1 day'
		GROUP BY module_name
		HAVING SUM(download_count) > 0
		ORDER BY total_downloads DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []PopularModule
	for rows.Next() {
		var m PopularModule
		if err := rows.Scan(
			&m.ModuleName,
			&m.TotalDownloads,
			&m.TotalViews,
			&m.ActiveDays,
			&m.AvgDailyDownloads,
		); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}

	return modules, nil
}

// TrendingModule represents a trending module with growth rate
type TrendingModule struct {
	ModuleName        string  `json:"module_name"`
	CurrentDownloads  int64   `json:"current_downloads"`
	PreviousDownloads int64   `json:"previous_downloads"`
	GrowthRate        float64 `json:"growth_rate"`
}

// GetTrendingModules retrieves modules with highest growth rate
func (s *Service) GetTrendingModules(ctx context.Context, limit int) ([]TrendingModule, error) {
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	query := `
		SELECT
			module_name,
			current_downloads,
			previous_downloads,
			growth_rate
		FROM trending_modules
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []TrendingModule
	for rows.Next() {
		var m TrendingModule
		if err := rows.Scan(
			&m.ModuleName,
			&m.CurrentDownloads,
			&m.PreviousDownloads,
			&m.GrowthRate,
		); err != nil {
			return nil, err
		}
		modules = append(modules, m)
	}

	return modules, nil
}
