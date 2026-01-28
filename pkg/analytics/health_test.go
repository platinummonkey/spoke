package analytics

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewHealthScorer(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)
	if scorer == nil {
		t.Fatal("Expected non-nil HealthScorer")
	}
	if scorer.db != db {
		t.Error("Expected db to be set correctly")
	}
}

func TestCalculateHealth(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock getVersionID
	mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	// Mock calculateComplexity
	mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{
			"message_count", "enum_count", "service_count", "field_count", "method_count",
		}).AddRow(10, 5, 2, 50, 8))

	// Mock findUnusedFields
	mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"entity_name"}).
			AddRow("unused_field_1").
			AddRow("unused_field_2"))

	// Mock countDeprecatedFields
	mock.ExpectQuery("SELECT COUNT(.+)FROM proto_search_index(.+)WHERE version_id").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	// Mock countRecentBreakingChanges
	mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE m.name").
		WithArgs("test-module").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Mock countDependents
	mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE v.dependencies").
		WithArgs("test-module@v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Execute
	health, err := scorer.CalculateHealth(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("CalculateHealth failed: %v", err)
	}

	// Assertions
	if health.ModuleName != "test-module" {
		t.Errorf("Expected module name test-module, got %s", health.ModuleName)
	}
	if health.Version != "v1.0.0" {
		t.Errorf("Expected version v1.0.0, got %s", health.Version)
	}
	if health.ComplexityScore == 0 {
		t.Error("Expected non-zero complexity score")
	}
	if len(health.UnusedFields) != 2 {
		t.Errorf("Expected 2 unused fields, got %d", len(health.UnusedFields))
	}
	if health.DeprecatedFieldCount != 3 {
		t.Errorf("Expected 3 deprecated fields, got %d", health.DeprecatedFieldCount)
	}
	if health.BreakingChanges30d != 1 {
		t.Errorf("Expected 1 breaking change, got %d", health.BreakingChanges30d)
	}
	if health.DependentsCount != 5 {
		t.Errorf("Expected 5 dependents, got %d", health.DependentsCount)
	}
	if health.MaintainabilityIndex == 0 {
		t.Error("Expected non-zero maintainability index")
	}
	if health.HealthScore == 0 {
		t.Error("Expected non-zero health score")
	}
	if len(health.Recommendations) == 0 {
		t.Error("Expected recommendations to be populated")
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetVersionID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock successful query
	mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)JOIN modules m(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(42))

	versionID, err := scorer.getVersionID(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("getVersionID failed: %v", err)
	}
	if versionID != 42 {
		t.Errorf("Expected version ID 42, got %d", versionID)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestGetVersionID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock query returning no rows
	mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)JOIN modules m(.+)").
		WithArgs("nonexistent-module", "v1.0.0").
		WillReturnError(sql.ErrNoRows)

	_, err = scorer.getVersionID(context.Background(), "nonexistent-module", "v1.0.0")
	if err != sql.ErrNoRows {
		t.Errorf("Expected sql.ErrNoRows, got %v", err)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCalculateComplexity(t *testing.T) {
	tests := []struct {
		name          string
		messageCount  int
		enumCount     int
		serviceCount  int
		fieldCount    int
		methodCount   int
		expectNonZero bool
	}{
		{
			name:          "simple schema",
			messageCount:  5,
			enumCount:     2,
			serviceCount:  1,
			fieldCount:    20,
			methodCount:   4,
			expectNonZero: true,
		},
		{
			name:          "complex schema",
			messageCount:  50,
			enumCount:     10,
			serviceCount:  5,
			fieldCount:    500,
			methodCount:   30,
			expectNonZero: true,
		},
		{
			name:          "empty schema",
			messageCount:  0,
			enumCount:     0,
			serviceCount:  0,
			fieldCount:    0,
			methodCount:   0,
			expectNonZero: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create mock database: %v", err)
			}
			defer db.Close()

			scorer := NewHealthScorer(db)

			mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
				WithArgs(int64(1)).
				WillReturnRows(sqlmock.NewRows([]string{
					"message_count", "enum_count", "service_count", "field_count", "method_count",
				}).AddRow(tt.messageCount, tt.enumCount, tt.serviceCount, tt.fieldCount, tt.methodCount))

			complexity, err := scorer.calculateComplexity(context.Background(), 1)
			if err != nil {
				t.Fatalf("calculateComplexity failed: %v", err)
			}

			if tt.expectNonZero && complexity == 0 {
				t.Error("Expected non-zero complexity")
			}
			if complexity < 0 || complexity > 100 {
				t.Errorf("Complexity should be 0-100, got %f", complexity)
			}

			// Verify all expectations met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unmet expectations: %v", err)
			}
		})
	}
}

func TestFindUnusedFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock query with multiple unused fields
	mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"entity_name"}).
			AddRow("field1").
			AddRow("field2").
			AddRow("field3"))

	unused, err := scorer.findUnusedFields(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("findUnusedFields failed: %v", err)
	}

	if len(unused) != 3 {
		t.Errorf("Expected 3 unused fields, got %d", len(unused))
	}
	if unused[0] != "field1" || unused[1] != "field2" || unused[2] != "field3" {
		t.Errorf("Expected [field1, field2, field3], got %v", unused)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestFindUnusedFields_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock query with no unused fields
	mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))

	unused, err := scorer.findUnusedFields(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("findUnusedFields failed: %v", err)
	}

	if len(unused) != 0 {
		t.Errorf("Expected 0 unused fields, got %d", len(unused))
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCountDeprecatedFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	mock.ExpectQuery("SELECT COUNT(.+)FROM proto_search_index(.+)WHERE version_id").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	count, err := scorer.countDeprecatedFields(context.Background(), 1)
	if err != nil {
		t.Fatalf("countDeprecatedFields failed: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 deprecated fields, got %d", count)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCountRecentBreakingChanges(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)JOIN modules m(.+)WHERE m.name").
		WithArgs("test-module").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := scorer.countRecentBreakingChanges(context.Background(), "test-module")
	if err != nil {
		t.Fatalf("countRecentBreakingChanges failed: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 breaking changes, got %d", count)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCountDependents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE v.dependencies").
		WithArgs("test-module@v1.0.0").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(12))

	count, err := scorer.countDependents(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("countDependents failed: %v", err)
	}

	if count != 12 {
		t.Errorf("Expected 12 dependents, got %d", count)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCountDependents_NoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE v.dependencies").
		WithArgs("test-module@v1.0.0").
		WillReturnError(sql.ErrNoRows)

	count, err := scorer.countDependents(context.Background(), "test-module", "v1.0.0")
	if err != nil {
		t.Fatalf("countDependents should return 0 on no rows, got error: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 dependents on no rows, got %d", count)
	}

	// Verify all expectations met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestCalculateMaintainability(t *testing.T) {
	tests := []struct {
		name           string
		health         *ModuleHealth
		expectedMin    float64
		expectedMax    float64
		shouldBeHigh   bool
	}{
		{
			name: "high maintainability",
			health: &ModuleHealth{
				ComplexityScore:      10.0,
				UnusedFields:         []string{},
				DeprecatedFieldCount: 0,
				BreakingChanges30d:   0,
			},
			expectedMin:  90,
			expectedMax:  100,
			shouldBeHigh: true,
		},
		{
			name: "low maintainability",
			health: &ModuleHealth{
				ComplexityScore:      90.0,
				UnusedFields:         []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10"},
				DeprecatedFieldCount: 10,
				BreakingChanges30d:   5,
			},
			expectedMin:  0,
			expectedMax:  50,
			shouldBeHigh: false,
		},
		{
			name: "medium maintainability",
			health: &ModuleHealth{
				ComplexityScore:      50.0,
				UnusedFields:         []string{"f1", "f2"},
				DeprecatedFieldCount: 2,
				BreakingChanges30d:   1,
			},
			expectedMin:  40,
			expectedMax:  80,
			shouldBeHigh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, _ := sqlmock.New()
			defer db.Close()

			scorer := NewHealthScorer(db)
			result := scorer.calculateMaintainability(tt.health)

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("Expected maintainability between %f and %f, got %f",
					tt.expectedMin, tt.expectedMax, result)
			}

			if result < 0 || result > 100 {
				t.Errorf("Maintainability should be 0-100, got %f", result)
			}
		})
	}
}

