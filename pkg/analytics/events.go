package analytics

import (
	"context"
	"database/sql"
	"time"
)

// EventTracker handles analytics event collection
type EventTracker struct {
	db *sql.DB
}

// NewEventTracker creates a new event tracker
func NewEventTracker(db *sql.DB) *EventTracker {
	return &EventTracker{db: db}
}

// DownloadEvent represents a module download
type DownloadEvent struct {
	UserID         *int64
	OrganizationID *int64
	ModuleName     string
	Version        string
	Language       string
	FileSize       int64
	Duration       time.Duration
	Success        bool
	ErrorMessage   string
	IPAddress      string
	UserAgent      string
	ClientSDK      string
	ClientVersion  string
	CacheHit       bool
}

// TrackDownload records a download event
func (t *EventTracker) TrackDownload(ctx context.Context, event DownloadEvent) error {
	query := `
		INSERT INTO download_events (
			user_id, organization_id, module_name, version, language,
			file_size, duration_ms, success, error_message,
			ip_address, user_agent, client_sdk, client_version, cache_hit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := t.db.ExecContext(ctx, query,
		event.UserID, event.OrganizationID, event.ModuleName, event.Version,
		event.Language, event.FileSize, event.Duration.Milliseconds(),
		event.Success, nullString(event.ErrorMessage), nullString(event.IPAddress),
		nullString(event.UserAgent), nullString(event.ClientSDK),
		nullString(event.ClientVersion), event.CacheHit,
	)
	return err
}

// ModuleViewEvent represents a module page view
type ModuleViewEvent struct {
	UserID         *int64
	OrganizationID *int64
	ModuleName     string
	Version        string
	Source         string // 'web', 'api', 'cli'
	PageType       string // 'list', 'detail', 'search'
	Referrer       string
	IPAddress      string
	UserAgent      string
}

// TrackModuleView records a module view event
func (t *EventTracker) TrackModuleView(ctx context.Context, event ModuleViewEvent) error {
	query := `
		INSERT INTO module_view_events (
			user_id, organization_id, module_name, version, source,
			page_type, referrer, ip_address, user_agent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := t.db.ExecContext(ctx, query,
		event.UserID, event.OrganizationID, event.ModuleName,
		nullString(event.Version), event.Source, nullString(event.PageType),
		nullString(event.Referrer), nullString(event.IPAddress),
		nullString(event.UserAgent),
	)
	return err
}

// CompilationEvent represents a compilation job
type CompilationEvent struct {
	ModuleName      string
	Version         string
	Language        string
	StartedAt       time.Time
	CompletedAt     *time.Time
	Duration        time.Duration
	Success         bool
	ErrorMessage    string
	ErrorType       string
	CacheHit        bool
	FileCount       int
	OutputSize      int64
	CompilerVersion string
}

// TrackCompilation records a compilation event
func (t *EventTracker) TrackCompilation(ctx context.Context, event CompilationEvent) error {
	query := `
		INSERT INTO compilation_events (
			module_name, version, language, started_at, completed_at,
			duration_ms, success, error_message, error_type, cache_hit,
			file_count, output_size, compiler_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	var completedAt *time.Time
	if event.CompletedAt != nil {
		completedAt = event.CompletedAt
	}

	_, err := t.db.ExecContext(ctx, query,
		event.ModuleName, event.Version, event.Language, event.StartedAt,
		completedAt, event.Duration.Milliseconds(), event.Success,
		nullString(event.ErrorMessage), nullString(event.ErrorType), event.CacheHit,
		event.FileCount, event.OutputSize, nullString(event.CompilerVersion),
	)
	return err
}

// Helper function to convert empty strings to NULL
func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
