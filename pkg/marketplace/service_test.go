package marketplace

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := &mockStorage{}
	service := NewService(db, storage)

	assert.NotNil(t, service)
	assert.Equal(t, db, service.db)
	assert.Equal(t, storage, service.storage)
}

func TestValidatePlugin(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name    string
		plugin  *Plugin
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plugin",
			plugin: &Plugin{
				ID:            "test-plugin",
				Name:          "Test Plugin",
				Author:        "Test Author",
				Type:          "language",
				SecurityLevel: "community",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			plugin: &Plugin{
				Name:   "Test Plugin",
				Author: "Test Author",
				Type:   "language",
			},
			wantErr: true,
			errMsg:  "plugin ID is required",
		},
		{
			name: "missing name",
			plugin: &Plugin{
				ID:     "test-plugin",
				Author: "Test Author",
				Type:   "language",
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "missing author",
			plugin: &Plugin{
				ID:   "test-plugin",
				Name: "Test Plugin",
				Type: "language",
			},
			wantErr: true,
			errMsg:  "plugin author is required",
		},
		{
			name: "invalid type",
			plugin: &Plugin{
				ID:     "test-plugin",
				Name:   "Test Plugin",
				Author: "Test Author",
				Type:   "invalid",
			},
			wantErr: true,
			errMsg:  "invalid plugin type",
		},
		{
			name: "invalid security level",
			plugin: &Plugin{
				ID:            "test-plugin",
				Name:          "Test Plugin",
				Author:        "Test Author",
				Type:          "language",
				SecurityLevel: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid security level",
		},
		{
			name: "defaults to community security level",
			plugin: &Plugin{
				ID:     "test-plugin",
				Name:   "Test Plugin",
				Author: "Test Author",
				Type:   "language",
			},
			wantErr: false,
		},
		{
			name: "invalid validator type",
			plugin: &Plugin{
				ID:            "test-plugin",
				Name:          "Test Plugin",
				Author:        "Test Author",
				Type:          "validator",
				SecurityLevel: "verified",
			},
			wantErr: true,
			errMsg:  "invalid plugin type",
		},
		{
			name: "invalid generator type",
			plugin: &Plugin{
				ID:            "test-plugin",
				Name:          "Test Plugin",
				Author:        "Test Author",
				Type:          "generator",
				SecurityLevel: "official",
			},
			wantErr: true,
			errMsg:  "invalid plugin type",
		},
		{
			name: "invalid runner type",
			plugin: &Plugin{
				ID:     "test-plugin",
				Name:   "Test Plugin",
				Author: "Test Author",
				Type:   "runner",
			},
			wantErr: true,
			errMsg:  "invalid plugin type",
		},
		{
			name: "invalid transform type",
			plugin: &Plugin{
				ID:     "test-plugin",
				Name:   "Test Plugin",
				Author: "Test Author",
				Type:   "transform",
			},
			wantErr: true,
			errMsg:  "invalid plugin type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validatePlugin(tt.plugin)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				// Check default security level
				if tt.plugin.SecurityLevel == "" {
					assert.Equal(t, "community", tt.plugin.SecurityLevel)
				}
			}
		})
	}
}

func TestGetPlugin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		pluginID := "test-plugin"
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			pluginID, "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs(pluginID).
			WillReturnRows(rows)

		plugin, err := service.GetPlugin(ctx, pluginID)
		assert.NoError(t, err)
		assert.NotNil(t, plugin)
		assert.Equal(t, pluginID, plugin.ID)
		assert.Equal(t, "Test Plugin", plugin.Name)
		assert.Equal(t, int64(100), plugin.DownloadCount)
	})

	t.Run("not found", func(t *testing.T) {
		pluginID := "nonexistent"

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs(pluginID).
			WillReturnError(sql.ErrNoRows)

		plugin, err := service.GetPlugin(ctx, pluginID)
		assert.Error(t, err)
		assert.Nil(t, plugin)
		assert.Contains(t, err.Error(), "plugin not found")
	})

	t.Run("database error", func(t *testing.T) {
		pluginID := "test-plugin"

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs(pluginID).
			WillReturnError(errors.New("database error"))

		plugin, err := service.GetPlugin(ctx, pluginID)
		assert.Error(t, err)
		assert.Nil(t, plugin)
		assert.Contains(t, err.Error(), "failed to get plugin")
	})
}