func TestCalculateOverallHealth(t *testing.T) {
	tests := []struct {
		name        string
		health      *ModuleHealth
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "excellent health",
			health: &ModuleHealth{
				ComplexityScore:       10.0,
				MaintainabilityIndex:  95.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    0,
			},
			expectedMin: 85,
			expectedMax: 100,
		},
		{
			name: "poor health",
			health: &ModuleHealth{
				ComplexityScore:       90.0,
				MaintainabilityIndex:  30.0,
				UnusedFields:          []string{"f1", "f2", "f3", "f4", "f5", "f6", "f7", "f8", "f9", "f10"},
				DeprecatedFieldCount:  10,
				BreakingChanges30d:    5,
			},
			expectedMin: 0,
			expectedMax: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, _ := sqlmock.New()
			defer db.Close()

			scorer := NewHealthScorer(db)
			result := scorer.calculateOverallHealth(tt.health)

			if result < tt.expectedMin || result > tt.expectedMax {
				t.Errorf("Expected health score between %f and %f, got %f",
					tt.expectedMin, tt.expectedMax, result)
			}

			if result < 0 || result > 100 {
				t.Errorf("Health score should be 0-100, got %f", result)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	tests := []struct {
		name                   string
		health                 *ModuleHealth
		expectedContains       []string
		expectedMinCount       int
	}{
		{
			name: "high complexity",
			health: &ModuleHealth{
				ComplexityScore:       75.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    0,
				DependentsCount:       0,
				HealthScore:           60.0,
			},
			expectedContains: []string{"splitting this module"},
			expectedMinCount: 1,
		},
		{
			name: "many unused fields",
			health: &ModuleHealth{
				ComplexityScore:       30.0,
				UnusedFields:          []string{"f1", "f2", "f3", "f4", "f5", "f6"},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    0,
				DependentsCount:       0,
				HealthScore:           70.0,
			},
			expectedContains: []string{"Remove unused fields"},
			expectedMinCount: 1,
		},
		{
			name: "many deprecated fields",
			health: &ModuleHealth{
				ComplexityScore:       30.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  5,
				BreakingChanges30d:    0,
				DependentsCount:       0,
				HealthScore:           65.0,
			},
			expectedContains: []string{"Remove deprecated fields"},
			expectedMinCount: 1,
		},
		{
			name: "frequent breaking changes",
			health: &ModuleHealth{
				ComplexityScore:       30.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    3,
				DependentsCount:       0,
				HealthScore:           60.0,
			},
			expectedContains: []string{"Frequent breaking changes"},
			expectedMinCount: 1,
		},
		{
			name: "many dependents with breaking changes",
			health: &ModuleHealth{
				ComplexityScore:       30.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    1,
				DependentsCount:       15,
				HealthScore:           65.0,
			},
			expectedContains: []string{"many dependents"},
			expectedMinCount: 1,
		},
		{
			name: "excellent health",
			health: &ModuleHealth{
				ComplexityScore:       20.0,
				UnusedFields:          []string{},
				DeprecatedFieldCount:  0,
				BreakingChanges30d:    0,
				DependentsCount:       5,
				HealthScore:           85.0,
			},
			expectedContains: []string{"excellent"},
			expectedMinCount: 1,
		},
		{
			name: "poor health",
			health: &ModuleHealth{
				ComplexityScore:       80.0,
				UnusedFields:          []string{"f1", "f2"},
				DeprecatedFieldCount:  2,
				BreakingChanges30d:    1,
				DependentsCount:       3,
				HealthScore:           45.0,
			},
			expectedContains: []string{"needs attention"},
			expectedMinCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, _, _ := sqlmock.New()
			defer db.Close()

			scorer := NewHealthScorer(db)
			recommendations := scorer.generateRecommendations(tt.health)

			if len(recommendations) < tt.expectedMinCount {
				t.Errorf("Expected at least %d recommendations, got %d",
					tt.expectedMinCount, len(recommendations))
			}

			for _, expectedText := range tt.expectedContains {
				found := false
				for _, rec := range recommendations {
					if len(rec) > 0 && containsSubstring(rec, expectedText) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected recommendation containing '%s', got: %v",
						expectedText, recommendations)
				}
			}
		})
	}
}

