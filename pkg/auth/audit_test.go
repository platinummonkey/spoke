package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	al := NewAuditLogger()
	if al == nil {
		t.Fatal("NewAuditLogger() returned nil")
	}
}

func TestAuditLogger_LogAction(t *testing.T) {
	al := NewAuditLogger()
	ctx := context.Background()

	tests := []struct {
		name    string
		log     *AuditLog
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid audit log",
			log: &AuditLog{
				Action:       ActionModuleCreate,
				ResourceType: "module",
				ResourceID:   "test-module",
				Status:       StatusSuccess,
			},
			wantErr: false,
		},
		{
			name: "missing action",
			log: &AuditLog{
				ResourceType: "module",
				Status:       StatusSuccess,
			},
			wantErr: true,
			errMsg:  "action is required",
		},
		{
			name: "missing resource type",
			log: &AuditLog{
				Action: ActionModuleCreate,
				Status: StatusSuccess,
			},
			wantErr: true,
			errMsg:  "resource_type is required",
		},
		{
			name: "missing status",
			log: &AuditLog{
				Action:       ActionModuleCreate,
				ResourceType: "module",
			},
			wantErr: true,
			errMsg:  "status is required",
		},
		{
			name: "complete audit log with all fields",
			log: &AuditLog{
				UserID:         ptrInt64(123),
				OrganizationID: ptrInt64(456),
				Action:         ActionVersionPush,
				ResourceType:   "version",
				ResourceID:     "v1.0.0",
				IPAddress:      "192.168.1.1",
				UserAgent:      "terraform/1.0",
				Status:         StatusSuccess,
			},
			wantErr: false,
		},
		{
			name: "audit log with error message",
			log: &AuditLog{
				Action:       ActionAuthFailure,
				ResourceType: "auth",
				Status:       StatusFailure,
				ErrorMessage: "invalid credentials",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			beforeTime := time.Now()
			err := al.LogAction(ctx, tt.log)
			afterTime := time.Now()

			if (err != nil) != tt.wantErr {
				t.Errorf("LogAction() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("LogAction() error message = %v, want %v", err.Error(), tt.errMsg)
			}

			// Check that CreatedAt was set for valid logs
			if !tt.wantErr {
				if tt.log.CreatedAt.IsZero() {
					t.Error("LogAction() did not set CreatedAt")
				}
				if tt.log.CreatedAt.Before(beforeTime) || tt.log.CreatedAt.After(afterTime) {
					t.Errorf("LogAction() CreatedAt = %v, should be between %v and %v",
						tt.log.CreatedAt, beforeTime, afterTime)
				}
			}
		})
	}
}

func TestAuditLogger_LogFromRequest(t *testing.T) {
	al := NewAuditLogger()

	tests := []struct {
		name         string
		setupRequest func() *http.Request
		action       string
		resourceType string
		resourceID   string
		status       string
		err          error
		wantErr      bool
		checkLog     func(t *testing.T, r *http.Request)
	}{
		{
			name: "basic request with no error",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/api/modules/test", nil)
				req.RemoteAddr = "192.168.1.100:12345"
				req.Header.Set("User-Agent", "terraform/1.5")
				return req
			},
			action:       ActionModuleCreate,
			resourceType: "module",
			resourceID:   "test-module",
			status:       StatusSuccess,
			err:          nil,
			wantErr:      false,
			checkLog: func(t *testing.T, r *http.Request) {
				if r.RemoteAddr != "192.168.1.100:12345" {
					t.Errorf("Expected RemoteAddr to be set")
				}
			},
		},
		{
			name: "request with error",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/modules/test", nil)
				req.RemoteAddr = "10.0.0.1:54321"
				return req
			},
			action:       ActionModuleUpdate,
			resourceType: "module",
			resourceID:   "test-module",
			status:       StatusFailure,
			err:          errors.New("database connection failed"),
			wantErr:      false,
		},
		{
			name: "request with X-Forwarded-For header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("DELETE", "/api/modules/test", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.1")
				req.Header.Set("User-Agent", "curl/7.68.0")
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			action:       ActionModuleDelete,
			resourceType: "module",
			resourceID:   "test-module",
			status:       StatusSuccess,
			err:          nil,
			wantErr:      false,
		},
		{
			name: "request with X-Real-IP header",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/api/tokens", nil)
				req.Header.Set("X-Real-IP", "198.51.100.42")
				req.Header.Set("User-Agent", "go-client/1.0")
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			action:       ActionTokenCreate,
			resourceType: "token",
			resourceID:   "tok_123",
			status:       StatusSuccess,
			err:          nil,
			wantErr:      false,
		},
		{
			name: "missing action returns validation error",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/", nil)
			},
			action:       "",
			resourceType: "module",
			resourceID:   "",
			status:       StatusSuccess,
			err:          nil,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			err := al.LogFromRequest(req, tt.action, tt.resourceType, tt.resourceID, tt.status, tt.err)

			if (err != nil) != tt.wantErr {
				t.Errorf("LogFromRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkLog != nil && !tt.wantErr {
				tt.checkLog(t, req)
			}
		})
	}
}

