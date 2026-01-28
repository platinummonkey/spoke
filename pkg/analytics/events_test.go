package analytics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewEventTracker(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)
	if tracker == nil {
		t.Fatal("Expected non-nil EventTracker")
	}
	if tracker.db != db {
		t.Error("Expected EventTracker to store the database reference")
	}
}

func TestTrackDownload(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	tests := []struct {
		name  string
		event DownloadEvent
	}{
		{
			name: "successful download with all fields",
			event: DownloadEvent{
				UserID:         intPtr(123),
				OrganizationID: intPtr(456),
				ModuleName:     "test-module",
				Version:        "v1.0.0",
				Language:       "go",
				FileSize:       1024,
				Duration:       100 * time.Millisecond,
				Success:        true,
				ErrorMessage:   "",
				IPAddress:      "192.168.1.1",
				UserAgent:      "test-agent",
				ClientSDK:      "go-sdk",
				ClientVersion:  "v1.0.0",
				CacheHit:       false,
			},
		},
		{
			name: "failed download with error",
			event: DownloadEvent{
				UserID:         nil,
				OrganizationID: nil,
				ModuleName:     "another-module",
				Version:        "v2.0.0",
				Language:       "python",
				FileSize:       2048,
				Duration:       200 * time.Millisecond,
				Success:        false,
				ErrorMessage:   "network timeout",
				IPAddress:      "10.0.0.1",
				UserAgent:      "python-client",
				ClientSDK:      "python-sdk",
				ClientVersion:  "v2.1.0",
				CacheHit:       true,
			},
		},
		{
			name: "download with empty strings",
			event: DownloadEvent{
				UserID:         intPtr(789),
				OrganizationID: intPtr(101),
				ModuleName:     "minimal-module",
				Version:        "v0.1.0",
				Language:       "java",
				FileSize:       512,
				Duration:       50 * time.Millisecond,
				Success:        true,
				ErrorMessage:   "",
				IPAddress:      "",
				UserAgent:      "",
				ClientSDK:      "",
				ClientVersion:  "",
				CacheHit:       false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectExec("INSERT INTO download_events").
				WithArgs(
					tt.event.UserID,
					tt.event.OrganizationID,
					tt.event.ModuleName,
					tt.event.Version,
					tt.event.Language,
					tt.event.FileSize,
					tt.event.Duration.Milliseconds(),
					tt.event.Success,
					sqlmock.AnyArg(), // nullString(ErrorMessage)
					sqlmock.AnyArg(), // nullString(IPAddress)
					sqlmock.AnyArg(), // nullString(UserAgent)
					sqlmock.AnyArg(), // nullString(ClientSDK)
					sqlmock.AnyArg(), // nullString(ClientVersion)
					tt.event.CacheHit,
				).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := tracker.TrackDownload(context.Background(), tt.event)
			if err != nil {
				t.Errorf("TrackDownload failed: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unmet expectations: %v", err)
			}
		})
	}
}

func TestTrackModuleView(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	tests := []struct {
		name  string
		event ModuleViewEvent
	}{
		{
			name: "web view with all fields",
			event: ModuleViewEvent{
				UserID:         intPtr(100),
				OrganizationID: intPtr(200),
				ModuleName:     "web-module",
				Version:        "v1.0.0",
				Source:         "web",
				PageType:       "detail",
				Referrer:       "https://example.com",
				IPAddress:      "192.168.1.100",
				UserAgent:      "Mozilla/5.0",
			},
		},
		{
			name: "api view without user",
			event: ModuleViewEvent{
				UserID:         nil,
				OrganizationID: nil,
				ModuleName:     "api-module",
				Version:        "v2.0.0",
				Source:         "api",
				PageType:       "list",
				Referrer:       "",
				IPAddress:      "10.0.0.2",
				UserAgent:      "curl/7.64.1",
			},
		},
		{
			name: "cli view with minimal data",
			event: ModuleViewEvent{
				UserID:         intPtr(300),
				OrganizationID: intPtr(400),
				ModuleName:     "cli-module",
				Version:        "",
				Source:         "cli",
				PageType:       "",
				Referrer:       "",
				IPAddress:      "",
				UserAgent:      "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectExec("INSERT INTO module_view_events").
				WithArgs(
					tt.event.UserID,
					tt.event.OrganizationID,
					tt.event.ModuleName,
					sqlmock.AnyArg(), // nullString(Version)
					tt.event.Source,
					sqlmock.AnyArg(), // nullString(PageType)
					sqlmock.AnyArg(), // nullString(Referrer)
					sqlmock.AnyArg(), // nullString(IPAddress)
					sqlmock.AnyArg(), // nullString(UserAgent)
				).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := tracker.TrackModuleView(context.Background(), tt.event)
			if err != nil {
				t.Errorf("TrackModuleView failed: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unmet expectations: %v", err)
			}
		})
	}
}