func TestCreatePlugin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		plugin := &Plugin{
			ID:            "test-plugin",
			Name:          "Test Plugin",
			Author:        "Test Author",
			Type:          "language",
			SecurityLevel: "community",
			Description:   "A test plugin",
			License:       "MIT",
			Homepage:      "https://example.com",
			Repository:    "https://github.com/test/plugin",
			Enabled:       true,
		}

		mock.ExpectExec("INSERT INTO plugins").
			WithArgs(
				plugin.ID, plugin.Name, plugin.Description, plugin.Author, plugin.License,
				plugin.Homepage, plugin.Repository, plugin.Type, plugin.SecurityLevel,
				plugin.Enabled, sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreatePlugin(ctx, plugin)
		assert.NoError(t, err)
	})

	t.Run("validation error", func(t *testing.T) {
		plugin := &Plugin{
			Name:   "Test Plugin",
			Author: "Test Author",
			Type:   "language",
		}

		err := service.CreatePlugin(ctx, plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin ID is required")
	})

	t.Run("database error", func(t *testing.T) {
		plugin := &Plugin{
			ID:     "test-plugin",
			Name:   "Test Plugin",
			Author: "Test Author",
			Type:   "language",
		}

		mock.ExpectExec("INSERT INTO plugins").
			WithArgs(
				plugin.ID, plugin.Name, plugin.Description, plugin.Author, plugin.License,
				plugin.Homepage, plugin.Repository, plugin.Type, sqlmock.AnyArg(),
				plugin.Enabled, sqlmock.AnyArg(), sqlmock.AnyArg(),
			).
			WillReturnError(errors.New("database error"))

		err := service.CreatePlugin(ctx, plugin)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create plugin")
	})
}

func TestCreatePluginVersion(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		version := &PluginVersion{
			PluginID:    "test-plugin",
			Version:     "1.0.0",
			APIVersion:  "v1",
			ManifestURL: "https://example.com/manifest.json",
			DownloadURL: "https://example.com/plugin.tar.gz",
			Checksum:    "abc123",
			SizeBytes:   1024,
		}

		mock.ExpectExec("INSERT INTO plugin_versions").
			WithArgs(
				version.PluginID, version.Version, version.APIVersion, version.ManifestURL,
				version.DownloadURL, version.Checksum, version.SizeBytes, 0, sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreatePluginVersion(ctx, version)
		assert.NoError(t, err)
	})

	t.Run("database error", func(t *testing.T) {
		version := &PluginVersion{
			PluginID:    "test-plugin",
			Version:     "1.0.0",
			APIVersion:  "v1",
			ManifestURL: "https://example.com/manifest.json",
			DownloadURL: "https://example.com/plugin.tar.gz",
		}

		mock.ExpectExec("INSERT INTO plugin_versions").
			WithArgs(
				version.PluginID, version.Version, version.APIVersion, version.ManifestURL,
				version.DownloadURL, version.Checksum, version.SizeBytes, 0, sqlmock.AnyArg(),
			).
			WillReturnError(errors.New("database error"))

		err := service.CreatePluginVersion(ctx, version)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create plugin version")
	})
}

