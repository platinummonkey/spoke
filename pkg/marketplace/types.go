package marketplace

import (
	"time"
)

// Plugin represents a plugin in the marketplace
type Plugin struct {
	ID             string    `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	Author         string    `json:"author" db:"author"`
	License        string    `json:"license" db:"license"`
	Homepage       string    `json:"homepage" db:"homepage"`
	Repository     string    `json:"repository" db:"repository"`
	Type           string    `json:"type" db:"type"`
	SecurityLevel  string    `json:"security_level" db:"security_level"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
	VerifiedAt     *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	VerifiedBy     string    `json:"verified_by,omitempty" db:"verified_by"`
	DownloadCount  int64     `json:"download_count" db:"download_count"`
	LatestVersion  string    `json:"latest_version,omitempty"`
	AvgRating      float64   `json:"avg_rating,omitempty"`
	ReviewCount    int64     `json:"review_count,omitempty"`
}

// PluginVersion represents a specific version of a plugin
type PluginVersion struct {
	ID          int64     `json:"id" db:"id"`
	PluginID    string    `json:"plugin_id" db:"plugin_id"`
	Version     string    `json:"version" db:"version"`
	APIVersion  string    `json:"api_version" db:"api_version"`
	ManifestURL string    `json:"manifest_url" db:"manifest_url"`
	DownloadURL string    `json:"download_url" db:"download_url"`
	Checksum    string    `json:"checksum" db:"checksum"`
	SizeBytes   int64     `json:"size_bytes" db:"size_bytes"`
	Downloads   int64     `json:"downloads" db:"downloads"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PluginReview represents a user review and rating
type PluginReview struct {
	ID        int64     `json:"id" db:"id"`
	PluginID  string    `json:"plugin_id" db:"plugin_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Rating    int       `json:"rating" db:"rating"`
	Review    string    `json:"review" db:"review"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	UserName  string    `json:"user_name,omitempty"` // Not in DB, joined
}

// PluginInstallation represents a plugin installation
type PluginInstallation struct {
	ID             int64      `json:"id" db:"id"`
	PluginID       string     `json:"plugin_id" db:"plugin_id"`
	Version        string     `json:"version" db:"version"`
	UserID         string     `json:"user_id" db:"user_id"`
	OrganizationID string     `json:"organization_id" db:"organization_id"`
	InstalledAt    time.Time  `json:"installed_at" db:"installed_at"`
	UninstalledAt  *time.Time `json:"uninstalled_at,omitempty" db:"uninstalled_at"`
}

// PluginStatDaily represents daily aggregated statistics
type PluginStatDaily struct {
	ID                  int64     `json:"id" db:"id"`
	PluginID            string    `json:"plugin_id" db:"plugin_id"`
	Date                time.Time `json:"date" db:"date"`
	Downloads           int64     `json:"downloads" db:"downloads"`
	Installations       int64     `json:"installations" db:"installations"`
	Uninstallations     int64     `json:"uninstallations" db:"uninstallations"`
	ActiveInstallations int64     `json:"active_installations" db:"active_installations"`
	AvgRating           float64   `json:"avg_rating" db:"avg_rating"`
	ReviewCount         int64     `json:"review_count" db:"review_count"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// PluginDependency represents a dependency between plugins
type PluginDependency struct {
	ID                 int64     `json:"id" db:"id"`
	PluginID           string    `json:"plugin_id" db:"plugin_id"`
	Version            string    `json:"version" db:"version"`
	DependsOnPluginID  string    `json:"depends_on_plugin_id" db:"depends_on_plugin_id"`
	DependsOnVersion   string    `json:"depends_on_version" db:"depends_on_version"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// PluginTag represents a tag for plugin categorization
type PluginTag struct {
	ID        int64     `json:"id" db:"id"`
	PluginID  string    `json:"plugin_id" db:"plugin_id"`
	Tag       string    `json:"tag" db:"tag"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PluginListRequest represents a request to list plugins
type PluginListRequest struct {
	Type          string   `json:"type"`
	SecurityLevel string   `json:"security_level"`
	Tags          []string `json:"tags"`
	Search        string   `json:"search"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
	SortBy        string   `json:"sort_by"` // downloads, rating, created_at
	SortOrder     string   `json:"sort_order"` // asc, desc
}

// PluginListResponse represents the response for listing plugins
type PluginListResponse struct {
	Plugins    []Plugin `json:"plugins"`
	Total      int64    `json:"total"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}

// PluginSubmitRequest represents a request to submit a new plugin
type PluginSubmitRequest struct {
	ID           string   `json:"id" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	Author       string   `json:"author" binding:"required"`
	License      string   `json:"license"`
	Homepage     string   `json:"homepage"`
	Repository   string   `json:"repository"`
	Type         string   `json:"type" binding:"required"`
	Version      string   `json:"version" binding:"required"`
	APIVersion   string   `json:"api_version" binding:"required"`
	ArchiveData  []byte   `json:"archive_data" binding:"required"` // Base64 encoded
	Tags         []string `json:"tags"`
}

// PluginReviewRequest represents a request to submit a review
type PluginReviewRequest struct {
	Rating int    `json:"rating" binding:"required,min=1,max=5"`
	Review string `json:"review"`
}

// PluginInstallRequest represents a request to record an installation
type PluginInstallRequest struct {
	Version        string `json:"version" binding:"required"`
	OrganizationID string `json:"organization_id"`
}

// PluginSearchResult represents a plugin in search results
type PluginSearchResult struct {
	Plugin
	Snippet     string   `json:"snippet"` // Highlighted search result
	MatchScore  float64  `json:"match_score"`
	Tags        []string `json:"tags"`
}

// TrendingPlugin represents a trending plugin
type TrendingPlugin struct {
	Plugin
	GrowthRate       float64 `json:"growth_rate"`
	WeeklyDownloads  int64   `json:"weekly_downloads"`
}

// PluginStats represents statistics for a plugin
type PluginStats struct {
	PluginID            string             `json:"plugin_id"`
	TotalDownloads      int64              `json:"total_downloads"`
	TotalInstallations  int64              `json:"total_installations"`
	ActiveInstallations int64              `json:"active_installations"`
	AvgRating           float64            `json:"avg_rating"`
	ReviewCount         int64              `json:"review_count"`
	DailyStats          []PluginStatDaily  `json:"daily_stats,omitempty"`
	TopVersions         []VersionDownloads `json:"top_versions,omitempty"`
}

// VersionDownloads represents download stats for a version
type VersionDownloads struct {
	Version   string `json:"version"`
	Downloads int64  `json:"downloads"`
}
