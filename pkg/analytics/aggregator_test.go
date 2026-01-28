package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewAggregator(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	if aggregator == nil {
		t.Error("Expected aggregator to be non-nil")
	}
	if aggregator.db != db {
		t.Error("Expected aggregator.db to match provided database")
	}
}

func TestAggregateModuleStatsDaily(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Expect the INSERT query to be executed
	mock.ExpectExec("INSERT INTO module_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateModuleStatsDaily(ctx, date)
	if err != nil {
		t.Fatalf("AggregateModuleStatsDaily failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateModuleStatsWeekly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	weekStart := time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Expect the INSERT query to be executed with weekStart and weekEnd
	mock.ExpectExec("INSERT INTO module_stats_weekly").
		WithArgs(weekStart, weekEnd).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateModuleStatsWeekly(ctx, weekStart)
	if err != nil {
		t.Fatalf("AggregateModuleStatsWeekly failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateModuleStatsMonthly(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	month := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	nextMonth := month.AddDate(0, 1, 0)

	// Expect the INSERT query to be executed
	mock.ExpectExec("INSERT INTO module_stats_monthly").
		WithArgs(month, nextMonth).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateModuleStatsMonthly(ctx, month)
	if err != nil {
		t.Fatalf("AggregateModuleStatsMonthly failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateLanguageStatsDaily(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Expect the INSERT query for language stats
	mock.ExpectExec("INSERT INTO language_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateLanguageStatsDaily(ctx, date)
	if err != nil {
		t.Fatalf("AggregateLanguageStatsDaily failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateOrgStatsDaily(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	date := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Expect the INSERT query for org stats
	mock.ExpectExec("INSERT INTO org_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateOrgStatsDaily(ctx, date)
	if err != nil {
		t.Fatalf("AggregateOrgStatsDaily failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestRefreshMaterializedViews(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()

	// Expect refresh for both materialized views
	mock.ExpectExec("REFRESH MATERIALIZED VIEW CONCURRENTLY top_modules_30d").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("REFRESH MATERIALIZED VIEW CONCURRENTLY trending_modules").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = aggregator.RefreshMaterializedViews(ctx)
	if err != nil {
		t.Fatalf("RefreshMaterializedViews failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateAll_RegularDay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	// Tuesday, Jan 14, 2026 - not Sunday, not first of month
	date := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)

	// Expect daily aggregations only
	mock.ExpectExec("INSERT INTO module_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO language_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO org_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateAll(ctx, date)
	if err != nil {
		t.Fatalf("AggregateAll failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateAll_Sunday(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	// Sunday, Jan 18, 2026
	date := time.Date(2026, 1, 18, 0, 0, 0, 0, time.UTC)
	weekStart := date.AddDate(0, 0, -6)
	weekEnd := weekStart.AddDate(0, 0, 7)

	// Expect daily aggregations
	mock.ExpectExec("INSERT INTO module_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO language_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO org_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect weekly aggregation
	mock.ExpectExec("INSERT INTO module_stats_weekly").
		WithArgs(weekStart, weekEnd).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateAll(ctx, date)
	if err != nil {
		t.Fatalf("AggregateAll failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestAggregateAll_FirstOfMonth(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	aggregator := NewAggregator(db)
	ctx := context.Background()
	// Monday, June 1, 2026 - first of month (not Sunday)
	date := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	month := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	nextMonth := month.AddDate(0, 1, 0)

	// Expect daily aggregations
	mock.ExpectExec("INSERT INTO module_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO language_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO org_stats_daily").
		WithArgs(date).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect monthly aggregation for previous month
	mock.ExpectExec("INSERT INTO module_stats_monthly").
		WithArgs(month, nextMonth).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = aggregator.AggregateAll(ctx, date)
	if err != nil {
		t.Fatalf("AggregateAll failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