func TestListPluginVersions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		pluginID := "test-plugin"
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "plugin_id", "version", "api_version", "manifest_url", "download_url",
			"checksum", "size_bytes", "downloads", "created_at",
		}).
			AddRow(1, pluginID, "1.0.0", "v1", "https://example.com/manifest.json", "https://example.com/plugin.tar.gz",
				"abc123", int64(1024), int64(50), now).
			AddRow(2, pluginID, "0.9.0", "v1", "https://example.com/manifest-0.9.json", "https://example.com/plugin-0.9.tar.gz",
				"def456", int64(512), int64(25), now)

		mock.ExpectQuery("SELECT (.+) FROM plugin_versions").
			WithArgs(pluginID).
			WillReturnRows(rows)

		versions, err := service.ListPluginVersions(ctx, pluginID)
		assert.NoError(t, err)
		assert.Len(t, versions, 2)
		assert.Equal(t, "1.0.0", versions[0].Version)
		assert.Equal(t, "0.9.0", versions[1].Version)
	})

	t.Run("database error", func(t *testing.T) {
		pluginID := "test-plugin"

		mock.ExpectQuery("SELECT (.+) FROM plugin_versions").
			WithArgs(pluginID).
			WillReturnError(errors.New("database error"))

		versions, err := service.ListPluginVersions(ctx, pluginID)
		assert.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "failed to query versions")
	})

	t.Run("scan error", func(t *testing.T) {
		pluginID := "test-plugin"
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "plugin_id", "version", "api_version", "manifest_url", "download_url",
			"checksum", "size_bytes", "downloads", "created_at",
		}).
			AddRow(1, pluginID, "1.0.0", "v1", "https://example.com/manifest.json", "https://example.com/plugin.tar.gz",
				"abc123", "invalid", int64(50), now) // invalid size_bytes type

		mock.ExpectQuery("SELECT (.+) FROM plugin_versions").
			WithArgs(pluginID).
			WillReturnRows(rows)

		versions, err := service.ListPluginVersions(ctx, pluginID)
		assert.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "failed to scan version")
	})
}

func TestRecordDownload(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		pluginID := "test-plugin"
		version := "1.0.0"

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE plugins SET download_count").
			WithArgs(pluginID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE plugin_versions SET downloads").
			WithArgs(pluginID, version).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := service.RecordDownload(ctx, pluginID, version)
		assert.NoError(t, err)
	})

	t.Run("begin transaction error", func(t *testing.T) {
		pluginID := "test-plugin"
		version := "1.0.0"

		mock.ExpectBegin().WillReturnError(errors.New("begin error"))

		err := service.RecordDownload(ctx, pluginID, version)
		assert.Error(t, err)
	})

	t.Run("plugin update error", func(t *testing.T) {
		pluginID := "test-plugin"
		version := "1.0.0"

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE plugins SET download_count").
			WithArgs(pluginID).
			WillReturnError(errors.New("update error"))
		mock.ExpectRollback()

		err := service.RecordDownload(ctx, pluginID, version)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update plugin download count")
	})

	t.Run("version update error", func(t *testing.T) {
		pluginID := "test-plugin"
		version := "1.0.0"

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE plugins SET download_count").
			WithArgs(pluginID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("UPDATE plugin_versions SET downloads").
			WithArgs(pluginID, version).
			WillReturnError(errors.New("update error"))
		mock.ExpectRollback()

		err := service.RecordDownload(ctx, pluginID, version)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update version download count")
	})
}

func TestCreateReview(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		review := &PluginReview{
			PluginID: "test-plugin",
			UserID:   "user123",
			Rating:   5,
			Review:   "Great plugin!",
		}

		mock.ExpectExec("INSERT INTO plugin_reviews").
			WithArgs(
				review.PluginID, review.UserID, review.Rating, review.Review,
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				review.Rating, review.Review, sqlmock.AnyArg(),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.CreateReview(ctx, review)
		assert.NoError(t, err)
	})

	t.Run("rating too low", func(t *testing.T) {
		review := &PluginReview{
			PluginID: "test-plugin",
			UserID:   "user123",
			Rating:   0,
			Review:   "Bad plugin",
		}

		err := service.CreateReview(ctx, review)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating must be between 1 and 5")
	})

	t.Run("rating too high", func(t *testing.T) {
		review := &PluginReview{
			PluginID: "test-plugin",
			UserID:   "user123",
			Rating:   6,
			Review:   "Amazing plugin",
		}

		err := service.CreateReview(ctx, review)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rating must be between 1 and 5")
	})

	t.Run("database error", func(t *testing.T) {
		review := &PluginReview{
			PluginID: "test-plugin",
			UserID:   "user123",
			Rating:   5,
			Review:   "Great plugin!",
		}

		mock.ExpectExec("INSERT INTO plugin_reviews").
			WithArgs(
				review.PluginID, review.UserID, review.Rating, review.Review,
				sqlmock.AnyArg(), sqlmock.AnyArg(),
				review.Rating, review.Review, sqlmock.AnyArg(),
			).
			WillReturnError(errors.New("database error"))

		err := service.CreateReview(ctx, review)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create review")
	})
}

