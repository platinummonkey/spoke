// Package audit provides comprehensive audit logging for security, compliance, and forensics.
//
// # Overview
//
// This package tracks all authentication events, authorization checks, data mutations,
// configuration changes, and admin actions with before/after values and request context.
//
// # Event Types
//
// Authentication: login, logout, password_change, token_create
// Authorization: permission_check, access_denied
// Data: module_create, module_update, module_delete, version_publish
// Admin: org_create, user_invite, role_assign
// Access: module_read, version_download
//
// # Usage Example
//
// Log authentication:
//
//	logger.LogAuthentication(ctx, &audit.AuthEvent{
//		UserID:    user.ID,
//		TokenID:   token.ID,
//		IPAddress: r.RemoteAddr,
//		Success:   true,
//	})
//
// Log data mutation with before/after:
//
//	logger.LogDataMutation(ctx, &audit.DataMutationEvent{
//		ResourceType: audit.ResourceTypeModule,
//		ResourceID:   module.Name,
//		Action:       audit.ActionUpdate,
//		Changes: &audit.ChangeDetails{
//			Before: oldModule,
//			After:  newModule,
//		},
//	})
//
// Search audit logs:
//
//	results, err := logger.Search(ctx, &audit.SearchFilter{
//		StartTime:   time.Now().Add(-24 * time.Hour),
//		EndTime:     time.Now(),
//		UserID:      &userID,
//		EventTypes:  []audit.EventType{audit.EventTypeAuthLogin},
//		Status:      audit.EventStatusFailure,
//	})
//
// # Retention Policy
//
// Default: 90 days active retention
// Archiving: Compress and move to long-term storage
// Export: JSON, CSV, NDJSON formats for external analysis
//
// # Related Packages
//
//   - pkg/auth: Authentication events
//   - pkg/rbac: Authorization events
//   - pkg/middleware: HTTP request logging
package audit
