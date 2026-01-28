package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGetOverview(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	service := NewService(db)

	// Mock total modules query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM modules").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(100))

	// Mock total versions query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM versions").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(500))

	// Mock downloads query
	mock.ExpectQuery("SELECT SUM\\(CASE WHEN date").
		WillReturnRows(sqlmock.NewRows([]string{"downloads_24h", "downloads_7d", "downloads_30d"}).
			AddRow(1000, 5000, 15000))

	// Mock active users query
	mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\)").
		WillReturnRows(sqlmock.NewRows([]string{"active_24h", "active_7d"}).
			AddRow(50, 200))

	// Mock top language query
	mock.ExpectQuery("SELECT language FROM language_stats_daily").
		WillReturnRows(sqlmock.NewRows([]string{"language"}).AddRow("go"))

	// Mock avg compilation time query
	mock.ExpectQuery("SELECT AVG\\(avg_duration_ms\\)").
		WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(1250.5))

	// Mock cache hit rate query
	mock.ExpectQuery("SELECT SUM\\(cache_hit_count\\)").
		WillReturnRows(sqlmock.NewRows([]string{"rate"}).AddRow(0.75))

	// Execute
	overview, err := service.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview failed: %v", err)
	}

	// Assertions
	if overview.TotalModules != 100 {
		t.Errorf("Expected TotalModules=100, got %d", overview.TotalModules)
	}
	if overview.TotalVersions != 500 {
		t.Errorf("Expected TotalVersions=500, got %d", overview.TotalVersions)
	}
	if overview.TotalDownloads24h != 1000 {
		t.Errorf("Expected TotalDownloads24h=1000, got %d", overview.TotalDownloads24h)
	}
	if overview.TotalDownloads7d != 5000 {
		t.Errorf("Expected TotalDownloads7d=5000, got %d", overview.TotalDownloads7d)
	}
	if overview.TotalDownloads30d != 15000 {
		t.Errorf("Expected TotalDownloads30d=15000, got %d", overview.TotalDownloads30d)
	}
	if overview.ActiveUsers24h != 50 {
		t.Errorf("Expected ActiveUsers24h=50, got %d", overview.ActiveUsers24h)
	}
	if overview.ActiveUsers7d != 200 {
		t.Errorf("Expected ActiveUsers7d=200, got %d", overview.ActiveUsers7d)
	}
	if overview.TopLanguage != "go" {
		t.Errorf("Expected TopLanguage=go, got %s", overview.TopLanguage)
	}
	if overview.AvgCompilationMs != 1250.5 {
		t.Errorf("Expected AvgCompilationMs=1250.5, got %f", overview.AvgCompilationMs)
	}
	if overview.CacheHitRate != 0.75 {
		t.Errorf("Expected CacheHitRate=0.75, got %f", overview.CacheHitRate)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetModuleStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	service := NewService(db)

	moduleName := "user-service"
	period := "30d"

	// Mock aggregate totals query with COALESCE and avg compilation time
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(view_count\\), 0\\), COALESCE\\(SUM\\(download_count\\), 0\\), COALESCE\\(SUM\\(unique_users\\), 0\\), COALESCE\\(AVG\\(avg_compilation_duration_ms\\), 0\\)").
		WillReturnRows(sqlmock.NewRows([]string{"views", "downloads", "users", "avg_compilation_ms"}).
			AddRow(1000, 500, 50, 100))

	// Mock time series query
	mock.ExpectQuery("SELECT date, download_count FROM module_stats_daily").
		WillReturnRows(sqlmock.NewRows([]string{"date", "download_count"}).
			AddRow(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 10).
			AddRow(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), 15))

	// Mock downloads by language query
	mock.ExpectQuery("SELECT language, COUNT\\(\\*\\) FROM download_events").
		WillReturnRows(sqlmock.NewRows([]string{"language", "count"}).
			AddRow("go", 300).
			AddRow("python", 150).
			AddRow("java", 50))

	// Mock popular versions query
	mock.ExpectQuery("SELECT version, COUNT\\(\\*\\) FROM download_events").
		WillReturnRows(sqlmock.NewRows([]string{"version", "count"}).
			AddRow("v1.0.0", 200).
			AddRow("v1.1.0", 150))

	// Mock compilation success rate query
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(compilation_success_count\\)").
		WillReturnRows(sqlmock.NewRows([]string{"rate"}).
			AddRow(0.95))

	// Mock last downloaded at query
	mock.ExpectQuery("SELECT MAX\\(downloaded_at\\) FROM download_events").
		WillReturnRows(sqlmock.NewRows([]string{"max"}).
			AddRow(time.Date(2026, 1, 28, 0, 0, 0, 0, time.UTC)))

	// Execute
	stats, err := service.GetModuleStats(context.Background(), moduleName, period)
	if err != nil {
		t.Fatalf("GetModuleStats failed: %v", err)
	}

	// Assertions
	if stats.ModuleName != moduleName {
		t.Errorf("Expected ModuleName=%s, got %s", moduleName, stats.ModuleName)
	}
	if stats.TotalViews != 1000 {
		t.Errorf("Expected TotalViews=1000, got %d", stats.TotalViews)
	}
	if stats.TotalDownloads != 500 {
		t.Errorf("Expected TotalDownloads=500, got %d", stats.TotalDownloads)
	}
	if stats.UniqueUsers != 50 {
		t.Errorf("Expected UniqueUsers=50, got %d", stats.UniqueUsers)
	}

	// Check time series data
	if len(stats.DownloadsByDay) != 2 {
		t.Errorf("Expected 2 time series points, got %d", len(stats.DownloadsByDay))
	}
	if stats.DownloadsByDay[0].Date != "2026-01-01" {
		t.Errorf("Expected date=2026-01-01, got %s", stats.DownloadsByDay[0].Date)
	}
	if stats.DownloadsByDay[0].Value != 10 {
		t.Errorf("Expected value=10, got %d", stats.DownloadsByDay[0].Value)
	}

	// Check language breakdown
	if len(stats.DownloadsByLanguage) != 3 {
		t.Errorf("Expected 3 languages, got %d", len(stats.DownloadsByLanguage))
	}
	if stats.DownloadsByLanguage["go"] != 300 {
		t.Errorf("Expected go=300, got %d", stats.DownloadsByLanguage["go"])
	}
	if stats.DownloadsByLanguage["python"] != 150 {
		t.Errorf("Expected python=150, got %d", stats.DownloadsByLanguage["python"])
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetPopularModules(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	service := NewService(db)

	// Mock popular modules query with aggregation
	mock.ExpectQuery("SELECT module_name, SUM\\(download_count\\) AS total_downloads, SUM\\(view_count\\) AS total_views, COUNT\\(DISTINCT date\\) AS active_days, AVG\\(download_count\\) AS avg_daily_downloads").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "total_downloads", "total_views", "active_days", "avg_daily_downloads"}).
			AddRow("common-types", 3000, 5000, 30, 100.0).
			AddRow("user-service", 1000, 2000, 28, 35.7))

	// Execute
	modules, err := service.GetPopularModules(context.Background(), "30d", 10)
	if err != nil {
		t.Fatalf("GetPopularModules failed: %v", err)
	}

	// Assertions
	if len(modules) != 2 {
		t.Fatalf("Expected 2 modules, got %d", len(modules))
	}

	// Check first module
	if modules[0].ModuleName != "common-types" {
		t.Errorf("Expected module=common-types, got %s", modules[0].ModuleName)
	}
	if modules[0].TotalDownloads != 3000 {
		t.Errorf("Expected downloads=3000, got %d", modules[0].TotalDownloads)
	}

	// Check second module
	if modules[1].ModuleName != "user-service" {
		t.Errorf("Expected module=user-service, got %s", modules[1].ModuleName)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetTrendingModules(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	service := NewService(db)

	// Mock trending modules query from materialized view
	mock.ExpectQuery("SELECT module_name, current_downloads, previous_downloads, growth_rate").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "current_downloads", "previous_downloads", "growth_rate"}).
			AddRow("auth-service", 80, 50, 0.6).
			AddRow("payment-service", 45, 30, 0.5))

	// Execute
	modules, err := service.GetTrendingModules(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetTrendingModules failed: %v", err)
	}

	// Assertions
	if len(modules) != 2 {
		t.Fatalf("Expected 2 modules, got %d", len(modules))
	}

	// Check first module (highest growth)
	if modules[0].ModuleName != "auth-service" {
		t.Errorf("Expected module=auth-service, got %s", modules[0].ModuleName)
	}
	if modules[0].CurrentDownloads != 80 {
		t.Errorf("Expected current=80, got %d", modules[0].CurrentDownloads)
	}
	if modules[0].PreviousDownloads != 50 {
		t.Errorf("Expected previous=50, got %d", modules[0].PreviousDownloads)
	}
	if modules[0].GrowthRate != 0.6 {
		t.Errorf("Expected growth=0.6, got %f", modules[0].GrowthRate)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