func TestListReviews(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		pluginID := "test-plugin"
		limit := 10
		offset := 0
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "plugin_id", "user_id", "rating", "review", "created_at", "updated_at",
		}).
			AddRow(1, pluginID, "user1", 5, "Great!", now, now).
			AddRow(2, pluginID, "user2", 4, "Good", now, now)

		mock.ExpectQuery("SELECT (.+) FROM plugin_reviews").
			WithArgs(pluginID, limit, offset).
			WillReturnRows(rows)

		reviews, err := service.ListReviews(ctx, pluginID, limit, offset)
		assert.NoError(t, err)
		assert.Len(t, reviews, 2)
		assert.Equal(t, "user1", reviews[0].UserID)
		assert.Equal(t, 5, reviews[0].Rating)
	})

	t.Run("database error", func(t *testing.T) {
		pluginID := "test-plugin"
		limit := 10
		offset := 0

		mock.ExpectQuery("SELECT (.+) FROM plugin_reviews").
			WithArgs(pluginID, limit, offset).
			WillReturnError(errors.New("database error"))

		reviews, err := service.ListReviews(ctx, pluginID, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, reviews)
		assert.Contains(t, err.Error(), "failed to query reviews")
	})

	t.Run("scan error", func(t *testing.T) {
		pluginID := "test-plugin"
		limit := 10
		offset := 0
		now := time.Now()

		rows := sqlmock.NewRows([]string{
			"id", "plugin_id", "user_id", "rating", "review", "created_at", "updated_at",
		}).
			AddRow(1, pluginID, "user1", "invalid", "Great!", now, now)

		mock.ExpectQuery("SELECT (.+) FROM plugin_reviews").
			WithArgs(pluginID, limit, offset).
			WillReturnRows(rows)

		reviews, err := service.ListReviews(ctx, pluginID, limit, offset)
		assert.Error(t, err)
		assert.Nil(t, reviews)
		assert.Contains(t, err.Error(), "failed to scan review")
	})
}

