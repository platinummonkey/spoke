package analytics

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCheckHealthAlerts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	alerter := NewAlerter(db)

	// Mock health alerts query
	mock.ExpectQuery("SELECT(.+)FROM modules m(.+)").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "version", "health_score"}).
			AddRow("legacy-module", "v1.0.0", 35.5).
			AddRow("deprecated-module", "v0.9.0", 42.0))

	// Execute
	alerts, err := alerter.CheckHealthAlerts(context.Background(), 50.0)
	if err != nil {
		t.Fatalf("CheckHealthAlerts failed: %v", err)
	}

	// Assertions
	if len(alerts) != 2 {
		t.Fatalf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check first alert (critical)
	if alerts[0].ModuleName != "legacy-module" {
		t.Errorf("Expected module=legacy-module, got %s", alerts[0].ModuleName)
	}
	if alerts[0].HealthScore != 35.5 {
		t.Errorf("Expected score=35.5, got %f", alerts[0].HealthScore)
	}
	if len(alerts[0].Issues) == 0 {
		t.Error("Expected issues to be populated")
	}
	// Health score < 40 should trigger critical issue
	foundCritical := false
	for _, issue := range alerts[0].Issues {
		if issue == "Critical: Immediate attention required" {
			foundCritical = true
			break
		}
	}
	if !foundCritical {
		t.Error("Expected critical issue for score < 40")
	}

	// Check second alert (warning)
	if alerts[1].HealthScore != 42.0 {
		t.Errorf("Expected score=42.0, got %f", alerts[1].HealthScore)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCheckPerformanceAlerts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	alerter := NewAlerter(db)

	// Mock performance alerts query
	mock.ExpectQuery("SELECT language, p95_duration_ms, compilation_count").
		WillReturnRows(sqlmock.NewRows([]string{"language", "p95_duration_ms", "compilation_count"}).
			AddRow("cpp", 8500, 234).
			AddRow("java", 6200, 567))

	// Execute
	alerts, err := alerter.CheckPerformanceAlerts(context.Background(), 5000)
	if err != nil {
		t.Fatalf("CheckPerformanceAlerts failed: %v", err)
	}

	// Assertions
	if len(alerts) != 2 {
		t.Fatalf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check first alert (slowest)
	if alerts[0].Language != "cpp" {
		t.Errorf("Expected language=cpp, got %s", alerts[0].Language)
	}
	if alerts[0].P95DurationMs != 8500 {
		t.Errorf("Expected p95=8500ms, got %d", alerts[0].P95DurationMs)
	}
	if alerts[0].ThresholdMs != 5000 {
		t.Errorf("Expected threshold=5000ms, got %d", alerts[0].ThresholdMs)
	}
	if alerts[0].CompilationCount != 234 {
		t.Errorf("Expected count=234, got %d", alerts[0].CompilationCount)
	}

	// Check second alert
	if alerts[1].Language != "java" {
		t.Errorf("Expected language=java, got %s", alerts[1].Language)
	}
	if alerts[1].P95DurationMs != 6200 {
		t.Errorf("Expected p95=6200ms, got %d", alerts[1].P95DurationMs)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCheckUsageAlerts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	alerter := NewAlerter(db)

	// Mock usage alerts query
	mock.ExpectQuery("SELECT m.name AS module_name(.+)").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "unused_fields", "last_access_days"}).
			AddRow("abandoned-module", 0, 150.0).
			AddRow("old-module", 0, 95.0))

	// Execute
	alerts, err := alerter.CheckUsageAlerts(context.Background(), 90)
	if err != nil {
		t.Fatalf("CheckUsageAlerts failed: %v", err)
	}

	// Assertions
	if len(alerts) != 2 {
		t.Fatalf("Expected 2 alerts, got %d", len(alerts))
	}

	// Check first alert (longest inactive)
	if alerts[0].ModuleName != "abandoned-module" {
		t.Errorf("Expected module=abandoned-module, got %s", alerts[0].ModuleName)
	}
	if alerts[0].LastAccessDays != 150 {
		t.Errorf("Expected days=150, got %d", alerts[0].LastAccessDays)
	}

	// Check second alert
	if alerts[1].ModuleName != "old-module" {
		t.Errorf("Expected module=old-module, got %s", alerts[1].ModuleName)
	}
	if alerts[1].LastAccessDays != 95 {
		t.Errorf("Expected days=95, got %d", alerts[1].LastAccessDays)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCheckHealthAlerts_NoAlerts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	alerter := NewAlerter(db)

	// Mock empty result (all modules healthy)
	mock.ExpectQuery("SELECT(.+)FROM modules m(.+)").
		WillReturnRows(sqlmock.NewRows([]string{"module_name", "version", "health_score"}))

	// Execute
	alerts, err := alerter.CheckHealthAlerts(context.Background(), 50.0)
	if err != nil {
		t.Fatalf("CheckHealthAlerts failed: %v", err)
	}

	// Assertions
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts, got %d", len(alerts))
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCheckPerformanceAlerts_BelowThreshold(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	alerter := NewAlerter(db)

	// Mock empty result (all languages fast)
	mock.ExpectQuery("SELECT language, p95_duration_ms, compilation_count").
		WillReturnRows(sqlmock.NewRows([]string{"language", "p95_duration_ms", "compilation_count"}))

	// Execute
	alerts, err := alerter.CheckPerformanceAlerts(context.Background(), 5000)
	if err != nil {
		t.Fatalf("CheckPerformanceAlerts failed: %v", err)
	}

	// Assertions
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts, got %d", len(alerts))
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