func TestCalculateHealth_ErrorPaths(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	tests := []struct {
		name          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "getVersionID error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name: "calculateComplexity error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name: "findUnusedFields error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{
						"message_count", "enum_count", "service_count", "field_count", "method_count",
					}).AddRow(10, 5, 2, 50, 8))
				mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name: "countDeprecatedFields error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{
						"message_count", "enum_count", "service_count", "field_count", "method_count",
					}).AddRow(10, 5, 2, 50, 8))
				mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))
				mock.ExpectQuery("SELECT COUNT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name: "countRecentBreakingChanges error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{
						"message_count", "enum_count", "service_count", "field_count", "method_count",
					}).AddRow(10, 5, 2, 50, 8))
				mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))
				mock.ExpectQuery("SELECT COUNT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE m.name").
					WithArgs("test-module").
					WillReturnError(sql.ErrConnDone)
			},
			expectedError: true,
		},
		{
			name: "countDependents error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT v.id(.+)FROM versions v(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{
						"message_count", "enum_count", "service_count", "field_count", "method_count",
					}).AddRow(10, 5, 2, 50, 8))
				mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
					WithArgs("test-module", "v1.0.0").
					WillReturnRows(sqlmock.NewRows([]string{"entity_name"}))
				mock.ExpectQuery("SELECT COUNT(.+)FROM proto_search_index(.+)WHERE version_id").
					WithArgs(1).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
				mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE m.name").
					WithArgs("test-module").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery("SELECT COUNT(.+)FROM versions v(.+)WHERE v.dependencies").
					WithArgs("test-module@v1.0.0").
					WillReturnError(context.DeadlineExceeded)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock(mock)

			_, err := scorer.CalculateHealth(context.Background(), "test-module", "v1.0.0")
			if tt.expectedError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unmet expectations: %v", err)
			}
		})
	}
}

func TestCalculateComplexity_Error(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	mock.ExpectQuery("SELECT(.+)FROM proto_search_index(.+)WHERE version_id").
		WithArgs(int64(1)).
		WillReturnError(context.DeadlineExceeded)

	_, err = scorer.calculateComplexity(context.Background(), 1)
	if err == nil {
		t.Error("Expected error but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

func TestFindUnusedFields_QueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	scorer := NewHealthScorer(db)

	// Mock query returning error
	mock.ExpectQuery("SELECT DISTINCT psi.entity_name(.+)FROM proto_search_index psi(.+)").
		WithArgs("test-module", "v1.0.0").
		WillReturnError(context.DeadlineExceeded)

	_, err = scorer.findUnusedFields(context.Background(), "test-module", "v1.0.0")
	if err == nil {
		t.Error("Expected error from query but got nil")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// Helper function to check if a string contains a substring (case-insensitive)
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		// Simple case-insensitive check
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				match := true
				for j := 0; j < len(substr); j++ {
					c1 := s[i+j]
					c2 := substr[j]
					// Simple ASCII lowercase conversion
					if c1 >= 'A' && c1 <= 'Z' {
						c1 = c1 + 32
					}
					if c2 >= 'A' && c2 <= 'Z' {
						c2 = c2 + 32
					}
					if c1 != c2 {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
			return false
		}())
}