func TestTrackCompilation(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	now := time.Now()
	completed := now.Add(5 * time.Second)

	tests := []struct {
		name  string
		event CompilationEvent
	}{
		{
			name: "successful compilation",
			event: CompilationEvent{
				ModuleName:      "compile-module",
				Version:         "v1.0.0",
				Language:        "go",
				StartedAt:       now,
				CompletedAt:     &completed,
				Duration:        5 * time.Second,
				Success:         true,
				ErrorMessage:    "",
				ErrorType:       "",
				CacheHit:        false,
				FileCount:       10,
				OutputSize:      4096,
				CompilerVersion: "go1.20.0",
			},
		},
		{
			name: "failed compilation with error",
			event: CompilationEvent{
				ModuleName:      "failed-module",
				Version:         "v2.0.0",
				Language:        "python",
				StartedAt:       now,
				CompletedAt:     nil,
				Duration:        2 * time.Second,
				Success:         false,
				ErrorMessage:    "syntax error",
				ErrorType:       "compilation_error",
				CacheHit:        false,
				FileCount:       5,
				OutputSize:      0,
				CompilerVersion: "python3.9",
			},
		},
		{
			name: "cached compilation",
			event: CompilationEvent{
				ModuleName:      "cached-module",
				Version:         "v3.0.0",
				Language:        "java",
				StartedAt:       now,
				CompletedAt:     &completed,
				Duration:        100 * time.Millisecond,
				Success:         true,
				ErrorMessage:    "",
				ErrorType:       "",
				CacheHit:        true,
				FileCount:       20,
				OutputSize:      8192,
				CompilerVersion: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock.ExpectExec("INSERT INTO compilation_events").
				WithArgs(
					tt.event.ModuleName,
					tt.event.Version,
					tt.event.Language,
					tt.event.StartedAt,
					sqlmock.AnyArg(), // CompletedAt (may be nil)
					tt.event.Duration.Milliseconds(),
					tt.event.Success,
					sqlmock.AnyArg(), // nullString(ErrorMessage)
					sqlmock.AnyArg(), // nullString(ErrorType)
					tt.event.CacheHit,
					tt.event.FileCount,
					tt.event.OutputSize,
					sqlmock.AnyArg(), // nullString(CompilerVersion)
				).
				WillReturnResult(sqlmock.NewResult(1, 1))

			err := tracker.TrackCompilation(context.Background(), tt.event)
			if err != nil {
				t.Errorf("TrackCompilation failed: %v", err)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unmet expectations: %v", err)
			}
		})
	}
}

func TestNullString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected interface{}
	}{
		{
			name:     "empty string returns nil",
			input:    "",
			expected: nil,
		},
		{
			name:     "non-empty string returns string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "whitespace string returns string",
			input:    " ",
			expected: " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTrackDownloadError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	event := DownloadEvent{
		ModuleName: "error-module",
		Version:    "v1.0.0",
		Language:   "go",
		FileSize:   1024,
		Duration:   100 * time.Millisecond,
		Success:    true,
		CacheHit:   false,
	}

	mock.ExpectExec("INSERT INTO download_events").
		WillReturnError(sql.ErrConnDone)

	err = tracker.TrackDownload(context.Background(), event)
	if err == nil {
		t.Error("Expected error from TrackDownload, got nil")
	}
	if err != sql.ErrConnDone {
		t.Errorf("Expected sql.ErrConnDone, got %v", err)
	}
}

func TestTrackModuleViewError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	event := ModuleViewEvent{
		ModuleName: "error-module",
		Version:    "v1.0.0",
		Source:     "web",
	}

	mock.ExpectExec("INSERT INTO module_view_events").
		WillReturnError(sql.ErrConnDone)

	err = tracker.TrackModuleView(context.Background(), event)
	if err == nil {
		t.Error("Expected error from TrackModuleView, got nil")
	}
	if err != sql.ErrConnDone {
		t.Errorf("Expected sql.ErrConnDone, got %v", err)
	}
}

func TestTrackCompilationError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	tracker := NewEventTracker(db)

	event := CompilationEvent{
		ModuleName: "error-module",
		Version:    "v1.0.0",
		Language:   "go",
		StartedAt:  time.Now(),
		Duration:   1 * time.Second,
		Success:    true,
		CacheHit:   false,
		FileCount:  5,
		OutputSize: 1024,
	}

	mock.ExpectExec("INSERT INTO compilation_events").
		WillReturnError(sql.ErrConnDone)

	err = tracker.TrackCompilation(context.Background(), event)
	if err == nil {
		t.Error("Expected error from TrackCompilation, got nil")
	}
	if err != sql.ErrConnDone {
		t.Errorf("Expected sql.ErrConnDone, got %v", err)
	}
}

// Helper function for creating int64 pointers
func intPtr(i int64) *int64 {
	return &i
}