func TestAuditLogger_QueryAuditLogs(t *testing.T) {
	al := NewAuditLogger()
	ctx := context.Background()

	tests := []struct {
		name    string
		filters *AuditLogFilters
		wantErr bool
	}{
		{
			name:    "nil filters",
			filters: nil,
			wantErr: true,
		},
		{
			name: "filters with user ID",
			filters: &AuditLogFilters{
				UserID: ptrInt64(123),
				Limit:  10,
			},
			wantErr: true, // not implemented yet
		},
		{
			name: "filters with organization ID",
			filters: &AuditLogFilters{
				OrganizationID: ptrInt64(456),
				Limit:          20,
			},
			wantErr: true,
		},
		{
			name: "filters with time range",
			filters: &AuditLogFilters{
				StartTime: ptrTime(time.Now().Add(-24 * time.Hour)),
				EndTime:   ptrTime(time.Now()),
				Limit:     100,
			},
			wantErr: true,
		},
		{
			name: "filters with action and status",
			filters: &AuditLogFilters{
				Action:       ActionModuleCreate,
				ResourceType: "module",
				Status:       StatusSuccess,
				Limit:        50,
				Offset:       10,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs, err := al.QueryAuditLogs(ctx, tt.filters)

			if (err != nil) != tt.wantErr {
				t.Errorf("QueryAuditLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// For now, should return nil and error since it's not implemented
			if logs != nil {
				t.Errorf("QueryAuditLogs() returned logs, expected nil (not implemented)")
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() *http.Request
		wantIP  string
	}{
		{
			name: "X-Forwarded-For header present",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "203.0.113.195")
				req.Header.Set("X-Real-IP", "198.51.100.1")
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			wantIP: "203.0.113.195",
		},
		{
			name: "X-Real-IP header present (no X-Forwarded-For)",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Real-IP", "198.51.100.42")
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			wantIP: "198.51.100.42",
		},
		{
			name: "no proxy headers, use RemoteAddr",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.RemoteAddr = "192.168.1.100:54321"
				return req
			},
			wantIP: "192.168.1.100:54321",
		},
		{
			name: "empty X-Forwarded-For should fall back to X-Real-IP",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "")
				req.Header.Set("X-Real-IP", "198.51.100.50")
				req.RemoteAddr = "10.0.0.1:12345"
				return req
			},
			wantIP: "198.51.100.50",
		},
		{
			name: "empty headers should use RemoteAddr",
			setup: func() *http.Request {
				req := httptest.NewRequest("GET", "/", nil)
				req.Header.Set("X-Forwarded-For", "")
				req.Header.Set("X-Real-IP", "")
				req.RemoteAddr = "172.16.0.1:9999"
				return req
			},
			wantIP: "172.16.0.1:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setup()
			gotIP := getClientIP(req)
			if gotIP != tt.wantIP {
				t.Errorf("getClientIP() = %v, want %v", gotIP, tt.wantIP)
			}
		})
	}
}

func TestAuditActionConstants(t *testing.T) {
	// Test that action constants are defined correctly
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"module create", ActionModuleCreate, "module.create"},
		{"module update", ActionModuleUpdate, "module.update"},
		{"module delete", ActionModuleDelete, "module.delete"},
		{"version push", ActionVersionPush, "version.push"},
		{"version delete", ActionVersionDelete, "version.delete"},
		{"token create", ActionTokenCreate, "token.create"},
		{"token revoke", ActionTokenRevoke, "token.revoke"},
		{"user create", ActionUserCreate, "user.create"},
		{"user update", ActionUserUpdate, "user.update"},
		{"user delete", ActionUserDelete, "user.delete"},
		{"org create", ActionOrgCreate, "organization.create"},
		{"org update", ActionOrgUpdate, "organization.update"},
		{"permission grant", ActionPermissionGrant, "permission.grant"},
		{"permission revoke", ActionPermissionRevoke, "permission.revoke"},
		{"auth success", ActionAuthSuccess, "auth.success"},
		{"auth failure", ActionAuthFailure, "auth.failure"},
		{"rate limit exceeded", ActionRateLimitExceeded, "ratelimit.exceeded"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Constant %s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestAuditStatusConstants(t *testing.T) {
	// Test that status constants are defined correctly
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"success", StatusSuccess, "success"},
		{"failure", StatusFailure, "failure"},
		{"denied", StatusDenied, "denied"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Constant %s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// Helper functions for creating pointers
func ptrInt64(i int64) *int64 {
	return &i
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
