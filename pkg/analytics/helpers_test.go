package analytics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/platinummonkey/spoke/pkg/auth"
	"github.com/platinummonkey/spoke/pkg/middleware"
)

func TestExtractUserID(t *testing.T) {
	tests := []struct {
		name     string
		authCtx  *auth.AuthContext
		expected *int64
	}{
		{
			name:     "nil auth context",
			authCtx:  nil,
			expected: nil,
		},
		{
			name: "nil user",
			authCtx: &auth.AuthContext{
				User: nil,
			},
			expected: nil,
		},
		{
			name: "valid user",
			authCtx: &auth.AuthContext{
				User: &auth.User{
					ID:       12345,
					Username: "testuser",
				},
			},
			expected: int64Ptr(12345),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authCtx != nil {
				ctx := context.WithValue(req.Context(), middleware.AuthContextKey, tt.authCtx)
				req = req.WithContext(ctx)
			}

			result := ExtractUserID(req)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestExtractOrganizationID(t *testing.T) {
	tests := []struct {
		name     string
		authCtx  *auth.AuthContext
		expected *int64
	}{
		{
			name:     "nil auth context",
			authCtx:  nil,
			expected: nil,
		},
		{
			name: "nil organization",
			authCtx: &auth.AuthContext{
				Organization: nil,
			},
			expected: nil,
		},
		{
			name: "valid organization",
			authCtx: &auth.AuthContext{
				Organization: &auth.Organization{
					ID:   67890,
					Name: "testorg",
				},
			},
			expected: int64Ptr(67890),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authCtx != nil {
				ctx := context.WithValue(req.Context(), middleware.AuthContextKey, tt.authCtx)
				req = req.WithContext(ctx)
			}

			result := ExtractOrganizationID(req)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", *result)
				}
			} else {
				if result == nil {
					t.Errorf("Expected %v, got nil", *tt.expected)
				} else if *result != *tt.expected {
					t.Errorf("Expected %v, got %v", *tt.expected, *result)
				}
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expected: "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3",
			},
			expected: "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For with spaces",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "  10.0.0.1  ",
			},
			expected: "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "192.168.1.1:12345",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.5",
			},
			expected: "10.0.0.5",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			headers:    map[string]string{},
			expected:   "192.168.1.1",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			headers:    map[string]string{},
			expected:   "[2001:db8::1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			result := GetClientIP(req)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetUserAgent(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "standard browser",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  "",
		},
		{
			name:      "custom user agent",
			userAgent: "spoke-cli/1.0.0",
			expected:  "spoke-cli/1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			result := GetUserAgent(req)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetClientSDK(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "spoke-cli",
			userAgent: "spoke-cli/1.0.0",
			expected:  "spoke-cli",
		},
		{
			name:      "spoke-python-sdk",
			userAgent: "spoke-python-sdk/2.1.0",
			expected:  "spoke-python-sdk",
		},
		{
			name:      "spoke-go-sdk",
			userAgent: "spoke-go-sdk/3.2.1",
			expected:  "spoke-go-sdk",
		},
		{
			name:      "non-spoke user agent",
			userAgent: "Mozilla/5.0",
			expected:  "",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  "",
		},
		{
			name:      "spoke prefix without version",
			userAgent: "spoke-cli",
			expected:  "spoke-cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			result := GetClientSDK(req)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetClientVersion(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "spoke-cli with version",
			userAgent: "spoke-cli/1.0.0",
			expected:  "1.0.0",
		},
		{
			name:      "spoke-python-sdk with version",
			userAgent: "spoke-python-sdk/2.1.0",
			expected:  "2.1.0",
		},
		{
			name:      "spoke-go-sdk with version",
			userAgent: "spoke-go-sdk/3.2.1",
			expected:  "3.2.1",
		},
		{
			name:      "non-spoke user agent",
			userAgent: "Mozilla/5.0",
			expected:  "",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  "",
		},
		{
			name:      "spoke prefix without version",
			userAgent: "spoke-cli",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("User-Agent", tt.userAgent)

			result := GetClientVersion(req)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetReferrer(t *testing.T) {
	tests := []struct {
		name     string
		referrer string
		expected string
	}{
		{
			name:     "valid referrer",
			referrer: "https://example.com/page",
			expected: "https://example.com/page",
		},
		{
			name:     "empty referrer",
			referrer: "",
			expected: "",
		},
		{
			name:     "internal referrer",
			referrer: "/internal/page",
			expected: "/internal/page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.referrer != "" {
				req.Header.Set("Referer", tt.referrer)
			}

			result := GetReferrer(req)

			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Helper function to create int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}