func TestListPlugins(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	service := NewService(db, &mockStorage{})
	ctx := context.Background()

	t.Run("basic list with defaults", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Plugins, 1)
		assert.Equal(t, int64(1), resp.Total)
		assert.Equal(t, 20, resp.Limit) // default limit
	})

	t.Run("with type filter", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "validator", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs("validator", 20, 0).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WithArgs("validator").
			WillReturnRows(countRows)

		req := &PluginListRequest{Type: "validator"}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Plugins, 1)
		assert.Equal(t, "validator", resp.Plugins[0].Type)
	})

	t.Run("with security level filter", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "official", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs("official", 20, 0).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WithArgs("official").
			WillReturnRows(countRows)

		req := &PluginListRequest{SecurityLevel: "official"}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "official", resp.Plugins[0].SecurityLevel)
	})

	t.Run("with search filter", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Search Plugin", "A plugin for searching", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs("%search%", "%search%", 20, 0).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{Search: "search"}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Plugins, 1)
	})

	t.Run("with tags filter", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Tagged Plugin", "A tagged plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs("golang", "api", 20, 0).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{Tags: []string{"golang", "api"}}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Plugins, 1)
	})

	t.Run("sort by downloads", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Popular Plugin", "A popular plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(1000),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{SortBy: "downloads"}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int64(1000), resp.Plugins[0].DownloadCount)
	})

	t.Run("sort by rating ascending", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Rated Plugin", "A rated plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(3.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{SortBy: "rating", SortOrder: "asc"}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, float64(3.5), resp.Plugins[0].AvgRating)
	})

	t.Run("custom limit and offset", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(100))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs(50, 10).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{Limit: 50, Offset: 10}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 50, resp.Limit)
		assert.Equal(t, 10, resp.Offset)
		assert.Equal(t, int64(100), resp.Total)
	})

	t.Run("limit exceeds maximum", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs(100, 0).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnRows(countRows)

		req := &PluginListRequest{Limit: 200} // exceeds max of 100
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, 100, resp.Limit) // capped at 100
	})

	t.Run("query error", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WillReturnError(errors.New("database error"))

		req := &PluginListRequest{}
		resp, err := service.ListPlugins(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to query plugins")
	})

	t.Run("count query error", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "community", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WillReturnError(errors.New("count error"))

		req := &PluginListRequest{}
		resp, err := service.ListPlugins(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to count plugins")
	})

	t.Run("combined filters", func(t *testing.T) {
		now := time.Now()
		rows := sqlmock.NewRows([]string{
			"id", "name", "description", "author", "license", "homepage",
			"repository", "type", "security_level", "enabled",
			"created_at", "updated_at", "verified_at", "verified_by", "download_count",
			"latest_version", "avg_rating", "review_count",
		}).AddRow(
			"plugin1", "Test Plugin", "A test plugin", "Test Author", "MIT", "https://example.com",
			"https://github.com/test/plugin", "language", "verified", true,
			now, now, nil, "", int64(100),
			"1.0.0", float64(4.5), int64(10),
		)

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(int64(1))

		mock.ExpectQuery("SELECT (.+) FROM plugins p").
			WithArgs("language", "verified", "%test%", "%test%", 30, 5).
			WillReturnRows(rows)
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM plugins").
			WithArgs("language", "verified").
			WillReturnRows(countRows)

		req := &PluginListRequest{
			Type:          "language",
			SecurityLevel: "verified",
			Search:        "test",
			Limit:         30,
			Offset:        5,
		}
		resp, err := service.ListPlugins(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Plugins, 1)
	})
}

func TestPluginListRequest_Defaults(t *testing.T) {
	req := &PluginListRequest{}

	assert.Equal(t, 0, req.Limit)
	assert.Equal(t, 0, req.Offset)
	assert.Equal(t, "", req.SortBy)
	assert.Equal(t, "", req.SortOrder)
}

func TestPluginReview_Validation(t *testing.T) {
	review := &PluginReview{
		PluginID: "test-plugin",
		UserID:   "user123",
		Rating:   5,
		Review:   "Great plugin!",
	}

	assert.Equal(t, "test-plugin", review.PluginID)
	assert.Equal(t, "user123", review.UserID)
	assert.Equal(t, 5, review.Rating)
	assert.Equal(t, "Great plugin!", review.Review)
}

// mockStorage is a mock implementation of Storage interface
type mockStorage struct{}

func (m *mockStorage) StorePluginArchive(ctx context.Context, pluginID, version string, data io.Reader) (string, error) {
	return "", nil
}

func (m *mockStorage) GetPluginArchive(ctx context.Context, pluginID, version string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockStorage) DeletePluginArchive(ctx context.Context, pluginID, version string) error {
	return nil
}

func (m *mockStorage) StorePluginManifest(ctx context.Context, pluginID, version string, data []byte) (string, error) {
	return "", nil
}

func (m *mockStorage) GetPluginManifest(ctx context.Context, pluginID, version string) ([]byte, error) {
	return nil, nil
}

func (m *mockStorage) ListPluginVersions(ctx context.Context, pluginID string) ([]string, error) {
	return nil, nil
}

func (m *mockStorage) GetArchiveChecksum(data io.Reader) (string, error) {
	return "", nil
}
