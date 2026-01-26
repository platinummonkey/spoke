package marketplace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
